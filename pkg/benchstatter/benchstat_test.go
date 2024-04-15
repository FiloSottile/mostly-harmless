package benchstatter

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/perf/benchstat"
)

//go:generate go test . -write-golden

func TestBenchstat_Run(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir("testdata"))
	t.Cleanup(func() {
		t.Helper()
		require.NoError(t, os.Chdir(pwd))
	})
	for _, td := range goldenTests {
		t.Run(td.name, func(t *testing.T) {
			var result *benchstat.Collection
			result, err = td.benchStat.Run(td.base, td.head)
			require.NoError(t, err)
			var buf bytes.Buffer
			err = td.benchStat.OutputTables(&buf, result.Tables())
			require.NoError(t, err)
			var want []byte
			want, err = os.ReadFile(goldenFile(td))
			require.NoError(t, err)
			require.Equal(t, string(want), buf.String())
		})
	}
}

type goldenTest struct {
	name      string
	base      string
	head      string
	benchStat *Benchstat
}

var goldenTests = []*goldenTest{
	{
		name:      "example.txt",
		base:      "exampleold.txt",
		head:      "examplenew.txt",
		benchStat: new(Benchstat),
	},
	{
		name: "example.html",
		base: "exampleold.txt",
		head: "examplenew.txt",
		benchStat: &Benchstat{
			OutputFormatter: HTMLFormatter(nil),
		},
	},
	{
		name: "example.csv",
		base: "exampleold.txt",
		head: "examplenew.txt",
		benchStat: &Benchstat{
			OutputFormatter: CSVFormatter(nil),
		},
	},
	{
		name: "example.md",
		base: "exampleold.txt",
		head: "examplenew.txt",
		benchStat: &Benchstat{
			OutputFormatter: MarkdownFormatter(nil),
		},
	},
	{
		name: "example-norange.csv",
		base: "exampleold.txt",
		head: "examplenew.txt",
		benchStat: &Benchstat{
			OutputFormatter: CSVFormatter(&CSVFormatterOptions{
				NoRange: true,
			}),
		},
	},
	{
		name: "example-norange.md",
		base: "exampleold.txt",
		head: "examplenew.txt",
		benchStat: &Benchstat{
			OutputFormatter: MarkdownFormatter(&MarkdownFormatterOptions{
				CSVFormatterOptions: CSVFormatterOptions{
					NoRange: true,
				},
			}),
		},
	},
	{
		name:      "oldnew.txt",
		base:      "old.txt",
		head:      "new.txt",
		benchStat: new(Benchstat),
	},
	{
		name: "oldnewgeo.txt",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			AddGeoMean: true,
		},
	},
	{
		name: "oldnewgeo.csv",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			OutputFormatter: CSVFormatter(nil),
			AddGeoMean:      true,
		},
	},
	{
		name:      "new4.txt",
		base:      "new.txt",
		head:      "slashslash4.txt",
		benchStat: new(Benchstat),
	},
	{
		name: "oldnew.html",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			OutputFormatter: HTMLFormatter(nil),
		},
	},
	{
		name: "oldnew.csv",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			OutputFormatter: CSVFormatter(nil),
		},
	},
	{
		name: "oldnew.md",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			OutputFormatter: MarkdownFormatter(nil),
		},
	},
	{
		name: "oldnewgeo.md",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			AddGeoMean:      true,
			OutputFormatter: MarkdownFormatter(nil),
		},
	},
	{
		name: "oldnewttest.txt",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			DeltaTest: benchstat.TTest,
		},
	},
	{
		name: "packages.txt",
		base: "packagesold.txt",
		head: "packagesnew.txt",
		benchStat: &Benchstat{
			SplitBy: []string{"pkg", "goos", "goarch"},
		},
	},
	{
		name: "packages.csv",
		base: "packagesold.txt",
		head: "packagesnew.txt",
		benchStat: &Benchstat{
			OutputFormatter: CSVFormatter(nil),
			SplitBy:         []string{"pkg", "goos", "goarch"},
		},
	},
	{
		name: "packages.md",
		base: "packagesold.txt",
		head: "packagesnew.txt",
		benchStat: &Benchstat{
			OutputFormatter: MarkdownFormatter(nil),
			SplitBy:         []string{"pkg", "goos", "note"},
		},
	},
	{
		name:      "units.txt",
		base:      "units-old.txt",
		head:      "units-new.txt",
		benchStat: new(Benchstat),
	},
	{
		name: "zero.txt",
		base: "zero-old.txt",
		head: "zero-new.txt",
		benchStat: &Benchstat{
			DeltaTest: benchstat.NoDeltaTest,
		},
	},
	{
		name: "namesort.txt",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			Order: benchstat.ByName,
		},
	},
	{
		name: "deltasort.txt",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			Order: benchstat.ByDelta,
		},
	},
	{
		name: "rdeltasort.txt",
		base: "old.txt",
		head: "new.txt",
		benchStat: &Benchstat{
			Order:        benchstat.ByDelta,
			ReverseOrder: true,
		},
	},
}

func TestMain(m *testing.M) {
	var err error
	var writeGolden bool
	flag.BoolVar(&writeGolden, "write-golden", false, "write golden files")
	flag.Parse()
	if writeGolden {
		err = updateGolden()
		if err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}

func updateGolden() (err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		e := os.Chdir(pwd)
		if err == nil {
			err = e
		}
	}()
	err = os.Chdir("testdata")
	if err != nil {
		return err
	}
	files, err := filepath.Glob("*golden*")
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.Remove(file)
		if err != nil {
			return err
		}
	}
	for _, td := range goldenTests {
		var result *benchstat.Collection
		result, err = td.benchStat.Run(td.base, td.head)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		err = td.benchStat.OutputTables(&buf, result.Tables())
		if err != nil {
			return err
		}
		err = os.WriteFile(goldenFile(td), buf.Bytes(), 0o600)
		if err != nil {
			return err
		}
	}
	return nil
}

func goldenFile(td *goldenTest) string {
	return fmt.Sprintf("_golden-%s", td.name)
}
