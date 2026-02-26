package cmd

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

// logWorkflowStepPattern matches workflow step target labels like "steps[0]", "steps[1]", etc.
var logWorkflowStepPattern = regexp.MustCompile(`^steps\[\d+\]$`)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Query build logs",
}

var logGetCmd = &cobra.Command{
	Use:   "get <invocation-id>",
	Short: "Get build log for an invocation",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogGet,
}

var logTailCmd = &cobra.Command{
	Use:   "tail <invocation-id>",
	Short: "Follow build log output in real-time",
	Long: `Follow build log output by polling GetLog for an in-progress invocation.

Polls the log contents periodically and shows only new content since the last poll.
Automatically stops when the invocation completes.

Similar to 'tail -f' for build logs. For completed invocations, shows the full log once.

Exit codes:
  0 - Invocation completed successfully
  1 - Invocation completed with failure
  2 - Timeout reached before completion`,
	Args: cobra.ExactArgs(1),
	RunE: runLogTail,
}

var (
	flagLogGrep   string
	flagLogTail   int
	flagLogHead   int
	flagLogTarget string
	flagLogRaw    bool
	flagLogJunit  bool

	// Tail command flags
	flagTailPoll    int
	flagTailTimeout int
	flagTailQuiet   bool
)

func init() {
	logGetCmd.Flags().StringVar(&flagLogGrep, "grep", "", "Filter log lines matching pattern")
	logGetCmd.Flags().IntVar(&flagLogTail, "tail", 0, "Show only last N lines")
	logGetCmd.Flags().IntVar(&flagLogHead, "head", 0, "Show only first N lines")
	logGetCmd.Flags().StringVar(&flagLogTarget, "target", "", "Target label to fetch test artifacts for (searches child invocations if not found)")
	logGetCmd.Flags().BoolVar(&flagLogRaw, "raw", false, "Show raw test.log output (with --target)")
	logGetCmd.Flags().BoolVar(&flagLogJunit, "junit", false, "Show raw JUnit XML (with --target)")

	logTailCmd.Flags().IntVar(&flagTailPoll, "poll", 5, "Polling interval in seconds")
	logTailCmd.Flags().IntVar(&flagTailTimeout, "timeout", 0, "Max wait time in seconds (0 = unlimited)")
	logTailCmd.Flags().BoolVar(&flagTailQuiet, "quiet", false, "Suppress status messages (only show log output)")

	logCmd.AddCommand(logGetCmd)
	logCmd.AddCommand(logTailCmd)
	rootCmd.AddCommand(logCmd)
}

func runLogGet(cmd *cobra.Command, args []string) error {
	// If --target is specified, fetch test artifacts for that target
	if flagLogTarget != "" {
		return runLogGetTarget(args[0])
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	req := &api.GetLogRequest{
		Selector: &api.LogSelector{
			InvocationID: args[0],
		},
	}

	var fullContents strings.Builder
	var resp api.GetLogResponse
	if err := client.CallAllPages("GetLog", req, &resp, func() {
		if resp.Log != nil {
			fullContents.WriteString(resp.Log.Contents)
		}
	}); err != nil {
		return output.NewAPIError(fmt.Sprintf("GetLog: %v", err))
	}

	content := fullContents.String()
	if content == "" {
		return output.NewNotFoundError("no log found for invocation")
	}

	opts := GetOutputOptions()

	if opts.Mode == output.ModeJSON {
		return output.Render(map[string]interface{}{
			"log": map[string]interface{}{
				"contents": content,
			},
		}, opts)
	}

	if flagLogGrep != "" {
		lines := strings.Split(content, "\n")
		var filtered []string
		for _, line := range lines {
			if strings.Contains(line, flagLogGrep) {
				filtered = append(filtered, line)
			}
		}
		content = strings.Join(filtered, "\n")
	}

	lines := strings.Split(content, "\n")
	if flagLogHead > 0 && flagLogHead < len(lines) {
		lines = lines[:flagLogHead]
	}
	if flagLogTail > 0 && flagLogTail < len(lines) {
		lines = lines[len(lines)-flagLogTail:]
	}
	content = strings.Join(lines, "\n")

	return output.RenderStream(strings.NewReader(content), opts)
}

// runLogTail follows build log output by polling GetLog for an in-progress invocation.
func runLogTail(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	invocationID := args[0]
	if strings.Contains(invocationID, "/invocation/") {
		parts := strings.Split(invocationID, "/invocation/")
		invocationID = parts[len(parts)-1]
	}

	pollInterval := time.Duration(flagTailPoll) * time.Second
	var deadline time.Time
	if flagTailTimeout > 0 {
		deadline = time.Now().Add(time.Duration(flagTailTimeout) * time.Second)
	}

	opts := GetOutputOptions()
	isJSON := opts.Mode == output.ModeJSON

	var prevLogLen int
	var isFirstPoll bool = true

	for {
		// Check timeout
		if flagTailTimeout > 0 && time.Now().After(deadline) {
			if !isJSON && !flagTailQuiet {
				fmt.Fprintf(os.Stderr, "Timeout reached after %d seconds\n", flagTailTimeout)
			}
			return output.NewTimeoutError(fmt.Sprintf("timeout after %d seconds", flagTailTimeout))
		}

		// Fetch invocation status
		invReq := &api.GetInvocationRequest{
			Selector: &api.InvocationSelector{
				InvocationID: invocationID,
			},
			IncludeMetadata: true,
		}

		var invResp api.GetInvocationResponse
		if err := client.Call("GetInvocation", invReq, &invResp); err != nil {
			return output.NewAPIError(fmt.Sprintf("GetInvocation: %v", err))
		}

		if len(invResp.Invocation) == 0 {
			return output.NewNotFoundError(fmt.Sprintf("invocation %s not found", invocationID))
		}

		inv := invResp.Invocation[0]

		// Fetch full log contents
		logReq := &api.GetLogRequest{
			Selector: &api.LogSelector{
				InvocationID: invocationID,
			},
		}

		var fullContents strings.Builder
		var logResp api.GetLogResponse
		if err := client.CallAllPages("GetLog", logReq, &logResp, func() {
			if logResp.Log != nil {
				fullContents.WriteString(logResp.Log.Contents)
			}
		}); err != nil {
			return output.NewAPIError(fmt.Sprintf("GetLog: %v", err))
		}

		currentLog := fullContents.String()
		currentLogLen := len(currentLog)

		// If JSON mode, we need to wait for completion and return the full log
		if isJSON {
			if inv.InvocationStatus == "PARTIAL_INVOCATION_STATUS" {
				// Still in progress, wait and poll again
				time.Sleep(pollInterval)
				continue
			}

			// Complete - return full response
			result := map[string]interface{}{
				"invocationId":     invocationID,
				"invocationStatus": inv.InvocationStatus,
				"success":          inv.Success,
				"log": map[string]interface{}{
					"contents": currentLog,
				},
			}
			if err := output.Render(result, opts); err != nil {
				return err
			}
			if inv.Success {
				return nil
			}
			return output.NewInvocationFailedError(fmt.Sprintf("invocation %s failed", invocationID))
		}

		// Streaming mode: show only new content
		if currentLogLen > prevLogLen {
			newContent := currentLog[prevLogLen:]
			// Print new content immediately (no trailing newline if content doesn't end with one)
			fmt.Print(newContent)
			prevLogLen = currentLogLen
		}

		// Check if invocation is complete
		if inv.InvocationStatus != "PARTIAL_INVOCATION_STATUS" {
			// Invocation complete
			if !flagTailQuiet {
				// Print final status
				status := "PASS"
				if !inv.Success {
					status = "FAIL"
				}
				if inv.InvocationStatus == "DISCONNECTED_INVOCATION_STATUS" {
					status = "DISCONNECTED"
				}
				fmt.Fprintf(os.Stderr, "\n[%s] Build %s: %s (%s)\n",
					time.Now().Format("15:04:05"),
					status,
					truncateStr(invocationID, 20),
					formatDuration(inv.DurationUsec.Int64()))
			}

			if inv.Success {
				return nil
			}
			return output.NewInvocationFailedError(fmt.Sprintf("invocation %s failed", invocationID))
		}

		// Print status update if not quiet and this isn't the first poll
		if !flagTailQuiet && !isFirstPoll {
			fmt.Fprintf(os.Stderr, "[%s] Still running (%s)...\r",
				time.Now().Format("15:04:05"),
				formatDuration(inv.DurationUsec.Int64()))
		}
		isFirstPoll = false

		time.Sleep(pollInterval)
	}
}

// targetArtifacts holds the test artifacts found for a target.
type targetArtifacts struct {
	TestXMLURI  string
	TestLogURI  string
	InvocationID string
}

// runLogGetTarget fetches test artifacts (test.xml/test.log) for a specific target.
// For workflow step targets (steps[N]), it fetches the console buffer instead.
// It searches the parent invocation first, then child invocations if not found.
func runLogGetTarget(invocationID string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	// Check if this is a workflow step target
	if logWorkflowStepPattern.MatchString(flagLogTarget) {
		return runLogGetWorkflowStep(client, invocationID, flagLogTarget)
	}

	// First, try to find artifacts in the parent invocation
	artifacts, err := findTargetArtifacts(client, invocationID, flagLogTarget)
	if err != nil {
		return err
	}

	// If not found in parent, search child invocations
	if artifacts == nil {
		artifacts, err = findTargetArtifactsInChildren(client, invocationID, flagLogTarget)
		if err != nil {
			return err
		}
	}

	if artifacts == nil {
		return output.NewNotFoundError(fmt.Sprintf("target %q not found in invocation %s or any child invocations", flagLogTarget, invocationID))
	}

	// Fetch the appropriate content based on flags
	return renderTargetArtifacts(client, artifacts)
}

// runLogGetWorkflowStep fetches the console buffer for a workflow step target.
// Workflow steps (steps[N]) don't produce test.xml/test.log - they need console output instead.
func runLogGetWorkflowStep(client *api.Client, invocationID, targetLabel string) error {
	// Fetch the invocation with console buffer
	invReq := &api.GetInvocationRequest{
		Selector: &api.InvocationSelector{
			InvocationID: invocationID,
		},
	}

	var invResp api.GetInvocationResponse
	if err := client.Call("GetInvocation", invReq, &invResp); err != nil {
		return output.NewAPIError(fmt.Sprintf("GetInvocation: %v", err))
	}

	if len(invResp.Invocation) == 0 {
		return output.NewNotFoundError(fmt.Sprintf("invocation %s not found", invocationID))
	}

	inv := invResp.Invocation[0]
	content := inv.ConsoleBuffer

	if content == "" {
		return output.NewNotFoundError(fmt.Sprintf("no console output found for workflow step %q in invocation %s", targetLabel, invocationID))
	}

	opts := GetOutputOptions()

	// JSON mode
	if opts.Mode == output.ModeJSON {
		return output.Render(map[string]interface{}{
			"target":       targetLabel,
			"invocationId": invocationID,
			"sourceType":   "workflow_step",
			"content":      content,
		}, opts)
	}

	// Apply filters (--grep, --head, --tail)
	content = applyLogFilters(content)

	return output.RenderStream(strings.NewReader(content), opts)
}

// applyLogFilters applies --grep, --head, and --tail filters to content.
func applyLogFilters(content string) string {
	if flagLogGrep != "" {
		lines := strings.Split(content, "\n")
		var filtered []string
		for _, line := range lines {
			if strings.Contains(line, flagLogGrep) {
				filtered = append(filtered, line)
			}
		}
		content = strings.Join(filtered, "\n")
	}

	lines := strings.Split(content, "\n")
	if flagLogHead > 0 && flagLogHead < len(lines) {
		lines = lines[:flagLogHead]
	}
	if flagLogTail > 0 && flagLogTail < len(lines) {
		lines = lines[len(lines)-flagLogTail:]
	}
	return strings.Join(lines, "\n")
}

// findTargetArtifacts searches for test.xml/test.log files for a target in a single invocation.
func findTargetArtifacts(client *api.Client, invocationID, targetLabel string) (*targetArtifacts, error) {
	req := &api.GetActionRequest{
		Selector: &api.ActionSelector{
			InvocationID: invocationID,
			TargetLabel:  targetLabel,
		},
	}

	var allActions []api.Action
	var resp api.GetActionResponse
	if err := client.CallAllPages("GetAction", req, &resp, func() {
		allActions = append(allActions, resp.Action...)
	}); err != nil {
		return nil, fmt.Errorf("GetAction for %s: %w", invocationID, err)
	}

	// Look for test.xml or test.log files
	var testXMLURI, testLogURI string
	for _, a := range allActions {
		for _, f := range a.File {
			if f.Name == "test.xml" && testXMLURI == "" {
				testXMLURI = f.URI
			}
			if f.Name == "test.log" && testLogURI == "" {
				testLogURI = f.URI
			}
		}
	}

	if testXMLURI == "" && testLogURI == "" {
		return nil, nil // Not found
	}

	return &targetArtifacts{
		TestXMLURI:   testXMLURI,
		TestLogURI:   testLogURI,
		InvocationID: invocationID,
	}, nil
}

// findTargetArtifactsInChildren searches child invocations for target artifacts.
func findTargetArtifactsInChildren(client *api.Client, invocationID, targetLabel string) (*targetArtifacts, error) {
	// Fetch the invocation with child invocation IDs
	invReq := &api.GetInvocationRequest{
		Selector: &api.InvocationSelector{
			InvocationID: invocationID,
		},
		IncludeChildInvocations: true,
	}

	var invResp api.GetInvocationResponse
	if err := client.Call("GetInvocation", invReq, &invResp); err != nil {
		return nil, fmt.Errorf("GetInvocation for %s: %w", invocationID, err)
	}

	if len(invResp.Invocation) == 0 {
		return nil, nil
	}

	// Search each child invocation
	for _, childID := range invResp.Invocation[0].ChildInvocations {
		if childID.InvocationID == "" {
			continue
		}

		artifacts, err := findTargetArtifacts(client, childID.InvocationID, targetLabel)
		if err != nil {
			// Continue on error, try other children
			continue
		}

		if artifacts != nil {
			return artifacts, nil
		}

		// Recursively search grandchildren
		artifacts, err = findTargetArtifactsInChildren(client, childID.InvocationID, targetLabel)
		if err != nil {
			continue
		}
		if artifacts != nil {
			return artifacts, nil
		}
	}

	return nil, nil
}

// renderTargetArtifacts outputs the test artifacts based on the flags.
func renderTargetArtifacts(client *api.Client, artifacts *targetArtifacts) error {
	opts := GetOutputOptions()

	// If --raw flag, just show test.log content
	if flagLogRaw {
		if artifacts.TestLogURI == "" {
			return output.NewNotFoundError("no test.log found for target")
		}
		return fetchAndRenderFile(client, artifacts.TestLogURI, opts)
	}

	// If --junit flag, show raw XML
	if flagLogJunit {
		if artifacts.TestXMLURI == "" {
			return output.NewNotFoundError("no test.xml found for target")
		}
		return fetchAndRenderFile(client, artifacts.TestXMLURI, opts)
	}

	// Default: parse JUnit XML and show formatted results
	if artifacts.TestXMLURI != "" {
		return fetchAndRenderJUnit(client, artifacts, opts)
	}

	// Fall back to test.log if no test.xml
	if artifacts.TestLogURI != "" {
		return fetchAndRenderFile(client, artifacts.TestLogURI, opts)
	}

	return output.NewNotFoundError("no test artifacts found for target")
}

// fetchAndRenderFile fetches a file and renders it.
// GetFile returns raw bytes (not JSON), so we use CallRaw to avoid JSON decode errors.
func fetchAndRenderFile(client *api.Client, uri string, opts output.Options) error {
	req := api.GetFileRequest{URI: uri}
	body, err := client.CallRaw("GetFile", req)
	if err != nil {
		return output.NewAPIError(fmt.Sprintf("GetFile: %v", err))
	}
	defer func() { _ = body.Close() }()

	data, err := io.ReadAll(body)
	if err != nil {
		return output.NewAPIError(fmt.Sprintf("GetFile: read response: %v", err))
	}

	content := string(data)

	if opts.Mode == output.ModeJSON {
		return output.Render(map[string]interface{}{
			"content": content,
		}, opts)
	}

	return output.RenderStream(strings.NewReader(content), opts)
}

// JUnit XML types for parsing test results.
type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure"`
	Error     *junitError   `xml:"error"`
	Skipped   *junitSkipped `xml:"skipped"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr"`
}

// testResult represents a parsed test result for output.
type testResult struct {
	Name      string `json:"name"`
	ClassName string `json:"className,omitempty"`
	Status    string `json:"status"`
	Time      string `json:"time,omitempty"`
	Message   string `json:"message,omitempty"`
}

// testSummary represents parsed test results for JSON output.
type testSummary struct {
	InvocationID string       `json:"invocationId"`
	Target       string       `json:"target"`
	Total        int          `json:"total"`
	Passed       int          `json:"passed"`
	Failed       int          `json:"failed"`
	Errors       int          `json:"errors"`
	Skipped      int          `json:"skipped"`
	Time         string       `json:"time,omitempty"`
	Tests        []testResult `json:"tests"`
}

// fetchAndRenderJUnit fetches and parses JUnit XML, then renders the results.
// GetFile returns raw bytes (not JSON), so we use CallRaw to avoid JSON decode errors.
func fetchAndRenderJUnit(client *api.Client, artifacts *targetArtifacts, opts output.Options) error {
	req := api.GetFileRequest{URI: artifacts.TestXMLURI}
	body, err := client.CallRaw("GetFile", req)
	if err != nil {
		return output.NewAPIError(fmt.Sprintf("GetFile: %v", err))
	}
	defer func() { _ = body.Close() }()

	data, err := io.ReadAll(body)
	if err != nil {
		return output.NewAPIError(fmt.Sprintf("GetFile: read response: %v", err))
	}

	// Try parsing as <testsuites> first
	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		// Try parsing as single <testsuite>
		var suite junitTestSuite
		if err2 := xml.Unmarshal(data, &suite); err2 != nil {
			// Can't parse, just return raw content
			return output.RenderStream(strings.NewReader(string(data)), opts)
		}
		suites.TestSuites = []junitTestSuite{suite}
	}

	// Build summary
	summary := testSummary{
		InvocationID: artifacts.InvocationID,
		Target:       flagLogTarget,
		Tests:        make([]testResult, 0),
	}

	for _, suite := range suites.TestSuites {
		summary.Total += suite.Tests
		summary.Failed += suite.Failures
		summary.Errors += suite.Errors
		summary.Skipped += suite.Skipped
		if suite.Time != "" {
			summary.Time = suite.Time
		}

		for _, tc := range suite.TestCases {
			result := testResult{
				Name:      tc.Name,
				ClassName: tc.ClassName,
				Time:      tc.Time,
			}

			if tc.Failure != nil {
				result.Status = "FAILED"
				result.Message = tc.Failure.Message
				if result.Message == "" {
					result.Message = tc.Failure.Content
				}
			} else if tc.Error != nil {
				result.Status = "ERROR"
				result.Message = tc.Error.Message
				if result.Message == "" {
					result.Message = tc.Error.Content
				}
			} else if tc.Skipped != nil {
				result.Status = "SKIPPED"
				result.Message = tc.Skipped.Message
			} else {
				result.Status = "PASSED"
			}

			summary.Tests = append(summary.Tests, result)
		}
	}

	summary.Passed = summary.Total - summary.Failed - summary.Errors - summary.Skipped

	// Render output
	if opts.Mode == output.ModeJSON {
		return output.Render(summary, opts)
	}

	// Table output
	if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
		td := output.TableData{
			Headers: []string{"STATUS", "NAME", "TIME", "MESSAGE"},
		}
		for _, t := range summary.Tests {
			msg := t.Message
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			td.Rows = append(td.Rows, []string{
				t.Status,
				truncateStr(t.Name, 50),
				t.Time,
				msg,
			})
		}

		// Print summary line first
		fmt.Printf("Target: %s (from %s)\n", flagLogTarget, artifacts.InvocationID)
		fmt.Printf("Total: %d, Passed: %d, Failed: %d, Errors: %d, Skipped: %d\n\n",
			summary.Total, summary.Passed, summary.Failed, summary.Errors, summary.Skipped)

		return output.RenderTable(td, summary.Tests, opts)
	}

	return output.Render(summary, opts)
}
