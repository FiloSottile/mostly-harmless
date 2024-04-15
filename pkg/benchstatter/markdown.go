package benchstatter

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/willabides/mdtable"
	"golang.org/x/perf/benchstat"
)

func splitTableByGroup(table *benchstat.Table) []*benchstat.Table {
	tables := make([]*benchstat.Table, len(table.Groups))
	groupLookup := make(map[string]int, len(table.Groups))
	for i, group := range table.Groups {
		groupLookup[group] = i
		tables[i] = &benchstat.Table{
			Groups:      []string{group},
			Metric:      table.Metric,
			OldNewDelta: table.OldNewDelta,
			Configs:     table.Configs,
		}
	}
	for _, row := range table.Rows {
		i := groupLookup[row.Group]
		row.Group = ""
		tables[i].Rows = append(tables[i].Rows, row)
	}
	return tables
}

func splitTablesByGroup(tables []*benchstat.Table) []*benchstat.Table {
	var result []*benchstat.Table
	for _, table := range tables {
		result = append(result, splitTableByGroup(table)...)
	}
	return result
}

func csv2Markdown(data []byte) ([]string, error) {
	var csvTables [][]byte
	var currentTable []byte
	var err error
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			if len(currentTable) > 0 {
				csvTables = append(csvTables, currentTable)
			}
			currentTable = []byte{}
			continue
		}
		line = append(line, '\n')
		currentTable = append(currentTable, line...)
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	if len(currentTable) > 0 {
		csvTables = append(csvTables, currentTable)
	}
	var mdTables []string
	for _, csvTable := range csvTables {
		var rows [][]string
		rows, err = buildRows(csvTable)
		if err != nil {
			return nil, err
		}

		var mdTable string
		mdTable, err = buildMD(rows)
		mdTable += "\n"
		if err != nil {
			return nil, err
		}
		mdTables = append(mdTables, mdTable)
	}
	return mdTables, nil
}

func buildMD(rows [][]string) (string, error) {
	if len(rows) < 2 {
		return "", fmt.Errorf("need at least one row plus header")
	}

	opts := []mdtable.Option{
		mdtable.HeaderAlignment(mdtable.AlignCenter),
	}

	for i, s := range rows[0] {
		if strings.Contains(s, "(") {
			opts = append(opts, mdtable.ColumnAlignment(i, mdtable.AlignRight))
		}
	}

	return string(mdtable.Generate(rows, opts...)), nil
}

// MarkdownFormatterOptions options for a markdown OutputFormatter
type MarkdownFormatterOptions struct {
	CSVFormatterOptions
}

func buildRows(src []byte) ([][]string, error) {
	cRdr := csv.NewReader(bytes.NewReader(src))
	cRdr.FieldsPerRecord = -1
	rows, err := cRdr.ReadAll()
	if err != nil {
		return nil, err
	}
	for i := range rows {
		for j, val := range rows[i] {
			f, fErr := strconv.ParseFloat(val, 64)
			if fErr != nil {
				continue
			}
			rows[i][j] = strconv.FormatFloat(f, 'f', -1, 64)
		}
	}
	return rows, nil
}

// FormatMarkdown formats benchstat output as markdown
func FormatMarkdown(w io.Writer, tables []*benchstat.Table, opts *MarkdownFormatterOptions) error {
	if opts == nil {
		opts = new(MarkdownFormatterOptions)
	}
	csvFormatter := CSVFormatter(&opts.CSVFormatterOptions)
	tables = splitTablesByGroup(tables)
	tmpTables := make([]*benchstat.Table, 0, len(tables))
	for _, table := range tables {
		if len(table.Rows) > 0 {
			tmpTables = append(tmpTables, table)
		}
	}
	tables = tmpTables

	var groups []string
	for _, table := range tables {
		for _, group := range table.Groups {
			groups = addStringIfMissing(group, groups)
		}
	}

	for groupIdx, group := range groups {
		err := writeGroupMarkdown(w, tables, groupIdx, group, csvFormatter)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeGroupMarkdown(w io.Writer, tables []*benchstat.Table, groupIdx int, group string, csvFormatter OutputFormatter) error {
	var err error
	var groupHeader string
	if groupIdx > 0 {
		groupHeader += "\n"
	}
	fg := formatGroup(group)
	if fg != "" {
		groupHeader += fg + "\n\n"
	}

	if len(groupHeader) > 0 {
		_, err = w.Write([]byte(groupHeader))
		if err != nil {
			return err
		}
	}

	var buf bytes.Buffer
	for i := range tables {
		if tables[i].Groups[0] != group {
			continue
		}
		err = csvFormatter(&buf, tables[i:i+1])
		if err != nil {
			return err
		}
		buf.WriteString("\n")
	}

	var mdTables []string
	mdTables, err = csv2Markdown(buf.Bytes())
	if err != nil {
		return err
	}

	output := strings.Join(mdTables, "\n")
	_, err = w.Write([]byte(output))
	if err != nil {
		return err
	}
	return nil
}

// MarkdownFormatter return a markdown OutputFormatter
func MarkdownFormatter(opts *MarkdownFormatterOptions) OutputFormatter {
	return func(w io.Writer, tables []*benchstat.Table) error {
		return FormatMarkdown(w, tables, opts)
	}
}

func addStringIfMissing(s string, slice []string) []string {
	for _, s2 := range slice {
		if s == s2 {
			return slice
		}
	}
	slice = append(slice, s)
	return slice
}

var formatGroupRegexp = regexp.MustCompile(`\s*([^:^ ]+:)\s?`)

func formatGroup(group string) string {
	g := formatGroupRegexp.ReplaceAllString(group, "\n${1} ")
	return strings.TrimSpace(g)
}
