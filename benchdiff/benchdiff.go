// Command benchdiff runs Go benchmarks on two git refs and uses benchstat to
// show the delta.
//
// By default, the base ref is HEAD and the head ref is the current worktree.
// Use the -base-ref and -head-ref flags to specify different refs.
//
// To pass flags to "go test", pass them after a double dash. For example:
//
//	benchdiff -- -benchmem
//
// Non-worktree runs are cached. To clear the cache, use the -clear-cache flag.
//
// Benchmarking the standard library is supported.
//
// On macOS, benchdiff will attempt to prevent the system from sleeping.
//
// This is inspired by and based on github.com/willabides/benchdiff, although
// the interface has significantly diverged.
package main

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"

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
	reduildStdlibFlag := flag.Bool("rebuild-stdlib", true, "rebuild the standard library")
	debugFlag := flag.Bool("debug", false, "enable debug output")

	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Invoke caffeinate to prevent the system from sleeping. Best effort.
	pid := fmt.Sprintf("%d", os.Getpid())
	exec.CommandContext(ctx, "caffeinate", "-d", "-w", pid).Start()

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

	benchArgs := []string{"test", "-run", "^$", "-bench", ".", "-count", "6"}
	benchArgs = append(benchArgs, flag.Args()...)

	bd := &Benchdiff{
		BenchArgs:     benchArgs,
		ResultsDir:    getCacheDir(),
		BaseRef:       *baseRef,
		HeadRef:       *headRef,
		Debug:         log.New(io.Discard, "", 0),
		RebuildStdlib: *reduildStdlibFlag,
	}
	if *debugFlag {
		bd.Debug = log.New(os.Stderr, "", 0)
	}
	result, err := bd.Run(ctx)
	if err != nil {
		log.Fatalf("error running benchmarks: %v", err)
	}

	cmd := exec.CommandContext(ctx, "benchstat",
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
	BenchArgs     []string
	ResultsDir    string
	BaseRef       string
	HeadRef       string
	RebuildStdlib bool
	Stdlib        bool
	RootPath      string
	Debug         *log.Logger
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

var errCached = fmt.Errorf("cached")

func (c *Benchdiff) runBenchmark(ctx context.Context, ref, filename string, count int) error {
	c.Debug.Printf("output file: %s", filename)
	if ref != "" && fileExists(filename) {
		return errCached
	}

	progress := pb.Simple.Start(count)
	defer progress.Finish()

	cmd := exec.CommandContext(ctx, "go", c.BenchArgs...)
	if c.Stdlib {
		cmd.Path = filepath.Join(c.RootPath, "bin", "go")
	}

	fileBuffer := &bytes.Buffer{}
	cmd.Stdout = io.MultiWriter(fileBuffer, &LineWriter{f: func(line string) {
		if strings.HasPrefix(line, "Benchmark") && strings.Contains(line, "\t") {
			progress.Increment()
			parts := strings.Split(line, "\t")
			if len(parts) < 3 {
				return
			}
			name := strings.TrimSpace(parts[0])
			name, _, _ = strings.Cut(name, "-")
			time := strings.TrimSpace(parts[2])
			progress.Set("prefix", name+" "+time+" |")
		}
	}})

	if !c.Stdlib {
		goVersion, err := c.runGoCmd("env", "GOVERSION")
		if err != nil {
			return err
		}
		fmt.Fprintf(fileBuffer, "go: %s\n", goVersion)
	}

	goFIPS, err := c.runGoCmd("env", "GOFIPS140")
	if err != nil {
		return err
	}
	if goFIPS != "" {
		fmt.Fprintf(fileBuffer, "fips140: %s\n", goFIPS)
	}

	var runErr error
	if ref == "" {
		runErr = runCmd(cmd, c.Debug)
	} else {
		err := c.runAtGitRef(c.BaseRef, func(workPath string) {
			if c.Stdlib && c.RebuildStdlib {
				// TODO: cache toolchains.
				makeCmd := exec.CommandContext(ctx, filepath.Join(workPath, "src", "make.bash"))
				makeCmd.Dir = filepath.Join(workPath, "src")
				makeCmd.Env = append(os.Environ(), "GOOS=", "GOARCH=")
				makeCmd.Stdout = &LineWriter{f: func(line string) {
					words := strings.Fields(line)
					if strings.HasPrefix(line, "Building Go") {
						words = words[:3]
					}
					if strings.HasPrefix(line, "Building packages") {
						words = words[:4]
					}
					if strings.HasPrefix(line, "***") {
						words = []string{"Toolchain", "built"}
					}
					line = strings.Join(words, " ")
					progress.Set("prefix", line+" |")
				}}
				runErr = runCmd(makeCmd, c.Debug)
				if runErr != nil {
					return
				}
				cmd.Path = filepath.Join(workPath, "bin", "go")
			} else if c.Stdlib {
				runErr = os.Symlink(filepath.Join(c.RootPath, "pkg"), filepath.Join(workPath, "pkg"))
				if runErr != nil {
					return
				}
				runErr = os.Symlink(filepath.Join(c.RootPath, "bin"), filepath.Join(workPath, "bin"))
				if runErr != nil {
					return
				}
				cmd.Env = append(os.Environ(), "GOROOT="+workPath)
			}
			cmd.Dir = workPath // TODO: add relative path of working directory.
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

func (c *Benchdiff) countBenchmarks(ctx context.Context) (int, error) {
	var count int

	benchArgs := append([]string(nil), c.BenchArgs...)
	benchArgs = append(benchArgs, "-benchtime", "1x", "-run", "^$")
	cmd := exec.CommandContext(ctx, "go", benchArgs...)
	if c.Stdlib {
		cmd.Path = filepath.Join(c.RootPath, "bin", "go")
	}
	cmd.Stdout = &LineWriter{f: func(line string) {
		if strings.HasPrefix(line, "Benchmark") && strings.Contains(line, "\t") {
			count++
		}
	}}

	err := runCmd(cmd, c.Debug)
	return count, err
}

func (c *Benchdiff) Run(ctx context.Context) (result *RunResult, err error) {
	rootPath, err := c.runGitCmd("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	c.RootPath = string(rootPath)

	if c.HeadRef == "" {
		goModPath := filepath.Join(c.RootPath, "go.mod")
		if diff, err := c.runGitCmd("diff", goModPath); err == nil && len(diff) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: go.mod is dirty.\n")
		}
	}

	// lib/time/zoneinfo.zip is a specific enough path, and it's here to
	// stay because it's one of the few paths hardcoded into Go binaries.
	zoneinfoPath := filepath.Join(c.RootPath, "lib", "time", "zoneinfo.zip")
	if _, err := os.Stat(zoneinfoPath); err == nil {
		c.Stdlib = true
		c.Debug.Println("standard library detected")
	}

	if err := os.MkdirAll(c.ResultsDir, 0o700); err != nil {
		return nil, err
	}

	tagsFlag := "--tags"
	if c.Stdlib {
		// TODO: use env GOVERSION (with -dirty).
		tagsFlag = "--long"
	}
	headFlag := "--dirty"
	if c.HeadRef != "" {
		headFlag = c.HeadRef
	}
	headRef, err := c.runGitCmd("describe", tagsFlag, "--always", headFlag)
	if err != nil {
		return nil, err
	}
	headFilename, err := c.cacheFilename(string(headRef))
	if err != nil {
		return nil, err
	}

	baseRef, err := c.runGitCmd("describe", tagsFlag, "--always", c.BaseRef)
	if err != nil {
		return nil, err
	}
	baseFilename, err := c.cacheFilename(string(baseRef))
	if err != nil {
		return nil, err
	}

	// TODO: use base-ref cache if available.
	count, err := c.countBenchmarks(ctx)
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

	if err := c.runBenchmark(ctx, c.BaseRef, baseFilename, count); err == errCached {
		fmt.Fprintf(os.Stderr, "Using cached benchmark for %s.\n", result.BaseRef)
	} else if err != nil {
		return nil, err
	}

	if err := c.runBenchmark(ctx, c.HeadRef, headFilename, count); err == errCached {
		fmt.Fprintf(os.Stderr, "Using cached benchmark for %s.\n", result.HeadRef)
	} else if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Benchdiff) cacheFilename(ref string) (string, error) {
	env, err := c.runGoCmd("env", "GOARCH", "GOEXPERIMENT", "GOOS", "GOVERSION", "CC", "CXX", "CGO_ENABLED", "CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS", "GOFIPS140")
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
	fmt.Fprintf(h, "%s\n", c.RootPath)
	cacheKey := base64.RawURLEncoding.EncodeToString(h.Sum(nil)[:16])

	return filepath.Join(c.ResultsDir, fmt.Sprintf("benchdiff-%s.out", cacheKey)), nil
}

func (c *Benchdiff) runGoCmd(args ...string) (string, error) {
	var stdout bytes.Buffer
	cmd := exec.Command("go", args...)
	if c.Stdlib {
		cmd.Path = filepath.Join(c.RootPath, "bin", "go")
	}
	cmd.Stdout = &stdout
	err := runCmd(cmd, c.Debug)
	return string(bytes.TrimSpace(stdout.Bytes())), err
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
		_, cerr := c.runGitCmd("worktree", "remove", "--force", worktree)
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

type LineWriter struct {
	f   func(line string)
	buf string
}

func (w *LineWriter) Write(p []byte) (n int, err error) {
	w.buf += string(p)
	for {
		line, rest, ok := strings.Cut(w.buf, "\n")
		if !ok {
			return len(p), nil
		}
		w.f(line)
		w.buf = rest
	}
}
