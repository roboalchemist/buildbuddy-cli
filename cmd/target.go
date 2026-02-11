package cmd

import (
	"fmt"
	"strings"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

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
	Short: "Show only failed/flaky targets",
	Args:  cobra.ExactArgs(1),
	RunE:  runTargetFailed,
}

var (
	flagTargetLabel string
	flagTargetTag   string
)

func init() {
	targetListCmd.Flags().StringVar(&flagTargetLabel, "label", "", "Filter by target label")
	targetListCmd.Flags().StringVar(&flagTargetTag, "tag", "", "Filter by target tag")
	targetFailedCmd.Flags().StringVar(&flagTargetLabel, "label", "", "Filter by target label")

	targetCmd.AddCommand(targetListCmd)
	targetCmd.AddCommand(targetFailedCmd)
	rootCmd.AddCommand(targetCmd)
}

func runTargetList(cmd *cobra.Command, args []string) error {
	return fetchTargets(args[0], false)
}

func runTargetFailed(cmd *cobra.Command, args []string) error {
	return fetchTargets(args[0], true)
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
