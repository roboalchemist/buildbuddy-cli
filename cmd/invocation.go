package cmd

import (
	"fmt"
	"strings"

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

var (
	flagInvListCommit string
	flagInvListLimit  int
)

func init() {
	invocationListCmd.Flags().StringVar(&flagInvListCommit, "commit", "", "Filter by commit SHA")
	invocationListCmd.Flags().IntVarP(&flagInvListLimit, "limit", "n", 0, "Max results to return (0 = all)")

	invocationCmd.AddCommand(invocationGetCmd)
	invocationCmd.AddCommand(invocationListCmd)
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
