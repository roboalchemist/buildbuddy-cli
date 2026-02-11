package cmd

import (
	"fmt"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "Query build actions",
}

var actionListCmd = &cobra.Command{
	Use:     "list <invocation-id>",
	Aliases: []string{"ls"},
	Short:   "List all actions for an invocation",
	Args:    cobra.ExactArgs(1),
	RunE:    runActionList,
}

var (
	flagActionTargetLabel string
	flagActionTargetID    string
)

func init() {
	actionListCmd.Flags().StringVar(&flagActionTargetLabel, "target-label", "", "Filter by target label")
	actionListCmd.Flags().StringVar(&flagActionTargetID, "target-id", "", "Filter by target ID")

	actionCmd.AddCommand(actionListCmd)
	rootCmd.AddCommand(actionCmd)
}

func runActionList(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	req := &api.GetActionRequest{
		Selector: &api.ActionSelector{
			InvocationID: args[0],
			TargetLabel:  flagActionTargetLabel,
			TargetID:     flagActionTargetID,
		},
	}

	var allActions []api.Action
	var resp api.GetActionResponse
	if err := client.CallAllPages("GetAction", req, &resp, func() {
		allActions = append(allActions, resp.Action...)
	}); err != nil {
		return output.NewAPIError(fmt.Sprintf("GetAction: %v", err))
	}

	opts := GetOutputOptions()
	if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
		td := actionsToTable(allActions)
		return output.RenderTable(td, allActions, opts)
	}
	return output.Render(allActions, opts)
}

func actionsToTable(actions []api.Action) output.TableData {
	td := output.TableData{
		Headers: []string{"TARGET", "SHARD", "RUN", "ATTEMPT", "FILES"},
	}
	for _, a := range actions {
		fileCount := fmt.Sprintf("%d", len(a.File))

		td.Rows = append(td.Rows, []string{
			truncateStr(a.TargetLabel, 60),
			fmt.Sprintf("%d", a.Shard.Int64()),
			fmt.Sprintf("%d", a.Run.Int64()),
			fmt.Sprintf("%d", a.Attempt.Int64()),
			fileCount,
		})
	}
	return td
}
