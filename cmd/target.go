package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

// workflowStepPattern matches workflow step target labels like "steps[0]", "steps[1]", etc.
var workflowStepPattern = regexp.MustCompile(`^steps\[\d+\]$`)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "Query build targets",
}

var targetListCmd = &cobra.Command{
	Use:     "list <invocation-id>",
	Aliases: []string{"ls"},
	Short:   "List all targets for an invocation",
	Args:    cobra.ExactArgs(1),
	RunE:    runTargetList,
}

var targetFailedCmd = &cobra.Command{
	Use:   "failed <invocation-id>",
	Short: "Show only failed/flaky targets (searches child invocations by default)",
	Args:  cobra.ExactArgs(1),
	RunE:  runTargetFailed,
}

var (
	flagTargetLabel string
	flagTargetTag   string
	flagTargetDepth int
)

func init() {
	targetListCmd.Flags().StringVar(&flagTargetLabel, "label", "", "Filter by target label")
	targetListCmd.Flags().StringVar(&flagTargetTag, "tag", "", "Filter by target tag")
	targetFailedCmd.Flags().StringVar(&flagTargetLabel, "label", "", "Filter by target label")
	targetFailedCmd.Flags().IntVar(&flagTargetDepth, "depth", -1, "Search depth: -1 (unlimited), 0 (top-level only), N (up to N levels)")

	targetCmd.AddCommand(targetListCmd)
	targetCmd.AddCommand(targetFailedCmd)
	rootCmd.AddCommand(targetCmd)
}

func runTargetList(cmd *cobra.Command, args []string) error {
	return fetchTargets(args[0], false)
}

func runTargetFailed(cmd *cobra.Command, args []string) error {
	return fetchTargetsWithChildren(args[0], flagTargetDepth)
}

func fetchTargets(invocationID string, failedOnly bool) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	req := &api.GetTargetRequest{
		Selector: &api.TargetSelector{
			InvocationID: invocationID,
			Label:        flagTargetLabel,
			Tag:          flagTargetTag,
		},
	}

	var allTargets []api.Target
	var resp api.GetTargetResponse
	if err := client.CallAllPages("GetTarget", req, &resp, func() {
		allTargets = append(allTargets, resp.Target...)
	}); err != nil {
		return output.NewAPIError(fmt.Sprintf("GetTarget: %v", err))
	}

	if failedOnly {
		var filtered []api.Target
		for _, t := range allTargets {
			s := strings.ToUpper(t.Status)
			if s == "FAILED" || s == "FLAKY" || s == "TIMED_OUT" || s == "BUILD_FAILED" {
				filtered = append(filtered, t)
			}
		}
		allTargets = filtered
	}

	opts := GetOutputOptions()
	if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
		td := targetsToTable(allTargets)
		return output.RenderTable(td, allTargets, opts)
	}
	return output.Render(allTargets, opts)
}

// targetWithInvocation extends Target with its source invocation ID for JSON output.
type targetWithInvocation struct {
	api.Target
	InvocationID string `json:"invocationId"`
	SourceType   string `json:"sourceType,omitempty"` // "workflow_step" for steps[N] targets
}

// isWorkflowStep returns true if the target label matches the workflow step pattern (steps[N]).
func isWorkflowStep(label string) bool {
	return workflowStepPattern.MatchString(label)
}

// fetchTargetsWithChildren fetches failed targets from the invocation and its children.
// depth: -1 = unlimited, 0 = top-level only, N = up to N levels deep.
func fetchTargetsWithChildren(invocationID string, depth int) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	seenLabels := make(map[string]bool)
	allTargets := make([]targetWithInvocation, 0)

	// Recursive function to fetch targets from an invocation and its children
	var fetchRecursive func(invID string, currentDepth int) error
	fetchRecursive = func(invID string, currentDepth int) error {
		// Fetch targets for this invocation
		req := &api.GetTargetRequest{
			Selector: &api.TargetSelector{
				InvocationID: invID,
				Label:        flagTargetLabel,
			},
		}

		var targets []api.Target
		var resp api.GetTargetResponse
		if err := client.CallAllPages("GetTarget", req, &resp, func() {
			targets = append(targets, resp.Target...)
		}); err != nil {
			return fmt.Errorf("GetTarget for %s: %w", invID, err)
		}

		// Filter for failed/flaky targets and deduplicate by label
		for _, t := range targets {
			s := strings.ToUpper(t.Status)
			if s == "FAILED" || s == "FLAKY" || s == "TIMED_OUT" || s == "BUILD_FAILED" {
				if !seenLabels[t.Label] {
					seenLabels[t.Label] = true
					twi := targetWithInvocation{
						Target:       t,
						InvocationID: invID,
					}
					if isWorkflowStep(t.Label) {
						twi.SourceType = "workflow_step"
					}
					allTargets = append(allTargets, twi)
				}
			}
		}

		// If depth is 0 (top-level only), don't recurse into children
		// If currentDepth has reached the limit (for positive depth), stop
		if depth == 0 || (depth > 0 && currentDepth >= depth) {
			return nil
		}

		// Fetch the invocation to get child invocation IDs
		invReq := &api.GetInvocationRequest{
			Selector: &api.InvocationSelector{
				InvocationID: invID,
			},
			IncludeChildInvocations: true,
		}

		var invResp api.GetInvocationResponse
		if err := client.Call("GetInvocation", invReq, &invResp); err != nil {
			return fmt.Errorf("GetInvocation for %s: %w", invID, err)
		}

		// Recurse into child invocations
		if len(invResp.Invocation) > 0 {
			for _, childID := range invResp.Invocation[0].ChildInvocations {
				if childID.InvocationID != "" {
					if err := fetchRecursive(childID.InvocationID, currentDepth+1); err != nil {
						// Log but continue on child errors
						continue
					}
				}
			}
		}

		return nil
	}

	// Start the recursive fetch
	if err := fetchRecursive(invocationID, 0); err != nil {
		return output.NewAPIError(err.Error())
	}

	opts := GetOutputOptions()
	if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
		td := targetsWithInvocationToTable(allTargets)
		return output.RenderTable(td, allTargets, opts)
	}
	return output.Render(allTargets, opts)
}

func targetsToTable(targets []api.Target) output.TableData {
	td := output.TableData{
		Headers: []string{"LABEL", "STATUS", "DURATION", "TYPE", "LANGUAGE"},
	}
	for _, t := range targets {
		duration := "-"
		if t.Timing != nil && t.Timing.Duration != "" {
			duration = t.Timing.Duration
		}
		td.Rows = append(td.Rows, []string{
			truncateStr(t.Label, 80),
			t.Status,
			duration,
			t.RuleType,
			t.Language,
		})
	}
	return td
}

func targetsWithInvocationToTable(targets []targetWithInvocation) output.TableData {
	td := output.TableData{
		Headers: []string{"LABEL", "STATUS", "DURATION", "TYPE", "LANGUAGE"},
	}
	for _, t := range targets {
		duration := "-"
		if t.Timing != nil && t.Timing.Duration != "" {
			duration = t.Timing.Duration
		}
		label := truncateStr(t.Label, 80)
		if t.SourceType == "workflow_step" {
			label = label + " (workflow step)"
		}
		td.Rows = append(td.Rows, []string{
			label,
			t.Status,
			duration,
			t.RuleType,
			t.Language,
		})
	}
	return td
}
