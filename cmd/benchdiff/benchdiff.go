package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/alecthomas/kong"
	"github.com/willabides/benchdiff/cmd/benchdiff/internal"
	"github.com/willabides/benchdiff/pkg/benchstatter"
	"golang.org/x/perf/benchstat"
)

const defaultBenchArgsTmpl = `test {{ .Packages }} -run '^$'
{{- if .Bench }} -bench {{ .Bench }}{{end}}
{{- if .Count }} -count {{ .Count }}{{end}}
{{- if .Benchtime }} -benchtime {{ .Benchtime }}{{end}}
{{- if .CPU }} -cpu {{ .CPU }}{{ end }}
{{- if .Tags }} -tags "{{ .Tags }}"{{ end }}
{{- if .Benchmem }} -benchmem{{ end }}`

var benchstatVars = kong.Vars{
	"AlphaDefault":        "0.05",
	"AlphaHelp":           `consider change significant if p < α`,
	"DeltaTestHelp":       `significance test to apply to delta: utest, ttest, or none`,
	"DeltaTestDefault":    `utest`,
	"DeltaTestEnum":       `utest,ttest,none`,
	"GeomeanHelp":         `print the geometric mean of each file`,
	"NorangeHelp":         `suppress range columns (CSV and markdown only)`,
	"ReverseSortHelp":     `reverse sort order`,
	"SortHelp":            `sort by order: delta, name, none`,
	"SortEnum":            `delta,name,none`,
	"SplitHelp":           `split benchmarks by labels`,
	"SplitDefault":        `pkg,goos,goarch`,
	"BenchstatOutputHelp": `format for benchstat output (csv,html,markdown or text)`,
	"BenchstatOutputEnum": `csv, html, markdown, text`,
}

type benchstatOpts struct {
	Alpha           float64 `kong:"default=${AlphaDefault},help=${AlphaHelp},group=benchstat"`
	BenchstatOutput string  `kong:"default=text,enum=${BenchstatOutputEnum},help=${BenchstatOutputHelp},group=benchstat"`
	DeltaTest       string  `kong:"help=${DeltaTestHelp},default=${DeltaTestDefault},enum='utest,ttest,none',group=benchstat"`
	Geomean         bool    `kong:"help=${GeomeanHelp},group=benchstat"`
	Norange         bool    `kong:"help=${NorangeHelp},group=benchstat"`
	ReverseSort     bool    `kong:"help=${ReverseSortHelp},group=benchstat"`
	Sort            string  `kong:"help=${SortHelp},enum=${SortEnum},default=none,group=benchstat"`
	Split           string  `kong:"help=${SplitHelp},default=${SplitDefault},group=benchstat"`
}

var version string

var benchVars = kong.Vars{
	"version":              version,
	"BenchCmdDefault":      `go`,
	"CountHelp":            `Run each benchmark n times. If --cpu is set, run n times for each GOMAXPROCS value.'`,
	"BenchHelp":            `Run only those benchmarks matching a regular expression. To run all benchmarks, use '--bench .'.`,
	"BenchmarkArgsHelp":    `Override the default args to the go command. This may be a template. See https://github.com/willabides/benchdiff for details."`,
	"BenchtimeHelp":        `Run enough iterations of each benchmark to take t, specified as a time.Duration (for example, --benchtime 1h30s). The default is 1 second (1s). The special syntax Nx means to run the benchmark N times (for example, -benchtime 100x).`,
	"PackagesHelp":         `Run benchmarks in these packages.`,
	"BenchCmdHelp":         `The command to use for benchmarks.`,
	"CacheDirHelp":         `Override the default directory where benchmark output is kept.`,
	"BaseRefHelp":          `The git ref to be used as a baseline.`,
	"CooldownHelp":         `How long to pause for cooldown between head and base runs.`,
	"ForceBaseHelp":        `Rerun benchmarks on the base reference even if the output already exists.`,
	"OnDegradeHelp":        `Exit code when there is a statistically significant degradation in the results.`,
	"JSONHelp":             `Format output as JSON.`,
	"GitCmdHelp":           `The executable to use for git commands.`,
	"ToleranceHelp":        `The minimum percent change before a result is considered degraded.`,
	"VersionHelp":          `Output the benchdiff version and exit.`,
	"ShowCacheDirHelp":     `Output the cache dir and exit.`,
	"ClearCacheHelp":       `Remove benchdiff files from the cache dir.`,
	"ShowBenchCmdlineHelp": `Instead of running benchmarks, output the command that would be used and exit.`,
	"CPUHelp":              `Specify a list of GOMAXPROCS values for which the benchmarks should be executed. The default is the current value of GOMAXPROCS.`,
	"BenchmemHelp":         `Memory allocation statistics for benchmarks.`,
	"WarmupCountHelp":      `Run benchmarks with -count=n as a warmup`,
	"WarmupTimeHelp":       `When warmups are run, set -benchtime=n`,
	"TagsHelp":             `Set the -tags flag on the go test command`,
}

var groupHelp = kong.Vars{
	"benchstatGroupHelp": "benchstat options:",
	"gotestGroupHelp":    "benchmark command line:",
	"cacheGroupHelp":     "benchmark result cache:",
}

var cli struct {
	Version kong.VersionFlag `kong:"help=${VersionHelp}"`
	Debug   bool             `kong:"help='write verbose output to stderr'"`

	BaseRef   string        `kong:"default=HEAD,help=${BaseRefHelp},group='x'"`
	Cooldown  time.Duration `kong:"default='100ms',help=${CooldownHelp},group='x'"`
	ForceBase bool          `kong:"help=${ForceBaseHelp},group='x'"`
	GitCmd    string        `kong:"default=git,help=${GitCmdHelp},group='x'"`
	JSON      bool          `kong:"help=${JSONHelp},group='x'"`
	OnDegrade int           `kong:"name=on-degrade,default=0,help=${OnDegradeHelp},group='x'"`
	Tolerance float64       `kong:"default='10.0',help=${ToleranceHelp},group='x'"`

	Bench            string               `kong:"default='.',help=${BenchHelp},group='gotest'"`
	BenchmarkArgs    string               `kong:"placeholder='args',help=${BenchmarkArgsHelp},group='gotest'"`
	BenchmarkCmd     string               `kong:"default=${BenchCmdDefault},help=${BenchCmdHelp},group='gotest'"`
	Benchmem         bool                 `kong:"help=${BenchmemHelp},group='gotest'"`
	Benchtime        string               `kong:"help=${BenchtimeHelp},group='gotest'"`
	Count            int                  `kong:"default=10,help=${CountHelp},group='gotest'"`
	CPU              CPUFlag              `kong:"help=${CPUHelp},group='gotest',placeholder='GOMAXPROCS,...'"`
	Packages         string               `kong:"default='./...',help=${PackagesHelp},group='gotest'"`
	ShowBenchCmdline ShowBenchCmdlineFlag `kong:"help=${ShowBenchCmdlineHelp},group='gotest'"`
	Tags             string               `kong:"help=${TagsHelp},group='gotest'"`
	WarmupCount      int                  `kong:"help=${WarmupCountHelp},group='gotest'"`
	WarmupTime       string               `kong:"help=${WarmupTimeHelp},group='gotest'"`

	BenchstatOpts benchstatOpts `kong:"embed"`

	CacheDir     string           `kong:"type=dir,help=${CacheDirHelp},group='cache'"`
	ClearCache   ClearCacheFlag   `kong:"help=${ClearCacheHelp},group='cache'"`
	ShowCacheDir ShowCacheDirFlag `kong:"help=${ShowCacheDirHelp},group='cache'"`

	ShowDefaultTemplate showDefaultTemplate `kong:"hidden"`
}

// ShowCacheDirFlag flag for showing the cache directory
type ShowCacheDirFlag bool

// AfterApply outputs cli.CacheDir
func (v ShowCacheDirFlag) AfterApply(app *kong.Kong) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}
	fmt.Fprintln(app.Stdout, cacheDir)
	app.Exit(0)
	return nil
}

type showDefaultTemplate bool

func (v showDefaultTemplate) BeforeApply(app *kong.Kong) error {
	fmt.Println(defaultBenchArgsTmpl)
	app.Exit(0)
	return nil
}

// ClearCacheFlag flag for clearing cache
type ClearCacheFlag bool

// AfterApply clears cache
func (v ClearCacheFlag) AfterApply(app *kong.Kong) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(cacheDir, "benchdiff-*.out"))
	if err != nil {
		return fmt.Errorf("error finding files in %s: %v", cacheDir, err)
	}
	for _, file := range files {
		err = os.Remove(file)
		if err != nil {
			return fmt.Errorf("error removing %s: %v", file, err)
		}
	}
	app.Exit(0)
	return nil
}

func getCacheDir() (string, error) {
	if cli.CacheDir != "" {
		return cli.CacheDir, nil
	}
	return defaultCacheDir()
}

func defaultCacheDir() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("error finding user cache dir: %v", err)
	}
	return filepath.Join(userCacheDir, "benchdiff"), nil
}

// ShowBenchCmdlineFlag flag for --show-bench-cmdling
type ShowBenchCmdlineFlag bool

// AfterApply shows benchmark command line and exits
func (v ShowBenchCmdlineFlag) AfterApply(app *kong.Kong) error {
	benchArgs, err := getBenchArgs()
	if err != nil {
		return err
	}
	fmt.Fprintln(app.Stdout, cli.BenchmarkCmd, benchArgs)
	app.Exit(0)
	return nil
}

// CPUFlag is the flag for --cpu
type CPUFlag []int

func (c CPUFlag) String() string {
	s := make([]string, len(c))
	for i, cc := range c {
		s[i] = strconv.Itoa(cc)
	}
	return strings.Join(s, ",")
}

func getBenchArgs() (string, error) {
	argsTmpl := cli.BenchmarkArgs
	if argsTmpl == "" {
		argsTmpl = defaultBenchArgsTmpl
	}
	tmpl, err := template.New("").Parse(argsTmpl)
	if err != nil {
		return "", err
	}
	var benchArgs bytes.Buffer
	err = tmpl.Execute(&benchArgs, cli)
	if err != nil {
		return "", err
	}
	args := benchArgs.String()
	return args, nil
}

const description = `
benchdiff runs go benchmarks on your current git worktree and a base ref then
uses benchstat to show the delta.

More documentation at https://github.com/willabides/benchdiff.
`

func main() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error finding user cache dir: %v\n", err)
		os.Exit(1)
	}
	benchVars["CacheDirDefault"] = filepath.Join(userCacheDir, "benchdiff")

	kctx := kong.Parse(&cli, benchstatVars, benchVars, groupHelp,
		kong.Description(strings.TrimSpace(description)),
		kong.ExplicitGroups([]kong.Group{
			{Key: "benchstat", Title: "benchstat options"},
			{Key: "cache", Title: "benchmark result cache"},
			{Key: "gotest", Title: "benchmark command line"},
			{Key: "x"},
		}),
	)

	benchArgs, err := getBenchArgs()
	kctx.FatalIfErrorf(err)

	cacheDir, err := getCacheDir()
	kctx.FatalIfErrorf(err)

	bStat, err := buildBenchstat(&cli.BenchstatOpts)
	kctx.FatalIfErrorf(err)

	bd := &internal.Benchdiff{
		BenchCmd:    cli.BenchmarkCmd,
		BenchArgs:   benchArgs,
		ResultsDir:  cacheDir,
		BaseRef:     cli.BaseRef,
		Path:        ".",
		Writer:      os.Stdout,
		Benchstat:   bStat,
		Force:       cli.ForceBase,
		GitCmd:      cli.GitCmd,
		Cooldown:    cli.Cooldown,
		WarmupTime:  cli.WarmupTime,
		WarmupCount: cli.WarmupCount,
	}
	if cli.Debug {
		bd.Debug = log.New(os.Stderr, "", 0)
	}
	result, err := bd.Run()
	kctx.FatalIfErrorf(err)

	outputFormat := "human"
	if cli.JSON {
		outputFormat = "json"
	}

	err = result.WriteOutput(os.Stdout, &internal.RunResultOutputOptions{
		BenchstatFormatter: bStat.OutputFormatter,
		OutputFormat:       outputFormat,
		Tolerance:          cli.Tolerance,
	})
	kctx.FatalIfErrorf(err)
	if result.HasDegradedResult(cli.Tolerance) {
		os.Exit(cli.OnDegrade)
	}
}

var deltaTestOpts = map[string]benchstat.DeltaTest{
	"none":  benchstat.NoDeltaTest,
	"utest": benchstat.UTest,
	"ttest": benchstat.TTest,
}

var sortOpts = map[string]benchstat.Order{
	"none":  nil,
	"name":  benchstat.ByName,
	"delta": benchstat.ByDelta,
}

func buildBenchstat(opts *benchstatOpts) (*benchstatter.Benchstat, error) {
	order := sortOpts[opts.Sort]
	reverse := opts.ReverseSort
	if order == nil {
		reverse = false
	}
	var formatter benchstatter.OutputFormatter
	switch opts.BenchstatOutput {
	case "text":
		formatter = benchstatter.TextFormatter(nil)
	case "csv":
		formatter = benchstatter.CSVFormatter(&benchstatter.CSVFormatterOptions{
			NoRange: opts.Norange,
		})
	case "html":
		formatter = benchstatter.HTMLFormatter(nil)
	case "markdown":
		formatter = benchstatter.MarkdownFormatter(&benchstatter.MarkdownFormatterOptions{
			CSVFormatterOptions: benchstatter.CSVFormatterOptions{
				NoRange: opts.Norange,
			},
		})
	default:
		return nil, fmt.Errorf("unexpected output format: %s", opts.BenchstatOutput)
	}

	return &benchstatter.Benchstat{
		DeltaTest:       deltaTestOpts[opts.DeltaTest],
		Alpha:           opts.Alpha,
		AddGeoMean:      opts.Geomean,
		SplitBy:         strings.Split(opts.Split, ","),
		Order:           order,
		ReverseOrder:    reverse,
		OutputFormatter: formatter,
	}, nil
}
