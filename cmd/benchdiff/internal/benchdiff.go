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

	"golang.org/x/crypto/sha3"
)

// Benchdiff runs benchstats and outputs their deltas
type Benchdiff struct {
	BenchCmd   string
	BenchArgs  string
	ResultsDir string
	BaseRef    string
	HeadRef    string
	Path       string
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

func (c *Benchdiff) runBenchmark(ref, filename string, force bool) error {
	cmd := exec.Command(c.BenchCmd, strings.Fields(c.BenchArgs)...)

	stdlib := false
	if rootPath, err := runGitCmd(c.debug(), c.GitCmd, c.Path, "rev-parse", "--show-toplevel"); err == nil {
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
		goVersion, err := runGoCmd(c.debug(), c.BenchCmd, "env", "GOVERSION")
		if err != nil {
			return err
		}
		fmt.Fprintf(fileBuffer, "go: %s\n", goVersion)
	}

	var runErr error
	if ref == "" {
		runErr = runCmd(cmd, c.debug())
	} else {
		err := runAtGitRef(c.debug(), c.GitCmd, c.Path, c.BaseRef, func(workPath string) {
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
	headRef, err := runGitCmd(c.debug(), c.GitCmd, c.Path, "describe", "--tags", "--always", headFlag)
	if err != nil {
		return nil, err
	}

	baseRef, err := runGitCmd(c.debug(), c.GitCmd, c.Path, "describe", "--tags", "--always", c.BaseRef)
	if err != nil {
		return nil, err
	}

	baseFilename := fmt.Sprintf("benchdiff-%s-%s.out", baseRef, c.cacheKey())
	baseFilename = filepath.Join(c.ResultsDir, baseFilename)

	worktreeFilename := fmt.Sprintf("benchdiff-%s-%s.out", headRef, c.cacheKey())
	worktreeFilename = filepath.Join(c.ResultsDir, worktreeFilename)

	result = &RunResult{
		BenchmarkCmd:   fmt.Sprintf("%s %s", c.BenchCmd, c.BenchArgs),
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

func runGoCmd(debug *log.Logger, goCmd string, args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command(goCmd, args...)
	cmd.Stdout = &stdout
	err := runCmd(cmd, debug)
	return bytes.TrimSpace(stdout.Bytes()), err
}
