package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/willabides/benchdiff/pkg/benchstatter"
	"golang.org/x/crypto/sha3"
	"golang.org/x/perf/benchstat"
)

// Benchdiff runs benchstats and outputs their deltas
type Benchdiff struct {
	BenchCmd    string
	BenchArgs   string
	ResultsDir  string
	BaseRef     string
	Path        string
	GitCmd      string
	Writer      io.Writer
	Benchstat   *benchstatter.Benchstat
	Force       bool
	JSONOutput  bool
	Cooldown    time.Duration
	WarmupCount int
	WarmupTime  string
	Debug       *log.Logger
}

type runBenchmarksResults struct {
	worktreeOutputFile string
	baseOutputFile     string
	benchmarkCmd       string
	headSHA            string
	baseSHA            string
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func (c *Benchdiff) debug() *log.Logger {
	if c.Debug == nil {
		return log.New(io.Discard, "", 0)
	}
	return c.Debug
}

func (c *Benchdiff) gitCmd() string {
	if c.GitCmd == "" {
		return "git"
	}
	return c.GitCmd
}

func (c *Benchdiff) cacheKey() string {
	var b []byte
	b = append(b, []byte(c.BenchCmd)...)
	b = append(b, []byte(c.BenchArgs)...)
	sum := sha3.Sum224(b)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// runCmd runs cmd sending its stdout and stderr to debug.Write()
func runCmd(cmd *exec.Cmd, debug *log.Logger) error {
	if debug == nil {
		debug = log.New(io.Discard, "", 0)
	}
	var bufStderr bytes.Buffer
	stderr := io.MultiWriter(&bufStderr, debug.Writer())
	if cmd.Stderr != nil {
		stderr = io.MultiWriter(cmd.Stderr, stderr)
	}
	cmd.Stderr = stderr
	stdout := debug.Writer()
	if cmd.Stdout != nil {
		stdout = io.MultiWriter(cmd.Stdout, stdout)
	}
	cmd.Stdout = stdout
	debug.Printf("+ %s", cmd)
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf(`error running command: %s
exit code: %d
stderr: %s`, cmd.String(), exitErr.ExitCode(), bufStderr.String())
	}
	return err
}

func (c *Benchdiff) runBenchmark(ref, filename, extraArgs string, pause time.Duration, force bool) error {
	cmd := exec.Command(c.BenchCmd, strings.Fields(c.BenchArgs+" "+extraArgs)...)

	stdlib := false
	if rootPath, err := runGitCmd(c.debug(), c.gitCmd(), c.Path, "rev-parse", "--show-toplevel"); err == nil {
		// lib/time/zoneinfo.zip is a specific enough path, and it's here to
		// stay because it's one of the few paths hardcoded into Go binaries.
		zoneinfoPath := filepath.Join(string(rootPath), "lib", "time", "zoneinfo.zip")
		if _, err := os.Stat(zoneinfoPath); err == nil {
			stdlib = true
			cmd.Path = filepath.Join(string(rootPath), "bin", "go")
		}
	}

	fileBuffer := &bytes.Buffer{}
	if filename != "" {
		c.debug().Printf("output file: %s", filename)
		if ref != "" && !force {
			if fileExists(filename) {
				c.debug().Printf("+ skipping benchmark for ref %q because output file exists", ref)
				return nil
			}
		}
		cmd.Stdout = fileBuffer
	}

	var runErr error
	if ref == "" {
		runErr = runCmd(cmd, c.debug())
	} else {
		err := runAtGitRef(c.debug(), c.gitCmd(), c.Path, c.BaseRef, func(workPath string) {
			if pause > 0 {
				time.Sleep(pause)
			}
			if stdlib {
				makeCmd := exec.Command(filepath.Join(workPath, "src", "make.bash"))
				makeCmd.Dir = filepath.Join(workPath, "src")
				makeCmd.Env = append(os.Environ(), "GOOS=", "GOARCH=")
				runErr = runCmd(makeCmd, c.debug())
				if runErr != nil {
					return
				}
				cmd.Path = filepath.Join(workPath, "bin", "go")
			}
			cmd.Dir = workPath // TODO: add relative path of working directory
			runErr = runCmd(cmd, c.debug())
		})
		if err != nil {
			return err
		}
	}
	if runErr != nil {
		return runErr
	}
	if filename == "" {
		return nil
	}
	return os.WriteFile(filename, fileBuffer.Bytes(), 0o666)
}

func (c *Benchdiff) runBenchmarks() (result *runBenchmarksResults, err error) {
	headSHA, err := runGitCmd(c.debug(), c.gitCmd(), c.Path, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}

	baseSHA, err := runGitCmd(c.debug(), c.gitCmd(), c.Path, "rev-parse", c.BaseRef)
	if err != nil {
		return nil, err
	}

	baseFilename := fmt.Sprintf("benchdiff-%s-%s.out", baseSHA, c.cacheKey())
	baseFilename = filepath.Join(c.ResultsDir, baseFilename)

	worktreeFilename := filepath.Join(c.ResultsDir, "benchdiff-worktree.out")

	result = &runBenchmarksResults{
		benchmarkCmd:       fmt.Sprintf("%s %s", c.BenchCmd, c.BenchArgs),
		headSHA:            strings.TrimSpace(string(headSHA)),
		baseSHA:            strings.TrimSpace(string(baseSHA)),
		baseOutputFile:     baseFilename,
		worktreeOutputFile: worktreeFilename,
	}

	doWarmup := c.WarmupCount > 0

	warmupArgs := fmt.Sprintf("-count %d", c.WarmupCount)
	if c.WarmupTime != "" {
		warmupArgs = fmt.Sprintf("%s -benchtime %s", warmupArgs, c.WarmupTime)
	}

	var cooldown time.Duration

	if doWarmup {
		err = c.runBenchmark(c.BaseRef, "", warmupArgs, cooldown, c.Force)
		if err != nil {
			return nil, err
		}
		cooldown = c.Cooldown
	}

	err = c.runBenchmark(c.BaseRef, baseFilename, "", cooldown, c.Force)
	if err != nil {
		return nil, err
	}
	cooldown = c.Cooldown

	err = c.runBenchmark("", worktreeFilename, "", cooldown, false)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Run runs the Benchdiff
func (c *Benchdiff) Run() (*RunResult, error) {
	err := os.MkdirAll(c.ResultsDir, 0o700)
	if err != nil {
		return nil, err
	}
	res, err := c.runBenchmarks()
	if err != nil {
		return nil, err
	}
	collection, err := c.Benchstat.Run(res.baseOutputFile, res.worktreeOutputFile)
	if err != nil {
		return nil, err
	}
	result := &RunResult{
		headSHA:  res.headSHA,
		baseSHA:  res.baseSHA,
		benchCmd: res.benchmarkCmd,
		tables:   collection.Tables(),
	}
	return result, nil
}

// RunResult is the result of a Run
type RunResult struct {
	headSHA  string
	baseSHA  string
	benchCmd string
	tables   []*benchstat.Table
}

// RunResultOutputOptions options for RunResult.WriteOutput
type RunResultOutputOptions struct {
	BenchstatFormatter benchstatter.OutputFormatter // default benchstatter.TextFormatter(nil)
	OutputFormat       string                       // one of json or human. default: human
	Tolerance          float64
}

// WriteOutput outputs the result
func (r *RunResult) WriteOutput(w io.Writer, opts *RunResultOutputOptions) error {
	if opts == nil {
		opts = new(RunResultOutputOptions)
	}
	finalOpts := &RunResultOutputOptions{
		BenchstatFormatter: benchstatter.TextFormatter(nil),
		OutputFormat:       "human",
		Tolerance:          opts.Tolerance,
	}
	if opts.BenchstatFormatter != nil {
		finalOpts.BenchstatFormatter = opts.BenchstatFormatter
	}

	if opts.OutputFormat != "" {
		finalOpts.OutputFormat = opts.OutputFormat
	}

	var benchstatBuf bytes.Buffer
	err := finalOpts.BenchstatFormatter(&benchstatBuf, r.tables)
	if err != nil {
		return err
	}

	switch finalOpts.OutputFormat {
	case "human":
		return r.writeHumanResult(w, benchstatBuf.String())
	case "json":
		return r.writeJSONResult(w, benchstatBuf.String(), finalOpts.Tolerance)
	default:
		return fmt.Errorf("unknown OutputFormat")
	}
}

func (r *RunResult) writeJSONResult(w io.Writer, benchstatResult string, tolerance float64) error {
	type runResultJSON struct {
		BenchCommand    string `json:"bench_command,omitempty"`
		HeadSHA         string `json:"head_sha,omitempty"`
		BaseSHA         string `json:"base_sha,omitempty"`
		DegradedResult  bool   `json:"degraded_result"`
		BenchstatOutput string `json:"benchstat_output,omitempty"`
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&runResultJSON{
		BenchCommand:    r.benchCmd,
		BenchstatOutput: benchstatResult,
		HeadSHA:         r.headSHA,
		BaseSHA:         r.baseSHA,
		DegradedResult:  r.HasDegradedResult(tolerance),
	})
}

func (r *RunResult) writeHumanResult(w io.Writer, benchstatResult string) error {
	var err error
	_, err = fmt.Fprintf(w, "bench command:\n  %s\n", r.benchCmd)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "HEAD sha:\n  %s\n", r.headSHA)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "base sha:\n  %s\n", r.baseSHA)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "benchstat output:\n\n%s\n", benchstatResult)
	if err != nil {
		return err
	}

	return nil
}

// HasDegradedResult returns true if there are any rows with DegradingChange and PctDelta over tolerance
func (r *RunResult) HasDegradedResult(tolerance float64) bool {
	return r.maxDegradedPct() > tolerance
}

func (r *RunResult) maxDegradedPct() float64 {
	max := 0.0
	for _, table := range r.tables {
		for _, row := range table.Rows {
			if row.Change != DegradingChange {
				continue
			}
			if row.PctDelta > max {
				max = row.PctDelta
			}
		}
	}
	return max
}

// BenchmarkChangeType is whether a change is an improvement or degradation
type BenchmarkChangeType int

// BenchmarkChangeType values
const (
	DegradingChange     = -1 // represents a statistically significant degradation
	InsignificantChange = 0  // represents no statistically significant change
	ImprovingChange     = 1  // represents a statistically significant improvement
)
