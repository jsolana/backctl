package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

func DetectFormat(explicit string) Format {
	switch strings.ToLower(explicit) {
	case "json":
		return FormatJSON
	case "yaml":
		return FormatYAML
	case "table":
		return FormatTable
	default:
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			return FormatJSON
		}
		return FormatTable
	}
}

func Print(w io.Writer, format Format, data any) error {
	switch format {
	case FormatJSON:
		return printJSON(w, data)
	case FormatYAML:
		return printYAML(w, data)
	case FormatTable:
		return printTable(w, data)
	default:
		return printJSON(w, data)
	}
}

func printYAML(w io.Writer, data any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(data)
}

func printJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func printTable(w io.Writer, data any) error {
	switch v := data.(type) {
	case TableData:
		return renderTable(w, v)
	default:
		return printJSON(w, data)
	}
}

type TableData struct {
	Headers []string
	Rows    [][]string
}

func renderTable(w io.Writer, td TableData) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(td.Headers, "\t"))
	for _, row := range td.Rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}
