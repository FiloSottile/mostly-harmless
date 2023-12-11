package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/alecthomas/kong"
	"golang.org/x/crypto/sha3"
)

const defaultBenchArgsTmpl = `test {{ .Packages }} -run '^$'
{{- if .Bench }} -bench {{ .Bench }}{{end}}
{{- if .Count }} -count {{ .Count }}{{end}}
{{- if .Benchtime }} -benchtime {{ .Benchtime }}{{end}}
{{- if .CPU }} -cpu {{ .CPU }}{{ end }}
{{- if .Tags }} -tags "{{ .Tags }}"{{ end }}
{{- if .Benchmem }} -benchmem{{ end }}`

var version string

var benchVars = kong.Vars{
	"version":           version,
	"BenchCmdDefault":   `go`,
	"CountHelp":         `Run each benchmark n times. If --cpu is set, run n times for each GOMAXPROCS value.'`,
	"BenchHelp":         `Run only those benchmarks matching a regular expression. To run all benchmarks, use '--bench .'.`,
	"BenchmarkArgsHelp": `Override the default args to the go command. This may be a template. See https://github.com/willabides/benchdiff for details."`,
	"BenchtimeHelp":     `Run enough iterations of each benchmark to take t, specified as a time.Duration (for example, --benchtime 1h30s). The default is 1 second (1s). The special syntax Nx means to run the benchmark N times (for example, -benchtime 100x).`,
	"PackagesHelp":      `Run benchmarks in these packages.`,
	"BenchCmdHelp":      `The command to use for benchmarks.`,
	"BenchstatCmdHelp":  `The command to use for benchstat.`,
	"CacheDirHelp":      `Override the default directory where benchmark output is kept.`,
	"BaseRefHelp":       `The git ref to be used as a baseline.`,
	"HeadRefHelp":       `The git ref to be benchmarked. By default the worktree is used.`,
	"NoCacheHelp":       `Rerun benchmarks even if the output already exists.`,
	"GitCmdHelp":        `The executable to use for git commands.`,
	"VersionHelp":       `Output the benchdiff version and exit.`,
	"ShowCacheDirHelp":  `Output the cache dir and exit.`,
	"ClearCacheHelp":    `Remove benchdiff files from the cache dir.`,
	"CPUHelp":           `Specify a list of GOMAXPROCS values for which the benchmarks should be executed. The default is the current value of GOMAXPROCS.`,
	"BenchmemHelp":      `Memory allocation statistics for benchmarks.`,
	"TagsHelp":          `Set the -tags flag on the go test command`,
}

var groupHelp = kong.Vars{
	"gotestGroupHelp": "benchmark command line:",
	"cacheGroupHelp":  "benchmark result cache:",
}

var cli struct {
	Version kong.VersionFlag `kong:"help=${VersionHelp}"`
	Debug   bool             `kong:"help='write verbose output to stderr'"`

	BaseRef      string `kong:"default=HEAD,help=${BaseRefHelp},group='x'"`
	HeadRef      string `kong:"help=${BaseRefHelp},group='x'"`
	GitCmd       string `kong:"default=git,help=${GitCmdHelp},group='x'"`
	BenchstatCmd string `kong:"default=benchstat,help=${BenchstatCmdHelp},group='x'"`

	Bench         string  `kong:"default='.',help=${BenchHelp},group='gotest'"`
	BenchmarkArgs string  `kong:"placeholder='args',help=${BenchmarkArgsHelp},group='gotest'"`
	BenchmarkCmd  string  `kong:"default=${BenchCmdDefault},help=${BenchCmdHelp},group='gotest'"`
	Benchmem      bool    `kong:"help=${BenchmemHelp},group='gotest'"`
	Benchtime     string  `kong:"help=${BenchtimeHelp},group='gotest'"`
	Count         int     `kong:"default=10,help=${CountHelp},group='gotest'"`
	CPU           CPUFlag `kong:"help=${CPUHelp},group='gotest',placeholder='GOMAXPROCS,...'"`
	Packages      string  `kong:"default='./...',help=${PackagesHelp},group='gotest'"`
	Tags          string  `kong:"help=${TagsHelp},group='gotest'"`

	CacheDir     string           `kong:"type=dir,help=${CacheDirHelp},group='cache'"`
	ClearCache   ClearCacheFlag   `kong:"help=${ClearCacheHelp},group='cache'"`
	ShowCacheDir ShowCacheDirFlag `kong:"help=${ShowCacheDirHelp},group='cache'"`
	NoCache      bool             `kong:"help=${NoCacheHelp},group='cache'"`

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

	kctx := kong.Parse(&cli, benchVars, groupHelp,
		kong.Description(strings.TrimSpace(description)),
		kong.ExplicitGroups([]kong.Group{
			{Key: "cache", Title: "benchmark result cache"},
			{Key: "gotest", Title: "benchmark command line"},
			{Key: "x"},
		}),
	)

	benchArgs, err := getBenchArgs()
	kctx.FatalIfErrorf(err)

	cacheDir, err := getCacheDir()
	kctx.FatalIfErrorf(err)

	bd := &Benchdiff{
		GoCmd:      cli.BenchmarkCmd,
		BenchArgs:  benchArgs,
		ResultsDir: cacheDir,
		BaseRef:    cli.BaseRef,
		HeadRef:    cli.HeadRef,
		Writer:     os.Stdout,
		Force:      cli.NoCache,
		GitCmd:     cli.GitCmd,
	}
	if cli.Debug {
		bd.Debug = log.New(os.Stderr, "", 0)
	}
	result, err := bd.Run()
	kctx.FatalIfErrorf(err)

	cmd := exec.Command(cli.BenchstatCmd, result.BaseRef+"="+result.BaseOutputFile,
		result.HeadRef+"="+result.HeadOutputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if cli.Debug {
		bd.Debug.Printf("+ %s", cmd)
	}
	err = cmd.Run()
	kctx.FatalIfErrorf(err)
}

type Benchdiff struct {
	GoCmd      string
	BenchArgs  string
	ResultsDir string
	BaseRef    string
	HeadRef    string
	GitCmd     string
	Writer     io.Writer
	Force      bool
	Debug      *log.Logger
}

type RunResult struct {
	HeadOutputFile string
	BaseOutputFile string
	BenchmarkCmd   string
	HeadRef        string
	BaseRef        string
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

func (c *Benchdiff) cacheKey() string {
	var b []byte
	b = append(b, []byte(c.GoCmd)...)
	b = append(b, []byte(c.BenchArgs)...)
	b = append(b, []byte(os.Getenv("GOOS"))...)
	b = append(b, []byte(os.Getenv("GOARCH"))...)
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

func (c *Benchdiff) runBenchmark(ref, filename string, force bool) error {
	cmd := exec.Command(c.GoCmd, strings.Fields(c.BenchArgs)...)

	stdlib := false
	if rootPath, err := c.runGitCmd("rev-parse", "--show-toplevel"); err == nil {
		// lib/time/zoneinfo.zip is a specific enough path, and it's here to
		// stay because it's one of the few paths hardcoded into Go binaries.
		zoneinfoPath := filepath.Join(string(rootPath), "lib", "time", "zoneinfo.zip")
		if _, err := os.Stat(zoneinfoPath); err == nil {
			stdlib = true
			c.debug().Println("standard library detected")
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

	if !stdlib {
		goVersion, err := c.runGoCmd("env", "GOVERSION")
		if err != nil {
			return err
		}
		fmt.Fprintf(fileBuffer, "go: %s\n", goVersion)
	}

	var runErr error
	if ref == "" {
		runErr = runCmd(cmd, c.debug())
	} else {
		err := c.runAtGitRef(c.BaseRef, func(workPath string) {
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

func (c *Benchdiff) Run() (result *RunResult, err error) {
	if err := os.MkdirAll(c.ResultsDir, 0o700); err != nil {
		return nil, err
	}

	headFlag := "--dirty"
	if c.HeadRef != "" {
		headFlag = c.HeadRef
	}
	headRef, err := c.runGitCmd("describe", "--tags", "--always", headFlag)
	if err != nil {
		return nil, err
	}

	baseRef, err := c.runGitCmd("describe", "--tags", "--always", c.BaseRef)
	if err != nil {
		return nil, err
	}

	baseFilename := fmt.Sprintf("benchdiff-%s-%s.out", baseRef, c.cacheKey())
	baseFilename = filepath.Join(c.ResultsDir, baseFilename)

	worktreeFilename := fmt.Sprintf("benchdiff-%s-%s.out", headRef, c.cacheKey())
	worktreeFilename = filepath.Join(c.ResultsDir, worktreeFilename)

	result = &RunResult{
		BenchmarkCmd:   fmt.Sprintf("%s %s", c.GoCmd, c.BenchArgs),
		HeadRef:        strings.TrimSpace(string(headRef)),
		BaseRef:        strings.TrimSpace(string(baseRef)),
		BaseOutputFile: baseFilename,
		HeadOutputFile: worktreeFilename,
	}

	err = c.runBenchmark(c.BaseRef, baseFilename, c.Force)
	if err != nil {
		return nil, err
	}

	err = c.runBenchmark(c.HeadRef, worktreeFilename, c.Force)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Benchdiff) runGoCmd(args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command(c.GoCmd, args...)
	cmd.Stdout = &stdout
	err := runCmd(cmd, c.debug())
	return bytes.TrimSpace(stdout.Bytes()), err
}

func (c *Benchdiff) runGitCmd(args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command(c.GitCmd, args...)
	cmd.Stdout = &stdout
	err := runCmd(cmd, c.debug())
	return bytes.TrimSpace(stdout.Bytes()), err
}

func (c *Benchdiff) runAtGitRef(ref string, fn func(path string)) error {
	worktree, err := os.MkdirTemp("", "benchdiff")
	if err != nil {
		return err
	}
	defer func() {
		rErr := os.RemoveAll(worktree)
		if rErr != nil {
			fmt.Printf("Could not delete temp directory: %s\n", worktree)
		}
	}()

	_, err = c.runGitCmd("worktree", "add", "--quiet", "--detach", worktree, ref)
	if err != nil {
		return err
	}

	defer func() {
		_, cerr := c.runGitCmd("worktree", "remove", worktree)
		if cerr != nil {
			if exitErr, ok := cerr.(*exec.ExitError); ok {
				fmt.Println(string(exitErr.Stderr))
			}
			fmt.Println(cerr)
		}
	}()
	fn(worktree)
	return nil
}
