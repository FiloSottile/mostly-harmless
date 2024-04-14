package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
)

func getCacheDir() string {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("error finding user cache dir: %v", err)
	}
	return filepath.Join(userCacheDir, "benchdiff")
}

func main() {
	clearCacheFlag := flag.Bool("clear-cache", false, "clear the cache")
	baseRef := flag.String("base-ref", "HEAD", "base git ref")
	headRef := flag.String("head-ref", "", "head git ref (defaults to worktree)")
	debugFlag := flag.Bool("debug", false, "enable debug output")

	flag.Parse()

	if *clearCacheFlag {
		cacheDir := getCacheDir()
		files, err := filepath.Glob(filepath.Join(cacheDir, "benchdiff-*.out"))
		if err != nil {
			log.Fatalf("error finding files in %s: %v", cacheDir, err)
		}
		for _, file := range files {
			err = os.Remove(file)
			if err != nil {
				log.Fatalf("error removing %s: %v", file, err)
			}
		}
		os.Exit(0)
	}

	benchArgs := []string{"test", "-json", "-run", "^$", "-bench", ".", "-count", "6"}
	benchArgs = append(benchArgs, flag.Args()...)

	bd := &Benchdiff{
		BenchArgs:  benchArgs,
		ResultsDir: getCacheDir(),
		BaseRef:    *baseRef,
		HeadRef:    *headRef,
		Debug:      log.New(io.Discard, "", 0),
	}
	if *debugFlag {
		bd.Debug = log.New(os.Stderr, "", 0)
	}
	result, err := bd.Run()
	if err != nil {
		log.Fatalf("error running benchmarks: %v", err)
	}

	cmd := exec.Command("benchstat",
		result.BaseRef+"="+result.BaseOutputFile,
		result.HeadRef+"="+result.HeadOutputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	bd.Debug.Printf("+ %s", cmd)
	if err := cmd.Run(); err != nil {
		log.Fatalf("error running benchstat: %v", err)
	}
}

type Benchdiff struct {
	BenchArgs  []string
	ResultsDir string
	BaseRef    string
	HeadRef    string
	Debug      *log.Logger
}

type RunResult struct {
	HeadOutputFile string
	BaseOutputFile string
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

func runCmd(cmd *exec.Cmd, debug *log.Logger) error {
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
		err = fmt.Errorf("error running command: %s\nexit code: %d\nstderr: \n%s", cmd.String(), exitErr.ExitCode(), bufStderr.String())
	}
	return err
}

func (c *Benchdiff) runBenchmark(ref, filename string, count int) error {
	c.Debug.Printf("output file: %s", filename)
	if ref != "" && fileExists(filename) {
		c.Debug.Printf("+ skipping benchmark for ref %q because output file exists", ref)
		return nil
	}

	// TODO: customize. Add ETA.
	progress := pb.StartNew(count)
	defer progress.Finish()

	cmd := exec.Command("go", c.BenchArgs...)

	stdlib := false
	if rootPath, err := c.runGitCmd("rev-parse", "--show-toplevel"); err == nil {
		// lib/time/zoneinfo.zip is a specific enough path, and it's here to
		// stay because it's one of the few paths hardcoded into Go binaries.
		zoneinfoPath := filepath.Join(string(rootPath), "lib", "time", "zoneinfo.zip")
		if _, err := os.Stat(zoneinfoPath); err == nil {
			stdlib = true
			c.Debug.Println("standard library detected")
			cmd.Path = filepath.Join(string(rootPath), "bin", "go")
		}
	}

	fileBuffer := &bytes.Buffer{}
	cmd.Stdout = &TestJSONWriter{f: func(e *TestEvent) {
		if e.Action == "output" {
			io.WriteString(fileBuffer, e.Output)
		}
		if e.Action == "output" && strings.Contains(e.Output, "\t") &&
			strings.HasPrefix(e.Output, "Benchmark") {
			// TODO: print more information about the benchmark.
			progress.Increment()
		}
	}}

	if !stdlib {
		goVersion, err := c.runGoCmd("env", "GOVERSION")
		if err != nil {
			return err
		}
		fmt.Fprintf(fileBuffer, "go: %s\n", goVersion)
	}

	var runErr error
	if ref == "" {
		runErr = runCmd(cmd, c.Debug)
	} else {
		err := c.runAtGitRef(c.BaseRef, func(workPath string) {
			if stdlib {
				makeCmd := exec.Command(filepath.Join(workPath, "src", "make.bash"))
				makeCmd.Dir = filepath.Join(workPath, "src")
				makeCmd.Env = append(os.Environ(), "GOOS=", "GOARCH=")
				runErr = runCmd(makeCmd, c.Debug)
				if runErr != nil {
					return
				}
				cmd.Path = filepath.Join(workPath, "bin", "go")
			}
			cmd.Dir = workPath // TODO: add relative path of working directory
			runErr = runCmd(cmd, c.Debug)
		})
		if err != nil {
			return err
		}
	}
	if runErr != nil {
		return runErr
	}
	return os.WriteFile(filename, fileBuffer.Bytes(), 0o666)
}

func (c *Benchdiff) countBenchmarks() (int, error) {
	var count int

	benchArgs := append([]string(nil), c.BenchArgs...)
	benchArgs = append(benchArgs, "-benchtime", "1ns", "-run", "^$")
	cmd := exec.Command("go", benchArgs...)
	cmd.Stdout = &TestJSONWriter{f: func(e *TestEvent) {
		// Unfortunately, the go test -json output makes it hard to track timing
		// output lines without heuristics. See https://go.dev/issue/66825.
		if e.Action == "output" && strings.Contains(e.Output, "\t") &&
			strings.HasPrefix(e.Output, "Benchmark") {
			count++
		}
	}}

	err := runCmd(cmd, c.Debug)
	return count, err
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
	headFilename, err := c.cacheFilename(string(headRef))
	if err != nil {
		return nil, err
	}

	baseRef, err := c.runGitCmd("describe", "--tags", "--always", c.BaseRef)
	if err != nil {
		return nil, err
	}
	baseFilename, err := c.cacheFilename(string(baseRef))
	if err != nil {
		return nil, err
	}

	// TODO: use base-ref cache if available.
	count, err := c.countBenchmarks()
	if err != nil {
		return nil, err
	}
	c.Debug.Printf("counted %d benchmarks", count)

	result = &RunResult{
		HeadRef:        strings.TrimSpace(string(headRef)),
		BaseRef:        strings.TrimSpace(string(baseRef)),
		BaseOutputFile: baseFilename,
		HeadOutputFile: headFilename,
	}

	// TODO: interleave runs?

	if err := c.runBenchmark(c.BaseRef, baseFilename, count); err != nil {
		return nil, err
	}

	if err := c.runBenchmark(c.HeadRef, headFilename, count); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Benchdiff) cacheFilename(ref string) (string, error) {
	env, err := c.runGoCmd("env", "GOARCH", "GOEXPERIMENT", "GOOS", "GOVERSION", "CC", "CXX", "CGO_ENABLED", "CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS")
	if err != nil {
		return "", err
	}
	rootPath, err := c.runGitCmd("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	h := sha512.New()
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		fmt.Fprintf(h, "%s\n", buildInfo.String())
	}
	fmt.Fprintf(h, "%q\n", c.BenchArgs)
	fmt.Fprintf(h, "%s\n", env)
	fmt.Fprintf(h, "%s\n", ref)
	fmt.Fprintf(h, "%s\n", rootPath)
	cacheKey := base64.RawURLEncoding.EncodeToString(h.Sum(nil)[:16])

	return filepath.Join(c.ResultsDir, fmt.Sprintf("benchdiff-%s.out", cacheKey)), nil
}

func (c *Benchdiff) runGoCmd(args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command("go", args...)
	cmd.Stdout = &stdout
	err := runCmd(cmd, c.Debug)
	return bytes.TrimSpace(stdout.Bytes()), err
}

func (c *Benchdiff) runGitCmd(args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdout
	err := runCmd(cmd, c.Debug)
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

type TestEvent struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string
}

type TestJSONWriter struct {
	f   func(e *TestEvent)
	buf []byte
}

func (w *TestJSONWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)

	var offset int64
	defer func() { w.buf = w.buf[offset:] }()
	d := json.NewDecoder(bytes.NewReader(w.buf))
	for {
		e := &TestEvent{}
		err := d.Decode(e)
		if err == io.EOF {
			return len(p), nil
		}
		if err != nil {
			return 0, err
		}
		offset = d.InputOffset()
		w.f(e)
	}
}
