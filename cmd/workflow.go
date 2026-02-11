package cmd

import (
	"fmt"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage workflows",
}

var workflowRunCmd = &cobra.Command{
	Use:   "run <action-name>",
	Short: "Execute a workflow action",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowRun,
}

var (
	flagWFRepo       string
	flagWFBranch     string
	flagWFCommitSHA  string
	flagWFVisibility string
	flagWFAsync      bool
)

func init() {
	workflowRunCmd.Flags().StringVar(&flagWFRepo, "repo", "", "Repository URL")
	workflowRunCmd.Flags().StringVar(&flagWFBranch, "branch", "", "Branch name")
	workflowRunCmd.Flags().StringVar(&flagWFCommitSHA, "commit", "", "Commit SHA")
	workflowRunCmd.Flags().StringVar(&flagWFVisibility, "visibility", "", "Visibility (e.g., PUBLIC)")
	workflowRunCmd.Flags().BoolVar(&flagWFAsync, "async", false, "Run asynchronously")

	workflowCmd.AddCommand(workflowRunCmd)
	rootCmd.AddCommand(workflowCmd)
}

func runWorkflowRun(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	req := api.ExecuteWorkflowRequest{
		ActionNames: []string{args[0]},
		RepoURL:     flagWFRepo,
		Branch:      flagWFBranch,
		CommitSHA:   flagWFCommitSHA,
		Visibility:  flagWFVisibility,
		Async:       flagWFAsync,
	}

	var resp api.ExecuteWorkflowResponse
	if err := client.Call("ExecuteWorkflow", req, &resp); err != nil {
		return output.NewAPIError(fmt.Sprintf("ExecuteWorkflow: %v", err))
	}

	opts := GetOutputOptions()
	if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
		td := output.TableData{
			Headers: []string{"ACTION", "INVOCATION_ID", "STATUS"},
		}
		for _, s := range resp.ActionStatuses {
			statusMsg := "OK"
			if s.Status != nil && s.Status.Code != 0 {
				statusMsg = fmt.Sprintf("ERROR: %s", s.Status.Message)
			}
			td.Rows = append(td.Rows, []string{
				s.ActionName,
				s.InvocationID,
				statusMsg,
			})
		}
		return output.RenderTable(td, resp, opts)
	}
	return output.Render(resp, opts)
}
