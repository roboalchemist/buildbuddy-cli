package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
	"github.com/spf13/cobra"
)

// appVersion is set from main via SetVersion.
var appVersion = "dev"

// Global flag values.
var (
	flagJSON       bool
	flagPlaintext  bool
	flagJQ         string
	flagTemplate   string
	flagFields     string
	flagNoColor    bool
	flagDebug      bool
	flagOutputFile string
)

var rootCmd = &cobra.Command{
	Use:     "buildbuddy-cli",
	Short:   "BuildBuddy from the command line.",
	Long:    "BuildBuddy from the command line.\n\nA CLI for querying BuildBuddy Enterprise API: invocations, targets, logs, actions, and artifacts.",
	Version: appVersion,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return validateOutputFlags()
	},
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&flagJSON, "json", "j", false, "JSON output")
	pf.BoolVar(&flagPlaintext, "plaintext", false, "Tab-separated output for piping")
	pf.StringVarP(&flagTemplate, "template", "t", "", "Go template for custom formatting")
	pf.StringVar(&flagJQ, "jq", "", "Inline jq expression (implies --json)")
	pf.StringVar(&flagFields, "fields", "", "Comma-separated field selection (implies --json)")
	pf.BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	pf.BoolVar(&flagDebug, "debug", false, "Verbose logging to stderr")
	pf.StringVarP(&flagOutputFile, "output", "o", "", "Write output to file instead of stdout")
}

func validateOutputFlags() error {
	modes := 0
	if flagJSON {
		modes++
	}
	if flagPlaintext {
		modes++
	}
	if flagTemplate != "" {
		modes++
	}
	if flagJQ != "" && !flagJSON {
		flagJSON = true
	}
	if flagFields != "" && !flagJSON {
		flagJSON = true
	}
	if modes > 1 {
		return output.NewUsageError("--json, --plaintext, and --template are mutually exclusive")
	}
	return nil
}

// GetOutputOptions returns the current output configuration from global flags.
func GetOutputOptions() output.Options {
	opts := output.Options{
		NoColor:    flagNoColor,
		JQExpr:     flagJQ,
		Fields:     flagFields,
		Template:   flagTemplate,
		Debug:      flagDebug,
		OutputFile: flagOutputFile,
	}
	switch {
	case flagJSON:
		opts.Mode = output.ModeJSON
	case flagPlaintext:
		opts.Mode = output.ModePlaintext
	case flagTemplate != "":
		opts.Mode = output.ModeTemplate
	default:
		opts.Mode = output.ModeTable
	}
	return opts
}

// Execute runs the root command and handles errors.
func Execute() error {
	err := rootCmd.Execute()
	if err == nil {
		return nil
	}

	var se *output.StructuredError
	if errors.As(err, &se) {
		se.WriteJSON(os.Stderr)
		return err
	}

	wrapped := output.NewInternalError(err.Error())
	wrapped.WriteJSON(os.Stderr)
	return err
}

// DebugLog writes a debug message to stderr if --debug is set.
func DebugLog(format string, args ...interface{}) {
	if flagDebug {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}

// SetVersion sets the version string for the root command.
func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}
