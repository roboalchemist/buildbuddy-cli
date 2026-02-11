package cmd

import (
	"fmt"
	"strings"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

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

var (
	flagLogGrep string
	flagLogTail int
	flagLogHead int
)

func init() {
	logGetCmd.Flags().StringVar(&flagLogGrep, "grep", "", "Filter log lines matching pattern")
	logGetCmd.Flags().IntVar(&flagLogTail, "tail", 0, "Show only last N lines")
	logGetCmd.Flags().IntVar(&flagLogHead, "head", 0, "Show only first N lines")

	logCmd.AddCommand(logGetCmd)
	rootCmd.AddCommand(logCmd)
}

func runLogGet(cmd *cobra.Command, args []string) error {
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
