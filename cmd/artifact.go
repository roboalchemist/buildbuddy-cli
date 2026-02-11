package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

var artifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Query test artifacts",
}

var artifactListCmd = &cobra.Command{
	Use:     "list <invocation-id>",
	Aliases: []string{"ls"},
	Short:   "List all artifact files for an invocation",
	Args:    cobra.ExactArgs(1),
	RunE:    runArtifactList,
}

var artifactGetCmd = &cobra.Command{
	Use:   "get <invocation-id>",
	Short: "Download test artifacts for an invocation",
	Args:  cobra.ExactArgs(1),
	RunE:  runArtifactGet,
}

var (
	flagArtifactDir         string
	flagArtifactTargetLabel string
)

func init() {
	artifactGetCmd.Flags().StringVar(&flagArtifactDir, "dir", "", "Directory to save artifacts to")
	artifactGetCmd.Flags().StringVar(&flagArtifactTargetLabel, "target-label", "", "Filter by target label")
	artifactListCmd.Flags().StringVar(&flagArtifactTargetLabel, "target-label", "", "Filter by target label")

	artifactCmd.AddCommand(artifactListCmd)
	artifactCmd.AddCommand(artifactGetCmd)
	rootCmd.AddCommand(artifactCmd)
}

func fetchArtifactFiles(client *api.Client, invocationID string) ([]artifactFile, error) {
	req := &api.GetActionRequest{
		Selector: &api.ActionSelector{
			InvocationID: invocationID,
			TargetLabel:  flagArtifactTargetLabel,
		},
	}

	var allActions []api.Action
	var resp api.GetActionResponse
	if err := client.CallAllPages("GetAction", req, &resp, func() {
		allActions = append(allActions, resp.Action...)
	}); err != nil {
		return nil, err
	}

	var files []artifactFile
	for _, a := range allActions {
		for _, f := range a.File {
			files = append(files, artifactFile{
				TargetLabel: a.TargetLabel,
				Name:        f.Name,
				URI:         f.URI,
				SizeBytes:   f.SizeBytes.Int64(),
			})
		}
	}
	return files, nil
}

type artifactFile struct {
	TargetLabel string `json:"targetLabel,omitempty"`
	Name        string `json:"name,omitempty"`
	URI         string `json:"uri,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
}

func runArtifactList(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	files, err := fetchArtifactFiles(client, args[0])
	if err != nil {
		return output.NewAPIError(fmt.Sprintf("GetAction: %v", err))
	}

	opts := GetOutputOptions()
	if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
		td := output.TableData{
			Headers: []string{"TARGET", "FILE", "SIZE", "URI"},
		}
		for _, f := range files {
			td.Rows = append(td.Rows, []string{
				truncateStr(f.TargetLabel, 50),
				f.Name,
				output.HumanSize(f.SizeBytes),
				truncateStr(f.URI, 60),
			})
		}
		return output.RenderTable(td, files, opts)
	}
	return output.Render(files, opts)
}

func runArtifactGet(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	files, err := fetchArtifactFiles(client, args[0])
	if err != nil {
		return output.NewAPIError(fmt.Sprintf("GetAction: %v", err))
	}

	if len(files) == 0 {
		return output.NewNotFoundError("no artifacts found for invocation")
	}

	// If no --dir, just list them (same as artifact list)
	if flagArtifactDir == "" {
		opts := GetOutputOptions()
		if opts.Mode == output.ModeTable || opts.Mode == output.ModePlaintext {
			td := output.TableData{
				Headers: []string{"TARGET", "FILE", "SIZE", "URI"},
			}
			for _, f := range files {
				td.Rows = append(td.Rows, []string{
					truncateStr(f.TargetLabel, 50),
					f.Name,
					output.HumanSize(f.SizeBytes),
					truncateStr(f.URI, 60),
				})
			}
			return output.RenderTable(td, files, opts)
		}
		return output.Render(files, opts)
	}

	// Download each file
	if err := os.MkdirAll(flagArtifactDir, 0o755); err != nil {
		return output.NewInternalError(fmt.Sprintf("create dir: %v", err))
	}

	var totalBytes int64
	for _, f := range files {
		if f.URI == "" {
			continue
		}

		req := api.GetFileRequest{URI: f.URI}
		var resp api.GetFileResponse
		if err := client.Call("GetFile", req, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download %s: %v\n", f.Name, err)
			continue
		}

		dest := filepath.Join(flagArtifactDir, f.Name)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create dir for %s: %v\n", f.Name, err)
			continue
		}

		if err := os.WriteFile(dest, resp.Data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", f.Name, err)
			continue
		}

		totalBytes += int64(len(resp.Data))
		fmt.Fprintf(os.Stderr, "  %s (%s)\n", f.Name, output.HumanSize(int64(len(resp.Data))))
	}

	fmt.Fprintf(os.Stderr, "Wrote %d artifacts to %s (%s total)\n", len(files), flagArtifactDir, output.HumanSize(totalBytes))
	return nil
}
