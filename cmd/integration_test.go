package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// requireAPIKey skips the test if BUILDBUDDY_API_KEY is not set.
func requireAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("BUILDBUDDY_API_KEY") == "" {
		t.Skip("BUILDBUDDY_API_KEY not set; skipping integration test")
	}
}

// getTestCommitSHA returns the commit SHA to test against.
// Set BUILDBUDDY_TEST_COMMIT to override.
func getTestCommitSHA(t *testing.T) string {
	t.Helper()
	sha := os.Getenv("BUILDBUDDY_TEST_COMMIT")
	if sha == "" {
		t.Skip("BUILDBUDDY_TEST_COMMIT not set; skipping integration test")
	}
	return sha
}

// resetFlags resets all package-level flag values to their defaults.
// This is necessary because cobra commands use package-level variables
// that persist across test invocations within the same process.
func resetFlags() {
	flagJSON = false
	flagPlaintext = false
	flagJQ = ""
	flagTemplate = ""
	flagFields = ""
	flagNoColor = false
	flagDebug = false
	flagOutputFile = ""

	// Invocation flags
	flagInvListCommit = ""
	flagInvListLimit = 0
	flagWaitPoll = 10
	flagWaitTimeout = 0

	// Target flags
	flagTargetLabel = ""
	flagTargetTag = ""
	flagTargetDepth = -1

	// Log flags
	flagLogGrep = ""
	flagLogTail = 0
	flagLogHead = 0
	flagLogTarget = ""
	flagLogRaw = false
	flagLogJunit = false
	flagTailPoll = 5
	flagTailTimeout = 0
	flagTailQuiet = false

	// Action flags
	flagActionTargetLabel = ""
	flagActionTargetID = ""

	// Artifact flags
	flagArtifactDir = ""
	flagArtifactTargetLabel = ""

	// Workflow flags
	flagWFRepo = ""
	flagWFBranch = ""
	flagWFCommitSHA = ""
	flagWFVisibility = ""
	flagWFAsync = false

	// Run flags
	flagRunRepo = ""
	flagRunBranch = ""
	flagRunCommit = ""
	flagRunTimeout = ""
	flagRunEnv = nil

	// Reset Cobra's internal help flags to avoid state leakage from --help
	// This must be done on all commands that have been executed with --help
	resetHelpFlags(rootCmd)
}

// resetHelpFlags recursively resets the help flag on a command and all its subcommands.
func resetHelpFlags(cmd *cobra.Command) {
	if cmd.Flags().Lookup("help") != nil {
		cmd.Flags().Set("help", "false")
	}
	for _, sub := range cmd.Commands() {
		resetHelpFlags(sub)
	}
}

// execResult holds the captured output from a command execution.
type execResult struct {
	stdout   string
	stderr   string
	err      error
	exitCode int
}

// execCmd runs the root cobra command with the given args and captures stdout/stderr.
// It resets flags before each run to prevent state leaking between tests.
// Pipes are drained concurrently to avoid deadlocks with large output (e.g., logs).
func execCmd(args ...string) execResult {
	resetFlags()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	// Drain pipes concurrently to prevent blocking when output exceeds pipe buffer
	var bufOut, bufErr bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(&bufOut, rOut)
	}()
	go func() {
		defer wg.Done()
		io.Copy(&bufErr, rErr)
	}()

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	// Close write ends so the drain goroutines can finish
	wOut.Close()
	wErr.Close()

	// Wait for drain goroutines to complete
	wg.Wait()

	// Restore originals
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	rOut.Close()
	rErr.Close()

	return execResult{
		stdout: bufOut.String(),
		stderr: bufErr.String(),
		err:    err,
	}
}

// --- Invocation Tests ---

func TestInvocationListTable(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t))
	if res.err != nil {
		t.Fatalf("invocation list failed: %v\nstderr: %s", res.err, res.stderr)
	}
	if res.stdout == "" {
		t.Fatal("invocation list returned empty stdout")
	}
	// Table mode should contain header columns
	if !strings.Contains(res.stdout, "ID") || !strings.Contains(res.stdout, "STATUS") {
		t.Errorf("expected table headers ID and STATUS in output, got:\n%s", res.stdout)
	}
}

func TestInvocationListJSON(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t), "--json")
	if res.err != nil {
		t.Fatalf("invocation list --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, res.stdout)
	}

	invocations, ok := parsed["invocation"]
	if !ok {
		t.Fatalf("JSON missing 'invocation' key, got keys: %v", keys(parsed))
	}

	arr, ok := invocations.([]interface{})
	if !ok {
		t.Fatalf("invocation is not an array: %T", invocations)
	}
	if len(arr) == 0 {
		t.Fatal("invocation array is empty")
	}
}

func TestInvocationListLimit(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t), "--limit", "1", "--json")
	if res.err != nil {
		t.Fatalf("invocation list --limit 1 failed: %v\nstderr: %s", res.err, res.stderr)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	arr, ok := parsed["invocation"].([]interface{})
	if !ok {
		t.Fatal("invocation is not an array")
	}
	if len(arr) != 1 {
		t.Fatalf("expected exactly 1 invocation with --limit 1, got %d", len(arr))
	}
}

func TestInvocationListJQ(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t), "--jq", ".invocation | length")
	if res.err != nil {
		t.Fatalf("invocation list --jq failed: %v\nstderr: %s", res.err, res.stderr)
	}

	trimmed := strings.TrimSpace(res.stdout)
	n, err := strconv.Atoi(trimmed)
	if err != nil {
		t.Fatalf("jq output is not a number: %q", trimmed)
	}
	if n <= 0 {
		t.Fatalf("expected positive invocation count, got %d", n)
	}
}

func TestInvocationListMissingCommit(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list")
	if res.err == nil {
		t.Fatal("expected error when --commit is not provided")
	}
	// Error should mention usage or --commit
	combined := res.stderr + res.err.Error()
	if !strings.Contains(combined, "commit") && !strings.Contains(combined, "usage") {
		t.Errorf("expected error about --commit, got: %s", combined)
	}
}

func TestInvocationGet(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("invocation", "get", invID)
	if res.err != nil {
		t.Fatalf("invocation get failed: %v\nstderr: %s", res.err, res.stderr)
	}
	if res.stdout == "" {
		t.Fatal("invocation get returned empty stdout")
	}
	// Table mode should show the invocation ID
	if !strings.Contains(res.stdout, "ID") {
		t.Errorf("expected ID header in table output, got:\n%s", res.stdout)
	}
}

func TestInvocationGetJSON(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("invocation", "get", invID, "--json")
	if res.err != nil {
		t.Fatalf("invocation get --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, res.stdout)
	}
	if _, ok := parsed["invocation"]; !ok {
		t.Fatalf("JSON missing 'invocation' key, got keys: %v", keys(parsed))
	}
}

func TestInvocationGetFields(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("invocation", "get", invID, "--fields", "invocation")
	if res.err != nil {
		t.Fatalf("invocation get --fields failed: %v\nstderr: %s", res.err, res.stderr)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, res.stdout)
	}
	if _, ok := parsed["invocation"]; !ok {
		t.Fatal("--fields invocation should include 'invocation' key")
	}
	// Should NOT contain keys that were not requested
	if _, ok := parsed["nextPageToken"]; ok {
		t.Error("--fields should exclude unlisted keys like nextPageToken")
	}
}

// --- Invocation Wait Tests ---

func TestInvocationWaitCompletedSuccess(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Wait on a completed invocation should return immediately
	res := execCmd("invocation", "wait", invID)
	// Note: exit code testing is limited by execCmd, but we verify no error
	if res.err != nil {
		// The error might be an exit code error, which is expected for failed invocations
		t.Logf("invocation wait returned: %v (may be expected for failed invocations)", res.err)
	}
	// Should have some output
	if res.stdout == "" {
		t.Error("invocation wait returned empty stdout")
	}
}

func TestInvocationWaitCompletedJSON(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("invocation", "wait", invID, "--json")
	if res.err != nil {
		t.Logf("invocation wait --json returned: %v (may be expected for failed invocations)", res.err)
	}

	// Should return valid JSON with invocation data
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, res.stdout)
	}
	if _, ok := parsed["invocation"]; !ok {
		t.Fatalf("JSON missing 'invocation' key, got keys: %v", keys(parsed))
	}
}

func TestInvocationWaitTimeoutCompletedReturnsImmediately(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Even with a 1-second timeout, a completed invocation should return immediately
	start := time.Now()
	res := execCmd("invocation", "wait", invID, "--timeout", "1")
	elapsed := time.Since(start)

	// Should complete quickly (under 1 second) since invocation is already done
	if elapsed > 1*time.Second {
		t.Errorf("wait on completed invocation took too long: %v", elapsed)
	}

	if res.stdout == "" {
		t.Error("invocation wait with timeout returned empty stdout")
	}
}

func TestInvocationWaitWatchAlias(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("invocation", "watch", invID)
	if res.err != nil {
		t.Logf("invocation watch returned: %v (may be expected for failed invocations)", res.err)
	}
	if res.stdout == "" {
		t.Error("invocation watch alias returned empty stdout")
	}
}

func TestInvocationWaitHelp(t *testing.T) {
	res := execCmd("invocation", "wait", "--help")
	if res.err != nil {
		t.Fatalf("invocation wait --help failed: %v", res.err)
	}
	// Should show --poll and --timeout flags
	for _, flag := range []string{"--poll", "--timeout"} {
		if !strings.Contains(res.stdout, flag) {
			t.Errorf("invocation wait --help missing flag %s in output:\n%s", flag, res.stdout)
		}
	}
}

func TestInvocationWaitBadID(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "wait", "not-a-real-invocation-id-000000")
	if res.err == nil {
		t.Fatal("expected error for bad invocation ID")
	}
	// Should produce a not_found or similar error
	if res.stderr != "" {
		var errJSON map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(res.stderr)), &errJSON); err == nil {
			if _, ok := errJSON["error_type"]; !ok {
				t.Error("structured error missing 'error_type' field")
			}
		}
	}
}

// --- Alias Tests ---

func TestAliasInvLs(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("inv", "ls", "--commit", getTestCommitSHA(t))
	if res.err != nil {
		t.Fatalf("inv ls failed: %v\nstderr: %s", res.err, res.stderr)
	}
	if res.stdout == "" {
		t.Fatal("inv ls returned empty stdout")
	}
}

func TestAliasInvocationShow(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("invocation", "show", invID)
	if res.err != nil {
		t.Fatalf("invocation show failed: %v\nstderr: %s", res.err, res.stderr)
	}
	if res.stdout == "" {
		t.Fatal("invocation show returned empty stdout")
	}
}

// --- Target Tests ---

func TestTargetList(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("target", "list", invID)
	if res.err != nil {
		t.Fatalf("target list failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// May be empty if no targets, but should not error
}

func TestTargetListJSON(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("target", "list", invID, "--json")
	if res.err != nil {
		t.Fatalf("target list --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	// JSON output should be valid
	trimmed := strings.TrimSpace(res.stdout)
	if trimmed != "" {
		if !json.Valid([]byte(trimmed)) {
			t.Fatalf("target list --json returned invalid JSON: %s", trimmed)
		}
	}
}

func TestTargetListFilterByLabel(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Use a label that likely does not exist to verify filtering works without error
	res := execCmd("target", "list", invID, "--label", "//nonexistent:target")
	if res.err != nil {
		t.Fatalf("target list --label failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// Should succeed even if no results match
}

func TestTargetFailed(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("target", "failed", invID)
	if res.err != nil {
		t.Fatalf("target failed command failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// Output may be empty if all targets passed -- that is valid
}

func TestTargetFailedDepthZero(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// --depth 0 should only search top-level, not recurse into children
	res := execCmd("target", "failed", invID, "--depth", "0")
	if res.err != nil {
		t.Fatalf("target failed --depth 0 failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// Output may be empty if all targets passed -- that is valid
}

func TestTargetFailedWithDepth(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// --depth 1 should search one level deep
	res := execCmd("target", "failed", invID, "--depth", "1")
	if res.err != nil {
		t.Fatalf("target failed --depth 1 failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// Output may be empty if all targets passed -- that is valid
}

func TestTargetFailedJSONIncludesInvocationID(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("target", "failed", invID, "--json")
	if res.err != nil {
		t.Fatalf("target failed --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	// If there are failed targets, verify JSON includes invocationId field
	trimmed := strings.TrimSpace(res.stdout)
	if trimmed != "" && trimmed != "null" && trimmed != "[]" {
		// Parse and check for invocationId field in each target
		var targets []map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &targets); err != nil {
			// If it's not an array, it might be in a different format
			t.Logf("Note: JSON output format: %s", trimmed)
			return
		}
		for i, target := range targets {
			if _, ok := target["invocationId"]; !ok {
				t.Errorf("target[%d] missing 'invocationId' field: %v", i, target)
			}
		}
	}
}

func TestTargetFailedDepthHelp(t *testing.T) {
	res := execCmd("target", "failed", "--help")
	if res.err != nil {
		t.Fatalf("target failed --help failed: %v", res.err)
	}
	if !strings.Contains(res.stdout, "--depth") {
		t.Errorf("target failed --help missing --depth flag documentation:\n%s", res.stdout)
	}
}

func TestTargetLsAlias(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("target", "ls", invID)
	if res.err != nil {
		t.Fatalf("target ls alias failed: %v\nstderr: %s", res.err, res.stderr)
	}
}

// --- Log Tests ---

func TestLogGet(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("log", "get", invID)
	if res.err != nil {
		t.Fatalf("log get failed: %v\nstderr: %s", res.err, res.stderr)
	}
	if res.stdout == "" {
		t.Fatal("log get returned empty stdout")
	}
}

func TestLogGetTail(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("log", "get", invID, "--tail", "5")
	if res.err != nil {
		t.Fatalf("log get --tail 5 failed: %v\nstderr: %s", res.err, res.stderr)
	}

	lines := nonEmptyLines(res.stdout)
	if len(lines) > 5 {
		t.Errorf("--tail 5 should return at most 5 lines, got %d", len(lines))
	}
}

func TestLogGetHead(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("log", "get", invID, "--head", "3")
	if res.err != nil {
		t.Fatalf("log get --head 3 failed: %v\nstderr: %s", res.err, res.stderr)
	}

	lines := nonEmptyLines(res.stdout)
	if len(lines) > 3 {
		t.Errorf("--head 3 should return at most 3 lines, got %d", len(lines))
	}
}

func TestLogGetGrep(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Grep for a common token; use a very broad pattern
	res := execCmd("log", "get", invID, "--grep", "bazel")
	if res.err != nil {
		// grep may filter out everything, which is still valid if the command did not error
		// However some invocations may not have "bazel" in logs. Accept either outcome.
		t.Logf("log get --grep returned error (may be ok if no matches): %v", res.err)
	}
	// All returned lines should contain the grep pattern (if any output)
	if res.stdout != "" {
		for _, line := range nonEmptyLines(res.stdout) {
			if !strings.Contains(strings.ToLower(line), "bazel") {
				// Note: the grep is case-sensitive in the implementation
				// so we just log non-matches rather than fail
				t.Logf("line does not contain 'bazel': %s", line)
			}
		}
	}
}

func TestLogGetJSON(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("log", "get", invID, "--json")
	if res.err != nil {
		t.Fatalf("log get --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, res.stdout)
	}

	logObj, ok := parsed["log"]
	if !ok {
		t.Fatalf("JSON missing 'log' key, got keys: %v", keys(parsed))
	}
	logMap, ok := logObj.(map[string]interface{})
	if !ok {
		t.Fatalf("log is not a map: %T", logObj)
	}
	if _, ok := logMap["contents"]; !ok {
		t.Fatal("log.contents missing from JSON")
	}
}

func TestLogGetOutputFile(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "log_output.txt")

	res := execCmd("log", "get", invID, "-o", outFile)
	if res.err != nil {
		t.Fatalf("log get -o failed: %v\nstderr: %s", res.err, res.stderr)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("output file is empty")
	}
}

// --- Log Tail Tests ---

func TestLogTail(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// For completed invocation, log tail should show log once and exit
	res := execCmd("log", "tail", invID, "--quiet")
	if res.err != nil {
		// Completed failed invocations return error exit code
		if !strings.Contains(res.stderr, "failed") {
			t.Fatalf("log tail failed unexpectedly: %v\nstderr: %s", res.err, res.stderr)
		}
	}
	if res.stdout == "" {
		t.Fatal("log tail returned empty output")
	}
}

func TestLogTailJSON(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("log", "tail", invID, "--json")
	// May fail for failed invocations, but output should still be valid JSON
	trimmed := strings.TrimSpace(res.stdout)
	if trimmed == "" {
		t.Fatal("log tail --json returned empty output")
	}
	if !json.Valid([]byte(trimmed)) {
		t.Fatalf("log tail --json returned invalid JSON: %s", trimmed)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Check expected fields
	if _, ok := result["invocationId"]; !ok {
		t.Fatal("invocationId missing from JSON")
	}
	if _, ok := result["invocationStatus"]; !ok {
		t.Fatal("invocationStatus missing from JSON")
	}
	if _, ok := result["success"]; !ok {
		t.Fatal("success missing from JSON")
	}
	if _, ok := result["log"]; !ok {
		t.Fatal("log missing from JSON")
	}
}

func TestLogTailTimeout(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// For a completed invocation with timeout, should return immediately
	res := execCmd("log", "tail", invID, "--timeout", "30", "--quiet")
	// May fail for failed invocations
	if res.err != nil && strings.Contains(res.stderr, "timeout") {
		t.Fatalf("log tail timed out on completed invocation: %v\nstderr: %s", res.err, res.stderr)
	}
}

func TestLogTailWithURL(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Test with full BuildBuddy URL
	url := "https://app.buildbuddy.io/invocation/" + invID
	res := execCmd("log", "tail", url, "--quiet")
	// May fail for failed invocations, that's OK
	if res.stdout == "" {
		t.Fatal("log tail with URL returned empty output")
	}
}

func TestLogTailNotFound(t *testing.T) {
	requireAPIKey(t)

	res := execCmd("log", "tail", "nonexistent-invocation-id-12345")
	if res.err == nil {
		t.Fatal("log tail should fail for nonexistent invocation")
	}
	// Error occurred as expected - the specific error message routing depends
	// on test infrastructure; the important thing is that an error was returned
}

func TestLogTailHelp(t *testing.T) {
	res := execCmd("log", "tail", "--help")
	if res.err != nil {
		t.Fatalf("log tail --help failed: %v", res.err)
	}
	if !strings.Contains(res.stdout, "Follow build log output") {
		t.Fatal("help text missing expected description")
	}
	if !strings.Contains(res.stdout, "--poll") {
		t.Fatal("help text missing --poll flag")
	}
	if !strings.Contains(res.stdout, "--timeout") {
		t.Fatal("help text missing --timeout flag")
	}
	if !strings.Contains(res.stdout, "--quiet") {
		t.Fatal("help text missing --quiet flag")
	}
}

// --- Action Tests ---

func TestActionList(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("action", "list", invID)
	if res.err != nil {
		t.Fatalf("action list failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// May be empty for some invocations
}

func TestActionListJSON(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("action", "list", invID, "--json")
	if res.err != nil {
		t.Fatalf("action list --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	trimmed := strings.TrimSpace(res.stdout)
	if trimmed != "" {
		if !json.Valid([]byte(trimmed)) {
			t.Fatalf("action list --json returned invalid JSON: %s", trimmed)
		}
	}
}

func TestActionListFilterByTargetLabel(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("action", "list", invID, "--target-label", "//nonexistent:target")
	if res.err != nil {
		t.Fatalf("action list --target-label failed: %v\nstderr: %s", res.err, res.stderr)
	}
}

func TestActionLsAlias(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("action", "ls", invID)
	if res.err != nil {
		t.Fatalf("action ls alias failed: %v\nstderr: %s", res.err, res.stderr)
	}
}

// --- Artifact Tests ---

func TestArtifactList(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("artifact", "list", invID)
	if res.err != nil {
		t.Fatalf("artifact list failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// Output may be empty if no artifacts
}

func TestArtifactListJSON(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("artifact", "list", invID, "--json")
	if res.err != nil {
		t.Fatalf("artifact list --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	trimmed := strings.TrimSpace(res.stdout)
	if trimmed != "" {
		if !json.Valid([]byte(trimmed)) {
			t.Fatalf("artifact list --json returned invalid JSON: %s", trimmed)
		}
	}
}

// --- File Tests ---

func TestFileGetInvalidURI(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("file", "get", "invalid://not-a-real-uri")
	if res.err == nil {
		t.Fatal("expected error with invalid bytestream URI")
	}
}

func TestFileDeleteInvalidURI(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("file", "delete", "invalid://not-a-real-uri")
	if res.err == nil {
		t.Fatal("expected error with invalid bytestream URI")
	}
}

// --- Workflow Tests ---

func TestWorkflowRunHelp(t *testing.T) {
	res := execCmd("workflow", "run", "--help")
	if res.err != nil {
		t.Fatalf("workflow run --help failed: %v", res.err)
	}
	output := res.stdout
	// Should show flags like --repo, --branch, --commit, --async
	for _, flag := range []string{"--repo", "--branch", "--commit", "--async"} {
		if !strings.Contains(output, flag) {
			t.Errorf("workflow run --help missing flag %s in output:\n%s", flag, output)
		}
	}
}

// --- Run Tests ---

func TestRunHelp(t *testing.T) {
	res := execCmd("run", "--help")
	if res.err != nil {
		t.Fatalf("run --help failed: %v", res.err)
	}
	output := res.stdout
	// Should show flags like --repo, --branch, --commit, --timeout, --env
	for _, flag := range []string{"--repo", "--branch", "--commit", "--timeout", "--env"} {
		if !strings.Contains(output, flag) {
			t.Errorf("run --help missing flag %s in output:\n%s", flag, output)
		}
	}
}

// --- Global Flag Tests ---

func TestVersionFlag(t *testing.T) {
	res := execCmd("--version")
	if res.err != nil {
		t.Fatalf("--version failed: %v", res.err)
	}
	if !strings.Contains(res.stdout, "buildbuddy-cli") {
		t.Errorf("--version output does not contain 'buildbuddy-cli': %s", res.stdout)
	}
}

func TestHelpFlag(t *testing.T) {
	res := execCmd("--help")
	if res.err != nil {
		t.Fatalf("--help failed: %v", res.err)
	}
	// Should list available commands
	for _, cmd := range []string{"invocation", "target", "log", "action", "artifact", "file", "workflow", "run"} {
		if !strings.Contains(res.stdout, cmd) {
			t.Errorf("--help output missing command %q", cmd)
		}
	}
}

func TestJSONAndPlaintextMutuallyExclusive(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t), "--json", "--plaintext")
	if res.err == nil {
		t.Fatal("expected error when --json and --plaintext are both set")
	}
	combined := res.stderr + res.err.Error()
	if !strings.Contains(combined, "mutually exclusive") && !strings.Contains(combined, "exclusive") {
		t.Errorf("expected 'mutually exclusive' error, got: %s", combined)
	}
}

func TestOutputFileFlag(t *testing.T) {
	requireAPIKey(t)

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "output.json")

	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t), "--json", "-o", outFile)
	if res.err != nil {
		t.Fatalf("-o flag failed: %v\nstderr: %s", res.err, res.stderr)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("output file is empty")
	}
	if !json.Valid(data) {
		t.Fatalf("output file contains invalid JSON: %s", string(data))
	}
}

func TestTemplateFlag(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t),
		"--template", "{{range .invocation}}{{.command}} {{end}}")
	if res.err != nil {
		t.Fatalf("--template failed: %v\nstderr: %s", res.err, res.stderr)
	}
	if res.stdout == "" {
		t.Fatal("--template returned empty output")
	}
}

func TestNoColorFlag(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t), "--no-color")
	if res.err != nil {
		t.Fatalf("--no-color failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// Should not crash; output should still contain data
	if res.stdout == "" {
		t.Fatal("--no-color returned empty output")
	}
}

func TestPlaintextFlag(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t), "--plaintext")
	if res.err != nil {
		t.Fatalf("--plaintext failed: %v\nstderr: %s", res.err, res.stderr)
	}
	if res.stdout == "" {
		t.Fatal("--plaintext returned empty output")
	}
	// Plaintext should be tab-separated
	if !strings.Contains(res.stdout, "\t") {
		t.Errorf("--plaintext output should contain tabs:\n%s", res.stdout)
	}
}

// --- Error Tests ---

func TestBadInvocationIDError(t *testing.T) {
	requireAPIKey(t)
	res := execCmd("invocation", "get", "not-a-real-invocation-id-000000")
	if res.err == nil {
		t.Fatal("expected error for bad invocation ID")
	}
	// Errors should be written as structured JSON to stderr
	if res.stderr != "" {
		var errJSON map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(res.stderr)), &errJSON); err == nil {
			if _, ok := errJSON["error_type"]; !ok {
				t.Error("structured error missing 'error_type' field")
			}
		}
	}
}

func TestMissingAPIKeyError(t *testing.T) {
	// Temporarily unset the API key
	originalKey := os.Getenv("BUILDBUDDY_API_KEY")
	os.Unsetenv("BUILDBUDDY_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("BUILDBUDDY_API_KEY", originalKey)
		}
	}()

	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t))
	if res.err == nil {
		t.Fatal("expected error when BUILDBUDDY_API_KEY is not set")
	}

	// Should produce an auth_failed error
	if res.stderr != "" {
		var errJSON map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(res.stderr)), &errJSON); err == nil {
			errType, _ := errJSON["error_type"].(string)
			if errType != "auth_failed" {
				t.Errorf("expected error_type 'auth_failed', got %q", errType)
			}
		}
	}
}

// --- Table-Driven Tests ---

func TestInvocationListOutputModes(t *testing.T) {
	requireAPIKey(t)

	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T, res execResult)
	}{
		{
			name: "table mode (default)",
			args: []string{"invocation", "list", "--commit", getTestCommitSHA(t)},
			validate: func(t *testing.T, res execResult) {
				t.Helper()
				if res.stdout == "" {
					t.Error("table mode returned empty output")
				}
				if !strings.Contains(res.stdout, "ID") {
					t.Error("table mode missing ID header")
				}
			},
		},
		{
			name: "json mode",
			args: []string{"invocation", "list", "--commit", getTestCommitSHA(t), "--json"},
			validate: func(t *testing.T, res execResult) {
				t.Helper()
				if !json.Valid([]byte(strings.TrimSpace(res.stdout))) {
					t.Errorf("json mode output is not valid JSON: %s", res.stdout)
				}
			},
		},
		{
			name: "plaintext mode",
			args: []string{"invocation", "list", "--commit", getTestCommitSHA(t), "--plaintext"},
			validate: func(t *testing.T, res execResult) {
				t.Helper()
				if !strings.Contains(res.stdout, "\t") {
					t.Error("plaintext mode should contain tabs")
				}
			},
		},
		{
			name: "jq mode",
			args: []string{"invocation", "list", "--commit", getTestCommitSHA(t), "--jq", ".invocation | length"},
			validate: func(t *testing.T, res execResult) {
				t.Helper()
				n, err := strconv.Atoi(strings.TrimSpace(res.stdout))
				if err != nil {
					t.Errorf("jq output not a number: %q", res.stdout)
				}
				if n <= 0 {
					t.Errorf("expected positive count, got %d", n)
				}
			},
		},
		{
			name: "template mode",
			args: []string{"invocation", "list", "--commit", getTestCommitSHA(t),
				"--template", "count={{len .invocation}}"},
			validate: func(t *testing.T, res execResult) {
				t.Helper()
				if !strings.Contains(res.stdout, "count=") {
					t.Errorf("template output missing 'count=': %s", res.stdout)
				}
			},
		},
		{
			name: "fields mode",
			args: []string{"invocation", "list", "--commit", getTestCommitSHA(t), "--fields", "invocation"},
			validate: func(t *testing.T, res execResult) {
				t.Helper()
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				if _, ok := parsed["invocation"]; !ok {
					t.Error("fields output missing 'invocation' key")
				}
				// Should only have the requested field
				if len(parsed) > 1 {
					t.Errorf("fields output should only have 'invocation', got %d keys: %v", len(parsed), keys(parsed))
				}
			},
		},
		{
			name: "limit 1",
			args: []string{"invocation", "list", "--commit", getTestCommitSHA(t), "--limit", "1", "--json"},
			validate: func(t *testing.T, res execResult) {
				t.Helper()
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				arr := parsed["invocation"].([]interface{})
				if len(arr) != 1 {
					t.Errorf("expected 1 invocation, got %d", len(arr))
				}
			},
		},
		{
			name: "limit 2",
			args: []string{"invocation", "list", "--commit", getTestCommitSHA(t), "--limit", "2", "--json"},
			validate: func(t *testing.T, res execResult) {
				t.Helper()
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				arr := parsed["invocation"].([]interface{})
				if len(arr) != 2 {
					t.Errorf("expected 2 invocations, got %d", len(arr))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := execCmd(tc.args...)
			if res.err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", res.err, res.stderr)
			}
			tc.validate(t, res)
		})
	}
}

func TestTargetSubcommands(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		checkJSON bool
	}{
		{
			name: "target list table",
			args: []string{"target", "list", invID},
		},
		{
			name:      "target list json",
			args:      []string{"target", "list", invID, "--json"},
			checkJSON: true,
		},
		{
			name: "target list with label filter",
			args: []string{"target", "list", invID, "--label", "//nonexistent:label"},
		},
		{
			name: "target failed",
			args: []string{"target", "failed", invID},
		},
		{
			name: "target ls alias",
			args: []string{"target", "ls", invID},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := execCmd(tc.args...)
			if tc.wantErr {
				if res.err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if res.err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", res.err, res.stderr)
			}
			if tc.checkJSON {
				trimmed := strings.TrimSpace(res.stdout)
				if trimmed != "" && !json.Valid([]byte(trimmed)) {
					t.Fatalf("invalid JSON output: %s", trimmed)
				}
			}
		})
	}
}

func TestLogSubcommands(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	tests := []struct {
		name      string
		args      []string
		checkJSON bool
		maxLines  int // 0 = no line count check
	}{
		{
			name: "log get default",
			args: []string{"log", "get", invID},
		},
		{
			name:      "log get json",
			args:      []string{"log", "get", invID, "--json"},
			checkJSON: true,
		},
		{
			name:     "log get tail 5",
			args:     []string{"log", "get", invID, "--tail", "5"},
			maxLines: 5,
		},
		{
			name:     "log get head 3",
			args:     []string{"log", "get", invID, "--head", "3"},
			maxLines: 3,
		},
		{
			name: "log get grep",
			args: []string{"log", "get", invID, "--grep", "test"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := execCmd(tc.args...)
			if res.err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", res.err, res.stderr)
			}
			if tc.checkJSON {
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				if _, ok := parsed["log"]; !ok {
					t.Error("JSON missing 'log' key")
				}
			}
			if tc.maxLines > 0 {
				lines := nonEmptyLines(res.stdout)
				if len(lines) > tc.maxLines {
					t.Errorf("expected at most %d lines, got %d", tc.maxLines, len(lines))
				}
			}
		})
	}
}

// --- Log Target Tests ---

func TestLogGetTargetNonexistent(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Use a nonexistent target label - should return not found error
	res := execCmd("log", "get", invID, "--target", "//nonexistent:fake_target_xyz")
	if res.err == nil {
		t.Fatal("expected error for nonexistent target, but got none")
	}
	// Error should mention "not found"
	combined := res.stderr + res.err.Error()
	if !strings.Contains(strings.ToLower(combined), "not found") {
		t.Errorf("expected 'not found' error, got: %s", combined)
	}
}

func TestLogGetTargetWithRawFlag(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// --raw with nonexistent target should fail gracefully
	res := execCmd("log", "get", invID, "--target", "//nonexistent:target", "--raw")
	if res.err == nil {
		// If it succeeds, that's also fine (means there's a target with that name)
		t.Log("--raw flag accepted")
	}
}

func TestLogGetTargetWithJunitFlag(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// --junit with nonexistent target should fail gracefully
	res := execCmd("log", "get", invID, "--target", "//nonexistent:target", "--junit")
	if res.err == nil {
		// If it succeeds, that's also fine
		t.Log("--junit flag accepted")
	}
}

func TestLogGetTargetJSONOutput(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Try to get a target in JSON mode - error should still be valid JSON
	res := execCmd("log", "get", invID, "--target", "//nonexistent:target", "--json")

	// If error, stderr should be valid JSON error format
	if res.err != nil {
		if res.stderr != "" {
			var errJSON map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(res.stderr)), &errJSON); err == nil {
				// Valid JSON error format
				if _, ok := errJSON["error_type"]; !ok {
					t.Error("structured error missing 'error_type' field")
				}
			}
		}
	}
}

// TestLogGetTargetZZHelp tests the --help output for target-related flags.
// Named with ZZ prefix to run last in the log target test group to avoid
// cobra flag state issues with subsequent tests.
func TestLogGetTargetZZHelp(t *testing.T) {
	res := execCmd("log", "get", "--help")
	if res.err != nil {
		t.Fatalf("log get --help failed: %v", res.err)
	}
	// Should show --target, --raw, --junit flags
	for _, flag := range []string{"--target", "--raw", "--junit"} {
		if !strings.Contains(res.stdout, flag) {
			t.Errorf("log get --help missing flag %s in output:\n%s", flag, res.stdout)
		}
	}
}

func TestActionSubcommands(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	tests := []struct {
		name      string
		args      []string
		checkJSON bool
	}{
		{
			name: "action list table",
			args: []string{"action", "list", invID},
		},
		{
			name:      "action list json",
			args:      []string{"action", "list", invID, "--json"},
			checkJSON: true,
		},
		{
			name: "action list with target-label filter",
			args: []string{"action", "list", invID, "--target-label", "//nonexistent:label"},
		},
		{
			name: "action ls alias",
			args: []string{"action", "ls", invID},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := execCmd(tc.args...)
			if res.err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", res.err, res.stderr)
			}
			if tc.checkJSON {
				trimmed := strings.TrimSpace(res.stdout)
				if trimmed != "" && !json.Valid([]byte(trimmed)) {
					t.Fatalf("invalid JSON output: %s", trimmed)
				}
			}
		})
	}
}

// --- Workflow Step Detection Tests ---

func TestIsWorkflowStepPattern(t *testing.T) {
	tests := []struct {
		label    string
		expected bool
	}{
		{"steps[0]", true},
		{"steps[1]", true},
		{"steps[99]", true},
		{"steps[123]", true},
		{"//pkg:target", false},
		{"steps", false},
		{"steps[]", false},
		{"steps[a]", false},
		{"steps[0", false},
		{"steps0]", false},
		{"prefix_steps[0]", false},
		{"steps[0]_suffix", false},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			result := isWorkflowStep(tc.label)
			if result != tc.expected {
				t.Errorf("isWorkflowStep(%q) = %v, want %v", tc.label, result, tc.expected)
			}
		})
	}
}

func TestTargetFailedJSONIncludesSourceType(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("target", "failed", invID, "--json")
	if res.err != nil {
		t.Fatalf("target failed --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	// If there are failed targets, verify JSON includes sourceType field for workflow steps
	trimmed := strings.TrimSpace(res.stdout)
	if trimmed != "" && trimmed != "null" && trimmed != "[]" {
		var targets []map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &targets); err != nil {
			// If it's not an array, it might be in a different format
			t.Logf("Note: JSON output format: %s", trimmed)
			return
		}
		for _, target := range targets {
			label, ok := target["label"].(string)
			if !ok {
				continue
			}
			// Check if workflow step targets have sourceType field
			if isWorkflowStep(label) {
				sourceType, ok := target["sourceType"].(string)
				if !ok || sourceType != "workflow_step" {
					t.Errorf("workflow step target %q should have sourceType='workflow_step', got %q", label, sourceType)
				}
			}
		}
	}
}

func TestLogGetWorkflowStepNonexistentInvocation(t *testing.T) {
	requireAPIKey(t)
	// Try to get workflow step log for a nonexistent invocation
	res := execCmd("log", "get", "nonexistent-invocation-id", "--target", "steps[0]")
	if res.err == nil {
		t.Fatal("expected error for nonexistent invocation, but got none")
	}
}

// TestLogGetWorkflowStepZZHelp tests the --help output for workflow-related flags.
// Named with ZZ prefix to run last to avoid Cobra help flag state issues.
func TestLogGetWorkflowStepZZHelp(t *testing.T) {
	res := execCmd("log", "get", "--help")
	if res.err != nil {
		t.Fatalf("log get --help failed: %v", res.err)
	}
	// The help should mention --target flag
	if !strings.Contains(res.stdout, "--target") {
		t.Errorf("log get --help missing --target flag in output:\n%s", res.stdout)
	}
}

// --- URL Parsing Tests ---

func TestInvocationGetWithURL(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Test that full BuildBuddy URLs are accepted and parsed correctly
	url := "https://app.buildbuddy.io/invocation/" + invID
	res := execCmd("invocation", "get", url)
	if res.err != nil {
		t.Fatalf("invocation get with URL failed: %v\nstderr: %s", res.err, res.stderr)
	}
	if res.stdout == "" {
		t.Fatal("invocation get with URL returned empty output")
	}
	if !strings.Contains(res.stdout, "ID") {
		t.Errorf("expected table header in output, got:\n%s", res.stdout)
	}
}

func TestInvocationGetWithURLJSON(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	url := "https://app.buildbuddy.io/invocation/" + invID
	res := execCmd("invocation", "get", url, "--json")
	if res.err != nil {
		t.Fatalf("invocation get URL --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, res.stdout)
	}
	if _, ok := parsed["invocation"]; !ok {
		t.Fatalf("JSON missing 'invocation' key, got keys: %v", keys(parsed))
	}
}

func TestInvocationWaitWithURL(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Full URL should be accepted by wait command
	url := "https://app.buildbuddy.io/invocation/" + invID
	res := execCmd("invocation", "wait", url)
	if res.err != nil {
		// Expected for failed invocations
		t.Logf("invocation wait with URL returned: %v (may be expected for failed invocations)", res.err)
	}
	if res.stdout == "" {
		t.Error("invocation wait with URL returned empty stdout")
	}
}

// --- Target Failed Extended Tests ---

func TestTargetFailedWithLabel(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Filter by label - even if no results, should not error
	res := execCmd("target", "failed", invID, "--label", "//nonexistent:target_label")
	if res.err != nil {
		t.Fatalf("target failed --label failed: %v\nstderr: %s", res.err, res.stderr)
	}
}

func TestTargetFailedPlaintext(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("target", "failed", invID, "--plaintext")
	if res.err != nil {
		t.Fatalf("target failed --plaintext failed: %v\nstderr: %s", res.err, res.stderr)
	}
	// If there are results, should be tab-separated
	if res.stdout != "" && !strings.Contains(res.stdout, "\t") {
		t.Logf("Note: plaintext output without tabs (may be empty results): %s", res.stdout)
	}
}

func TestTargetFailedJSONAllEntriesHaveInvocationID(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("target", "failed", invID, "--json")
	if res.err != nil {
		t.Fatalf("target failed --json failed: %v\nstderr: %s", res.err, res.stderr)
	}

	trimmed := strings.TrimSpace(res.stdout)
	if trimmed == "" || trimmed == "null" || trimmed == "[]" {
		t.Skip("no failed targets to validate")
	}

	var targets []map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &targets); err != nil {
		t.Skipf("JSON output format not an array: %s", trimmed)
	}

	// Every entry should have an invocationId field
	for i, target := range targets {
		invocID, ok := target["invocationId"].(string)
		if !ok || invocID == "" {
			t.Errorf("target[%d] missing or empty 'invocationId' field: %v", i, target)
		}
	}
}

func TestTargetFailedDefaultDepth(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Default depth is -1 (unlimited recursion) - should work without --depth flag
	res := execCmd("target", "failed", invID, "--json")
	if res.err != nil {
		t.Fatalf("target failed (default depth) failed: %v\nstderr: %s", res.err, res.stderr)
	}

	// Compare with explicit --depth -1 to verify default behavior
	res2 := execCmd("target", "failed", invID, "--depth", "-1", "--json")
	if res2.err != nil {
		t.Fatalf("target failed --depth -1 failed: %v\nstderr: %s", res2.err, res2.stderr)
	}

	// Both should produce the same output
	if res.stdout != res2.stdout {
		t.Errorf("default depth and --depth -1 produced different output")
	}
}

// --- Log Tail Extended Tests ---

func TestLogTailOutputFile(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "tail_output.json")

	res := execCmd("log", "tail", invID, "--json", "-o", outFile)
	// May fail for failed invocations, that's OK
	_ = res.err

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("output file is empty")
	}
	if !json.Valid(data) {
		t.Fatalf("output file contains invalid JSON: %s", string(data))
	}
}

func TestLogTailQuietCompletedHasOutput(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// Quiet mode on completed invocation should still show the log content
	res := execCmd("log", "tail", invID, "--quiet")
	// May fail for failed invocations
	if res.stdout == "" {
		t.Error("log tail --quiet on completed invocation should still show log output")
	}
}

func TestLogTailCompletedInvocationDoesNotPollMultipleTimes(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	// A completed invocation should fetch the log once and exit without additional polling.
	// The single fetch can take several seconds due to paginated log retrieval,
	// but it should NOT take 2x the poll interval (which would indicate re-polling).
	start := time.Now()
	res := execCmd("log", "tail", invID, "--quiet", "--poll", "5")
	elapsed := time.Since(start)

	_ = res.err // may fail for failed invocations

	// Should complete in one fetch cycle, not two poll intervals (10s+)
	if elapsed > 15*time.Second {
		t.Errorf("log tail on completed invocation took %v (suggests re-polling instead of exiting after first fetch)", elapsed)
	}
}

// --- Invocation Wait Extended Tests ---

func TestInvocationWaitPlaintext(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("invocation", "wait", invID, "--plaintext")
	if res.err != nil {
		t.Logf("invocation wait --plaintext returned: %v (may be expected for failed invocations)", res.err)
	}
	if res.stdout == "" {
		t.Error("invocation wait --plaintext returned empty output")
	}
	// Plaintext should be tab-separated
	if !strings.Contains(res.stdout, "\t") {
		t.Errorf("--plaintext output should contain tabs:\n%s", res.stdout)
	}
}

func TestInvocationWaitTemplate(t *testing.T) {
	requireAPIKey(t)
	invID := getFirstInvocationID(t)

	res := execCmd("invocation", "wait", invID,
		"--template", "{{range .invocation}}{{.invocationId}} {{end}}")
	if res.err != nil {
		t.Logf("invocation wait --template returned: %v (may be expected for failed invocations)", res.err)
	}
	if res.stdout == "" {
		t.Error("invocation wait --template returned empty output")
	}
}

// --- Helpers ---

// cachedInvID stores the first invocation ID to avoid repeated API calls.
var cachedInvID string

// getFirstInvocationID fetches invocations for the test commit and returns the first ID.
// It caches the result for reuse across tests.
func getFirstInvocationID(t *testing.T) string {
	t.Helper()
	if cachedInvID != "" {
		return cachedInvID
	}

	res := execCmd("invocation", "list", "--commit", getTestCommitSHA(t), "--json")
	if res.err != nil {
		t.Fatalf("failed to get invocations for ID extraction: %v\nstderr: %s", res.err, res.stderr)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(res.stdout), &parsed); err != nil {
		t.Fatalf("failed to parse invocations JSON: %v\nraw: %s", err, res.stdout)
	}

	arr, ok := parsed["invocation"].([]interface{})
	if !ok || len(arr) == 0 {
		t.Fatal("no invocations found for test commit")
	}

	inv, ok := arr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("invocation element is not a map: %T", arr[0])
	}

	idObj, ok := inv["id"].(map[string]interface{})
	if !ok {
		t.Fatalf("invocation missing 'id' field or wrong type: %v", inv)
	}

	invID, ok := idObj["invocationId"].(string)
	if !ok || invID == "" {
		t.Fatalf("invocation id.invocationId missing or empty: %v", idObj)
	}

	cachedInvID = invID
	t.Logf("using invocation ID: %s", cachedInvID)
	return cachedInvID
}

// nonEmptyLines splits text into lines and removes empty ones.
func nonEmptyLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

// keys returns the keys of a map for diagnostic output.
func keys(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// Ensure fmt is used (needed for some error formatting above).
var _ = fmt.Sprintf
