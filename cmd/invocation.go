package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

var invocationCmd = &cobra.Command{
	Use:     "invocation",
	Short:   "Query build invocations",
	Aliases: []string{"inv"},
}

var invocationGetCmd = &cobra.Command{
	Use:     "get <invocation-id>",
	Aliases: []string{"show"},
	Short:   "Get a specific invocation by ID",
	Args:    cobra.ExactArgs(1),
	RunE:    runInvocationGet,
}

var invocationListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List invocations by commit SHA",
	RunE:    runInvocationList,
}

var invocationWaitCmd = &cobra.Command{
	Use:     "wait <invocation-id>",
	Aliases: []string{"watch"},
	Short:   "Wait for an invocation to complete",
	Long: `Wait for an invocation to complete by polling GetInvocation.

Polls the invocation status until it is no longer in progress (PARTIAL_INVOCATION_STATUS).
Prints status updates to stderr in table mode, and the final invocation to stdout.

Exit codes:
  0 - Invocation completed successfully
  1 - Invocation completed with failure
  2 - Timeout reached before completion`,
	Args: cobra.ExactArgs(1),
	RunE: runInvocationWait,
}

var (
	flagInvListCommit string
	flagInvListLimit  int

	// Wait command flags
	flagWaitPoll    int
	flagWaitTimeout int
)

func init() {
	invocationListCmd.Flags().StringVar(&flagInvListCommit, "commit", "", "Filter by commit SHA")
	invocationListCmd.Flags().IntVarP(&flagInvListLimit, "limit", "n", 0, "Max results to return (0 = all)")

	invocationWaitCmd.Flags().IntVar(&flagWaitPoll, "poll", 10, "Polling interval in seconds")
	invocationWaitCmd.Flags().IntVar(&flagWaitTimeout, "timeout", 0, "Max wait time in seconds (0 = unlimited)")

	invocationCmd.AddCommand(invocationGetCmd)
	invocationCmd.AddCommand(invocationListCmd)
	invocationCmd.AddCommand(invocationWaitCmd)
	rootCmd.AddCommand(invocationCmd)
}

func runInvocationGet(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	invocationID := args[0]
	if strings.Contains(invocationID, "/invocation/") {
		parts := strings.Split(invocationID, "/invocation/")
		invocationID = parts[len(parts)-1]
	}

	req := &api.GetInvocationRequest{
		Selector: &api.InvocationSelector{
			InvocationID: invocationID,
		},
		IncludeMetadata: true,
	}

	var resp api.GetInvocationResponse
	if err := client.Call("GetInvocation", req, &resp); err != nil {
		return output.NewAPIError(fmt.Sprintf("GetInvocation: %v", err))
	}

	opts := GetOutputOptions()
	if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
		if len(resp.Invocation) == 0 {
			return output.NewNotFoundError(fmt.Sprintf("invocation %s not found", invocationID))
		}
		td := invocationsToTable(resp.Invocation)
		return output.RenderTable(td, resp, opts)
	}
	return output.Render(resp, opts)
}

func runInvocationList(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	if flagInvListCommit == "" {
		return output.NewUsageError("--commit is required for invocation list")
	}

	req := &api.GetInvocationRequest{
		Selector: &api.InvocationSelector{
			CommitSHA: flagInvListCommit,
		},
		IncludeMetadata: true,
	}

	var allInvocations []api.Invocation
	var resp api.GetInvocationResponse
	if err := client.CallAllPages("GetInvocation", req, &resp, func() {
		allInvocations = append(allInvocations, resp.Invocation...)
	}); err != nil {
		return output.NewAPIError(fmt.Sprintf("GetInvocation: %v", err))
	}

	if flagInvListLimit > 0 && flagInvListLimit < len(allInvocations) {
		allInvocations = allInvocations[:flagInvListLimit]
	}

	opts := GetOutputOptions()
	result := map[string]interface{}{"invocation": allInvocations}
	if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
		td := invocationsToTable(allInvocations)
		return output.RenderTable(td, result, opts)
	}
	return output.Render(result, opts)
}

func invocationsToTable(invocations []api.Invocation) output.TableData {
	td := output.TableData{
		Headers: []string{"ID", "STATUS", "DURATION", "COMMAND", "BRANCH", "USER"},
	}
	for _, inv := range invocations {
		status := "PASS"
		if !inv.Success {
			status = "FAIL"
		}
		if inv.InvocationStatus == "PARTIAL_INVOCATION_STATUS" {
			status = "IN_PROGRESS"
		}
		if inv.InvocationStatus == "DISCONNECTED_INVOCATION_STATUS" {
			status = "DISCONNECTED"
		}

		td.Rows = append(td.Rows, []string{
			truncateStr(inv.GetInvocationID(), 36),
			status,
			formatDuration(inv.DurationUsec.Int64()),
			truncateStr(inv.Command, 20),
			truncateStr(inv.BranchName, 30),
			truncateStr(inv.User, 20),
		})
	}
	return td
}

func runInvocationWait(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	invocationID := args[0]
	if strings.Contains(invocationID, "/invocation/") {
		parts := strings.Split(invocationID, "/invocation/")
		invocationID = parts[len(parts)-1]
	}

	pollInterval := time.Duration(flagWaitPoll) * time.Second
	var deadline time.Time
	if flagWaitTimeout > 0 {
		deadline = time.Now().Add(time.Duration(flagWaitTimeout) * time.Second)
	}

	opts := GetOutputOptions()
	isJSON := opts.Mode == output.ModeJSON

	for {
		// Check timeout
		if flagWaitTimeout > 0 && time.Now().After(deadline) {
			if !isJSON {
				fmt.Fprintf(os.Stderr, "Timeout reached after %d seconds\n", flagWaitTimeout)
			}
			return output.NewTimeoutError(fmt.Sprintf("timeout after %d seconds", flagWaitTimeout))
		}

		// Fetch invocation status
		req := &api.GetInvocationRequest{
			Selector: &api.InvocationSelector{
				InvocationID: invocationID,
			},
			IncludeMetadata: true,
		}

		var resp api.GetInvocationResponse
		if err := client.Call("GetInvocation", req, &resp); err != nil {
			return output.NewAPIError(fmt.Sprintf("GetInvocation: %v", err))
		}

		if len(resp.Invocation) == 0 {
			return output.NewNotFoundError(fmt.Sprintf("invocation %s not found", invocationID))
		}

		inv := resp.Invocation[0]

		// Check if still in progress
		if inv.InvocationStatus == "PARTIAL_INVOCATION_STATUS" {
			if !isJSON {
				fmt.Fprintf(os.Stderr, "[%s] Invocation %s is still in progress (%s)...\n",
					time.Now().Format("15:04:05"),
					truncateStr(invocationID, 20),
					formatDuration(inv.DurationUsec.Int64()))
			}
			time.Sleep(pollInterval)
			continue
		}

		// Invocation is complete - output final result
		if isJSON {
			if renderErr := output.Render(resp, opts); renderErr != nil {
				return renderErr
			}
		} else {
			// Table mode output
			td := invocationsToTable(resp.Invocation)
			if renderErr := output.RenderTable(td, resp, opts); renderErr != nil {
				return renderErr
			}
		}

		// Return appropriate error for exit code (nil for success)
		if inv.Success {
			return nil
		}
		return output.NewInvocationFailedError(fmt.Sprintf("invocation %s failed", invocationID))
	}
}
