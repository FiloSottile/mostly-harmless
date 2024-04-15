// Package benchstatter is used to run benchstatter programmatically
package benchstatter

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/perf/benchstat"
)

// Benchstat is a benchstat runner
type Benchstat struct {
	// DeltaTest is the test to use to decide if a change is significant.
	// If nil, it defaults to UTest.
	DeltaTest benchstat.DeltaTest

	// Alpha is the p-value cutoff to report a change as significant.
	// If zero, it defaults to 0.05.
	Alpha float64

	// AddGeoMean specifies whether to add a line to the table
	// showing the geometric mean of all the benchmark results.
	AddGeoMean bool

	// SplitBy specifies the labels to split results by.
	// By default, results will only be split by full name.
	SplitBy []string

	// Order specifies the row display order for this table.
	// If Order is nil, the table rows are printed in order of
	// first appearance in the input.
	Order benchstat.Order

	// ReverseOrder reverses the display order. Not valid if Order is nil.
	ReverseOrder bool

	// OutputFormatter determines how the output will be formatted. Default is TextFormatter
	OutputFormatter OutputFormatter
}

// OutputFormatter formats benchstat output
type OutputFormatter func(w io.Writer, tables []*benchstat.Table) error

// Collection returns a *benchstat.Collection
func (b *Benchstat) Collection() *benchstat.Collection {
	order := b.Order
	if b.ReverseOrder {
		order = benchstat.Reverse(order)
	}

	return &benchstat.Collection{
		Alpha:      b.Alpha,
		AddGeoMean: b.AddGeoMean,
		DeltaTest:  b.DeltaTest,
		SplitBy:    b.SplitBy,
		Order:      order,
	}
}

// Run runs benchstat
func (b *Benchstat) Run(files ...string) (*benchstat.Collection, error) {
	collection := b.Collection()
	err := AddCollectionFiles(collection, files...)
	if err != nil {
		return nil, err
	}
	return collection, nil
}

// OutputTables outputs the results from tables using b.OutputFormatter
func (b *Benchstat) OutputTables(writer io.Writer, tables []*benchstat.Table) error {
	formatter := b.OutputFormatter
	if formatter == nil {
		formatter = TextFormatter(nil)
	}
	return formatter(writer, tables)
}

// AddCollectionFiles adds files to a collection
func AddCollectionFiles(c *benchstat.Collection, files ...string) error {
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		err = c.AddFile(file, f)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// TextFormatterOptions options for a text OutputFormatter
type TextFormatterOptions struct{}

// TextFormatter returns a text OutputFormatter
func TextFormatter(_ *TextFormatterOptions) OutputFormatter {
	return func(w io.Writer, tables []*benchstat.Table) error {
		benchstat.FormatText(w, tables)
		return nil
	}
}

// CSVFormatterOptions options for a csv OutputFormatter
type CSVFormatterOptions struct {
	NoRange bool
}

// CSVFormatter returns a csv OutputFormatter
func CSVFormatter(opts *CSVFormatterOptions) OutputFormatter {
	noRange := false
	if opts != nil {
		noRange = opts.NoRange
	}
	return func(w io.Writer, tables []*benchstat.Table) error {
		benchstat.FormatCSV(w, tables, noRange)
		return nil
	}
}

// HTMLFormatterOptions options for an html OutputFormatter
type HTMLFormatterOptions struct {
	Header string
	Footer string
}

// HTMLFormatter return an html OutputFormatter
func HTMLFormatter(opts *HTMLFormatterOptions) OutputFormatter {
	header := defaultHTMLHeader
	footer := defaultHTMLFooter
	if opts != nil {
		header = opts.Header
		footer = opts.Footer
	}
	return func(w io.Writer, tables []*benchstat.Table) error {
		if header != "" {
			_, err := w.Write([]byte(header))
			if err != nil {
				return err
			}
		}
		var buf bytes.Buffer
		benchstat.FormatHTML(&buf, tables)
		_, err := w.Write(buf.Bytes())
		if err != nil {
			return err
		}
		if footer != "" {
			_, err = w.Write([]byte(footer))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

var defaultHTMLHeader = `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>Performance Result Comparison</title>
<style>
.benchstat { border-collapse: collapse; }
.benchstat th:nth-child(1) { text-align: left; }
.benchstat tbody td:nth-child(1n+2):not(.note) { text-align: right; padding: 0em 1em; }
.benchstat tr:not(.configs) th { border-top: 1px solid #666; border-bottom: 1px solid #ccc; }
.benchstat .nodelta { text-align: center !important; }
.benchstat .better td.delta { font-weight: bold; }
.benchstat .worse td.delta { font-weight: bold; color: #c00; }
</style>
</head>
<body>
`

var defaultHTMLFooter = `</body>
</html>
`
