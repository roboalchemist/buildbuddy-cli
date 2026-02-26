package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

// callGetFileRaw fetches a bytestream URI and returns the raw bytes.
// GetFile returns raw bytes (not JSON), so we use CallRaw to avoid JSON decode errors.
func callGetFileRaw(client *api.Client, uri string) ([]byte, error) {
	req := api.GetFileRequest{URI: uri}
	body, err := client.CallRaw("GetFile", req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	return io.ReadAll(body)
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage cached files",
}

var fileGetCmd = &cobra.Command{
	Use:   "get <bytestream-uri>",
	Short: "Download a file by bytestream URI",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileGet,
}

var fileDeleteCmd = &cobra.Command{
	Use:   "delete <bytestream-uri>",
	Short: "Delete a cached file by bytestream URI",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileDelete,
}

func init() {
	fileCmd.AddCommand(fileGetCmd)
	fileCmd.AddCommand(fileDeleteCmd)
	rootCmd.AddCommand(fileCmd)
}

func runFileGet(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	uri := args[0]

	data, err := callGetFileRaw(client, uri)
	if err != nil {
		return output.NewAPIError(fmt.Sprintf("GetFile: %v", err))
	}

	opts := GetOutputOptions()

	// If output file specified, write binary data to file
	if opts.OutputFile != "" {
		if err := os.WriteFile(opts.OutputFile, data, 0o644); err != nil {
			return output.NewInternalError(fmt.Sprintf("write file: %v", err))
		}
		fmt.Fprintf(os.Stderr, "Wrote %s (%s)\n", opts.OutputFile, humanSize(int64(len(data))))
		return nil
	}

	// Otherwise write to stdout
	if opts.Mode == output.ModeJSON {
		return output.Render(map[string]interface{}{
			"uri":  uri,
			"size": len(data),
		}, opts)
	}

	_, err = io.Copy(os.Stdout, io.LimitReader(
		&byteReader{data: data, pos: 0}, int64(len(data)),
	))
	return err
}

func runFileDelete(cmd *cobra.Command, args []string) error {
	client, err := newClient()
	if err != nil {
		return err
	}

	uri := args[0]
	req := api.DeleteFileRequest{URI: uri}

	var resp api.DeleteFileResponse
	if err := client.Call("DeleteFile", req, &resp); err != nil {
		return output.NewAPIError(fmt.Sprintf("DeleteFile: %v", err))
	}

	opts := GetOutputOptions()
	if opts.Mode == output.ModeJSON {
		return output.Render(map[string]string{
			"status": "deleted",
			"uri":    uri,
		}, opts)
	}

	fmt.Fprintf(os.Stderr, "Deleted: %s\n", uri)
	return nil
}

// humanSize formats bytes as human-readable (duplicated for cmd package access)
func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
