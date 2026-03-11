package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func cmdTest(mutDir, relDir string, args []string) error {
	f := flag.NewFlagSet("muzoo test", flag.ContinueOnError)
	jobs := f.Int("j", runtime.NumCPU(), "number of parallel jobs")
	timeout := f.Duration("timeout", 0, "timeout per test invocation")
	verbose := f.Bool("v", false, "show output for killed mutations")
	if err := f.Parse(args); err != nil {
		return err
	}
	testCmd := f.Args()

	defaultGoTest := len(testCmd) == 0
	if defaultGoTest {
		testCmd = []string{"go test -json -failfast -short ./... && go test -json -failfast ./..."}
	}

	pytestCmd := !defaultGoTest && isPytestCmd(testCmd)
	if pytestCmd {
		testCmd = addPytestFlags(testCmd)
	}

	// List and validate all patches.
	patches, err := listPatches(mutDir)
	if err != nil {
		return fmt.Errorf("listing patches: %w", err)
	}
	if len(patches) == 0 {
		fmt.Println("No mutations found.")
		return nil
	}

	// Use git common dir parent for worktree placement.
	wtRoot, err := worktreeRoot()
	if err != nil {
		return fmt.Errorf("finding repository root: %w", err)
	}

	if err := ensureWorktreeParent(wtRoot); err != nil {
		return fmt.Errorf("creating worktree directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Create or reuse worker worktrees, named by worker slot so the Go
	// build/test cache (keyed by absolute path) is shared across mutations.
	// Worktrees are kept across runs to preserve tool-managed directories
	// like .venv that are expensive to rebuild.
	workerPaths := make([]string, *jobs)
	for i := range workerPaths {
		workerPaths[i] = worktreeDir(wtRoot, strconv.Itoa(i))
		if err := reuseOrCreateWorktree(workerPaths[i]); err != nil {
			for j := range i {
				removeWorktree(workerPaths[j])
			}
			return fmt.Errorf("creating worktree: %w", err)
		}
	}

	// Pre-read and validate all patches against a clean worktree (at HEAD),
	// not the user's potentially-dirty working tree.
	type patchInfo struct {
		name string
		desc string
		diff string
	}
	var infos []patchInfo
	for _, p := range patches {
		desc, diff, err := readPatch(mutDir, p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		if err := gitApplyCheck(workerPaths[0], diff); err != nil {
			return &exitError{code: 2, msg: fmt.Sprintf("patch %s does not apply cleanly; run 'muzoo rebase' first", p)}
		}
		infos = append(infos, patchInfo{name: p, desc: desc, diff: diff})
	}

	type result struct {
		patch       string
		desc        string
		survived    bool
		errored     bool
		output      string
		killedTests string
	}

	results := make([]result, len(infos))
	// Pre-populate results so cancelled goroutines still have names.
	for i, info := range infos {
		results[i] = result{patch: info.name, desc: descriptionLabel(info.desc)}
	}

	// Worker pool: each slot is a worktree index.
	sem := make(chan int, *jobs)
	for i := range *jobs {
		sem <- i
	}
	var wg sync.WaitGroup

	testCmdStr := strings.Join(testCmd, " ")

	for i, info := range infos {
		wg.Add(1)
		go func(idx int, info patchInfo) {
			defer wg.Done()

			var worker int
			select {
			case <-ctx.Done():
				return
			case worker = <-sem:
			}
			defer func() { sem <- worker }()

			wtPath := workerPaths[worker]

			// Reset worktree to clean state for this mutation.
			if err := resetWorktree(wtPath); err != nil {
				results[idx].errored = true
				results[idx].output = "worktree reset failed: " + err.Error()
				return
			}

			// Apply patch.
			if err := gitApply(wtPath, info.diff); err != nil {
				results[idx].errored = true
				results[idx].output = "apply failed: " + err.Error()
				return
			}

			// Run test command.
			var cmd *exec.Cmd
			if *timeout > 0 {
				tctx, tcancel := context.WithTimeout(ctx, *timeout)
				defer tcancel()
				cmd = exec.CommandContext(tctx, "sh", "-c", testCmdStr)
			} else {
				cmd = exec.CommandContext(ctx, "sh", "-c", testCmdStr)
			}
			cmd.Dir = filepath.Join(wtPath, relDir)
			cmd.Env = append(os.Environ(),
				"MUZOO_PATCH="+info.name,
				"MUZOO_DESCRIPTION="+firstLine(info.desc),
			)
			// Use a process group so we can kill child processes on
			// timeout or signal, preventing orphaned test processes.
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Cancel = func() error {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				return nil
			}
			cmd.WaitDelay = time.Second
			var outBuf bytes.Buffer
			cmd.Stdout = &outBuf
			cmd.Stderr = &outBuf

			err := cmd.Run()
			output := outBuf.String()
			if defaultGoTest {
				output = formatGoTestOutput(output)
			} else if pytestCmd {
				output = formatPytestOutput(output)
			}
			if err == nil {
				// exit 0 = tests passed = mutation survived (BAD)
				results[idx].survived = true
				results[idx].output = output
			} else if ctx.Err() != nil {
				// Parent context cancelled (SIGINT/SIGTERM).
				return
			} else if errors.Is(err, context.DeadlineExceeded) {
				// Timeout expired = mutation killed (GOOD).
				results[idx].output = output
			} else {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) &&
					exitErr.ExitCode() != 126 && exitErr.ExitCode() != 127 {
					// Non-zero exit = tests failed = mutation killed (GOOD).
					results[idx].output = output
					if defaultGoTest {
						results[idx].killedTests = formatFailedTests(
							parseFailedTests(outBuf.String()), 3)
					} else if pytestCmd {
						results[idx].killedTests = formatFailedTests(
							parsePytestFailedTests(outBuf.String()), 1)
					}
				} else {
					// Infrastructure error: either not an ExitError (e.g.
					// working directory doesn't exist) or shell exit 126/127
					// (command not found or not executable).
					results[idx].errored = true
					results[idx].output = output + err.Error()
				}
			}
		}(i, info)
	}

	wg.Wait()

	signal.Stop(sigCh)

	// If interrupted, don't print a misleading partial summary.
	if ctx.Err() != nil {
		return &exitError{code: 2, msg: "interrupted"}
	}

	// Print results.
	killed := 0
	survivedCount := 0
	errorCount := 0
	for _, r := range results {
		num := strings.TrimSuffix(r.patch, ".patch")
		switch {
		case r.errored:
			fmt.Printf("%s  ERROR     %s\n", num, r.desc)
			errorCount++
		case r.survived:
			fmt.Printf("%s  SURVIVED  %s\n", num, r.desc)
			survivedCount++
		default:
			fmt.Printf("%s  KILLED    %s%s\n", num, r.desc, r.killedTests)
			killed++
		}
	}

	// Print output for errored mutations, and killed if verbose.
	for _, r := range results {
		show := (r.errored || *verbose) && r.output != ""
		if show {
			fmt.Printf("\n--- Output for %s (%s) ---\n%s\n", strings.TrimSuffix(r.patch, ".patch"), r.desc, r.output)
		}
	}

	total := killed + survivedCount + errorCount
	fmt.Printf("\n%d/%d mutations killed.", killed, total)
	if survivedCount > 0 {
		fmt.Printf(" %d survived.", survivedCount)
	}
	if errorCount > 0 {
		fmt.Printf(" %d errored.", errorCount)
	}
	fmt.Println()

	if survivedCount > 0 || errorCount > 0 {
		return &exitError{code: 1, msg: fmt.Sprintf("%d mutation(s) survived, %d errored", survivedCount, errorCount)}
	}
	return nil
}

// parseFailedTests extracts unique leaf failed test names from go test -json output.
func parseFailedTests(output string) []string {
	type testEvent struct {
		Action string `json:"Action"`
		Test   string `json:"Test"`
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var ev testEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Action == "fail" && ev.Test != "" {
			seen[ev.Test] = true
		}
	}
	// Filter to leaf tests only (exclude parents of subtests).
	var failed []string
	for t := range seen {
		isParent := false
		for t2 := range seen {
			if t2 != t && strings.HasPrefix(t2, t+"/") {
				isParent = true
				break
			}
		}
		if !isParent {
			failed = append(failed, t)
		}
	}
	sort.Strings(failed)
	return failed
}

// formatGoTestOutput extracts human-readable output from go test -json lines.
func formatGoTestOutput(output string) string {
	type testEvent struct {
		Action string `json:"Action"`
		Output string `json:"Output"`
	}
	var b strings.Builder
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			if line != "" {
				b.WriteString(line)
				b.WriteByte('\n')
			}
			continue
		}
		var ev testEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		if ev.Action == "output" {
			b.WriteString(ev.Output)
		}
	}
	return b.String()
}

// formatFailedTests returns a short summary of failed tests for display.
// maxShow controls how many test names to include before truncating.
func formatFailedTests(tests []string, maxShow int) string {
	if len(tests) == 0 {
		return ""
	}
	if len(tests) <= maxShow {
		return " [" + strings.Join(tests, ", ") + "]"
	}
	return fmt.Sprintf(" [%s, ... +%d more]", strings.Join(tests[:maxShow], ", "), len(tests)-maxShow)
}

// isPytestCmd returns true if the test command runs pytest.
func isPytestCmd(testCmd []string) bool {
	cmd := strings.Join(testCmd, " ")
	cmd = strings.TrimSpace(cmd)
	return cmd == "pytest" || cmd == "uv run pytest" ||
		strings.HasPrefix(cmd, "pytest ") || strings.HasPrefix(cmd, "uv run pytest ")
}

// addPytestFlags adds -v and --tb=short to a pytest command for parseable output.
func addPytestFlags(testCmd []string) []string {
	cmd := strings.Join(testCmd, " ")
	cmd = strings.TrimSpace(cmd)
	// Insert flags right after "pytest".
	if strings.HasPrefix(cmd, "uv run pytest") {
		cmd = strings.Replace(cmd, "uv run pytest", "uv run pytest -v --tb=short", 1)
	} else {
		cmd = strings.Replace(cmd, "pytest", "pytest -v --tb=short", 1)
	}
	return []string{cmd}
}

// parsePytestFailedTests extracts failed test names from pytest -v output.
// It looks for lines like "FAILED tests/test_foo.py::test_bar" in the
// short test summary section.
func parsePytestFailedTests(output string) []string {
	var failed []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "FAILED ") {
			continue
		}
		// "FAILED tests/test_foo.py::TestClass::test_bar - reason"
		name := strings.TrimPrefix(line, "FAILED ")
		if dash := strings.Index(name, " - "); dash != -1 {
			name = name[:dash]
		}
		// Extract just the test function/method name (after last ::).
		if idx := strings.LastIndex(name, "::"); idx != -1 {
			name = name[idx+2:]
		}
		failed = append(failed, name)
	}
	sort.Strings(failed)
	return failed
}

// formatPytestOutput trims pytest output to the most useful parts.
// With -v --tb=short, the output is already fairly concise, so we
// just return it as-is.
func formatPytestOutput(output string) string {
	return output
}
