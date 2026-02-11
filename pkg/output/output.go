package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// Mode represents the output rendering mode.
type Mode int

const (
	ModeTable     Mode = iota // default, TTY-aware with colors
	ModePlaintext             // tab-separated, no colors
	ModeJSON                  // JSON with omitempty pruning
	ModeTemplate              // Go template rendering
)

// Options holds the global output configuration derived from flags.
type Options struct {
	Mode       Mode
	NoColor    bool
	JQExpr     string
	Fields     string
	Template   string
	Debug      bool
	OutputFile string
}

// TableData represents data for table output.
type TableData struct {
	Headers []string
	Rows    [][]string
}

// writer returns the appropriate writer based on OutputFile setting.
// If OutputFile is set, creates the file. Caller must close the returned file.
func (o Options) writer() (io.Writer, *os.File, error) {
	if o.OutputFile == "" {
		return os.Stdout, nil, nil
	}
	dir := filepath.Dir(o.OutputFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create output dir: %w", err)
	}
	f, err := os.Create(o.OutputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("create output file: %w", err)
	}
	return f, f, nil
}

// Render dispatches to the appropriate output mode.
func Render(data interface{}, opts Options) error {
	w, f, err := opts.writer()
	if err != nil {
		return err
	}
	if f != nil {
		defer func() {
			_ = f.Close()
			info, _ := f.Stat()
			if info != nil {
				fmt.Fprintf(os.Stderr, "Wrote %s (%s)\n", opts.OutputFile, HumanSize(info.Size()))
			}
		}()
	}

	switch opts.Mode {
	case ModeJSON:
		return renderJSONFull(w, data, opts)
	case ModeTemplate:
		return renderTemplate(w, data, opts)
	case ModePlaintext:
		return renderJSONFull(w, data, opts)
	default:
		return renderJSON(w, data)
	}
}

// RenderTable renders TableData in the appropriate output mode.
func RenderTable(td TableData, data interface{}, opts Options) error {
	w, f, err := opts.writer()
	if err != nil {
		return err
	}
	if f != nil {
		defer func() {
			_ = f.Close()
			info, _ := f.Stat()
			if info != nil {
				fmt.Fprintf(os.Stderr, "Wrote %s (%s)\n", opts.OutputFile, HumanSize(info.Size()))
			}
		}()
	}

	switch opts.Mode {
	case ModeJSON:
		return renderJSONFull(w, data, opts)
	case ModeTemplate:
		return renderTemplate(w, data, opts)
	case ModePlaintext:
		return renderPlaintext(w, td)
	default:
		return renderTableFormatted(w, td, opts)
	}
}

// RenderStream writes a reader directly to output (file or stdout).
// Used for large responses like logs.
func RenderStream(r io.Reader, opts Options) error {
	w, f, err := opts.writer()
	if err != nil {
		return err
	}
	if f != nil {
		defer func() {
			_ = f.Close()
			info, _ := f.Stat()
			if info != nil {
				fmt.Fprintf(os.Stderr, "Wrote %s (%s)\n", opts.OutputFile, HumanSize(info.Size()))
			}
		}()
	}
	_, err = io.Copy(w, r)
	return err
}

func renderJSONFull(w io.Writer, data interface{}, opts Options) error {
	if opts.Fields != "" {
		fields := ParseFieldList(opts.Fields)
		if len(fields) > 0 {
			data = FilterFields(data, fields)
		}
	}

	if opts.JQExpr != "" {
		result, err := ApplyJQ(data, opts.JQExpr)
		if err != nil {
			return fmt.Errorf("jq: %w", err)
		}
		data = result
	}

	return renderJSON(w, data)
}

func renderJSON(w io.Writer, data interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return fmt.Errorf("json unmarshal for pruning: %w", err)
	}

	pruned := PruneEmpty(generic)
	if pruned == nil {
		pruned = map[string]interface{}{}
	}

	out, err := json.MarshalIndent(pruned, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal pruned: %w", err)
	}

	_, err = fmt.Fprintln(w, string(out))
	return err
}

func renderTemplate(w io.Writer, data interface{}, opts Options) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("template marshal: %w", err)
	}
	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return fmt.Errorf("template unmarshal: %w", err)
	}

	tmpl, err := template.New("output").Parse(opts.Template)
	if err != nil {
		return fmt.Errorf("template parse: %w", err)
	}

	if err := tmpl.Execute(w, generic); err != nil {
		return fmt.Errorf("template exec: %w", err)
	}
	_, err = fmt.Fprintln(w)
	return err
}

func renderPlaintext(w io.Writer, td TableData) error {
	if len(td.Headers) > 0 {
		fmt.Fprintln(w, strings.Join(td.Headers, "\t"))
	}
	for _, row := range td.Rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return nil
}

func renderTableFormatted(w io.Writer, td TableData, opts Options) error {
	table := tablewriter.NewWriter(w)

	// Color headers if not disabled
	if !opts.NoColor && ShouldColor(opts) {
		coloredHeaders := make([]string, len(td.Headers))
		for i, header := range td.Headers {
			coloredHeaders[i] = color.New(color.FgCyan, color.Bold).Sprint(header)
		}
		table.SetHeader(coloredHeaders)
	} else {
		table.SetHeader(td.Headers)
	}

	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("   ")
	table.SetNoWhiteSpace(true)

	for _, row := range td.Rows {
		table.Append(row)
	}
	table.Render()
	return nil
}

// DetectTTY returns true if stdout is a terminal.
func DetectTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// ShouldColor returns true if colored output is appropriate.
func ShouldColor(opts Options) bool {
	if opts.NoColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return DetectTTY()
}

// HumanSize formats bytes as human-readable (e.g., "1.5MB").
func HumanSize(b int64) string {
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
