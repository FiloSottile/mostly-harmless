package internal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
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
	Force       bool
	Cooldown    time.Duration
	WarmupCount int
	WarmupTime  string
	Debug       *log.Logger
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

func (c *Benchdiff) Run() (result *RunResult, err error) {
	if err := os.MkdirAll(c.ResultsDir, 0o700); err != nil {
		return nil, err
	}

	headRef, err := runGitCmd(c.debug(), c.gitCmd(), c.Path, "describe", "--tags", "--always", "--dirty")
	if err != nil {
		return nil, err
	}

	baseRef, err := runGitCmd(c.debug(), c.gitCmd(), c.Path, "describe", "--tags", "--always", c.BaseRef)
	if err != nil {
		return nil, err
	}

	baseFilename := fmt.Sprintf("benchdiff-%s-%s.out", baseRef, c.cacheKey())
	baseFilename = filepath.Join(c.ResultsDir, baseFilename)

	worktreeFilename := fmt.Sprintf("benchdiff-worktree-%s.out", c.cacheKey())
	worktreeFilename = filepath.Join(c.ResultsDir, worktreeFilename)

	result = &RunResult{
		BenchmarkCmd:   fmt.Sprintf("%s %s", c.BenchCmd, c.BenchArgs),
		HeadRef:        strings.TrimSpace(string(headRef)),
		BaseRef:        strings.TrimSpace(string(baseRef)),
		BaseOutputFile: baseFilename,
		HeadOutputFile: worktreeFilename,
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
