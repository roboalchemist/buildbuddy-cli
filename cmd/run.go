package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <command>",
	Short: "Run a command on a remote BuildBuddy runner",
	Long:  "Execute a command on a remote BuildBuddy runner. The command is run in a clean workspace with the specified repo checked out.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRunCmd,
}

var (
	flagRunRepo    string
	flagRunBranch  string
	flagRunCommit  string
	flagRunTimeout string
	flagRunEnv     []string
)

func init() {
	runCmd.Flags().StringVar(&flagRunRepo, "repo", "", "Repository URL")
	runCmd.Flags().StringVar(&flagRunBranch, "branch", "", "Branch name")
	runCmd.Flags().StringVar(&flagRunCommit, "commit", "", "Commit SHA")
	runCmd.Flags().StringVar(&flagRunTimeout, "timeout", "", "Timeout duration (e.g., 10m, 1h)")
	runCmd.Flags().StringSliceVarP(&flagRunEnv, "env", "e", nil, "Environment variables (KEY=VALUE, repeatable)")

	rootCmd.AddCommand(runCmd)
}

func runRunCmd(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	// Build steps from args - join all args as a single command
	command := strings.Join(args, " ")

	// Parse env vars
	envMap := make(map[string]string)
	for _, e := range flagRunEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	req := api.RunRequest{
		Repo:      flagRunRepo,
		Branch:    flagRunBranch,
		CommitSHA: flagRunCommit,
		Steps:     []api.Step{{Run: command}},
		Timeout:   flagRunTimeout,
	}
	if len(envMap) > 0 {
		req.Env = envMap
	}

	var resp api.RunResponse
	if err := client.Call("Run", req, &resp); err != nil {
		return output.NewAPIError(fmt.Sprintf("Run: %v", err))
	}

	opts := GetOutputOptions()
	if opts.Mode == output.ModeJSON {
		return output.Render(resp, opts)
	}

	fmt.Fprintf(os.Stderr, "Invocation: %s\n", resp.InvocationID)
	fmt.Println(resp.InvocationID)
	return nil
}
