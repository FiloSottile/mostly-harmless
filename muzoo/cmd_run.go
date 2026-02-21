package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

func cmdRun(repoRoot, mutDir, relDir string, args []string) error {
	f := flag.NewFlagSet("muzoo run", flag.ContinueOnError)
	jobs := f.Int("j", runtime.NumCPU(), "number of parallel jobs")
	timeout := f.Duration("timeout", 0, "timeout per test invocation")
	if err := f.Parse(args); err != nil {
		return err
	}
	testCmd := f.Args()

	if len(testCmd) == 0 {
		return fmt.Errorf("usage: muzoo run [-j <jobs>] [-timeout <duration>] [--] <test-command...>")
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

	// Pre-read and validate all patches.
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
		if err := gitApplyCheck(repoRoot, diff); err != nil {
			return &exitError{code: 2, msg: fmt.Sprintf("patch %s does not apply cleanly; run 'muzoo rebase' first", p)}
		}
		infos = append(infos, patchInfo{name: p, desc: desc, diff: diff})
	}

	// Use git common dir parent for worktree placement.
	wtRoot, err := worktreeRoot()
	if err != nil {
		return fmt.Errorf("finding repository root: %w", err)
	}

	if err := ensureWorktreeParent(wtRoot); err != nil {
		return fmt.Errorf("creating worktree directory: %w", err)
	}

	var worktreeMu sync.Mutex
	activeWorktrees := make(map[string]bool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	type result struct {
		patch    string
		desc     string
		survived bool
		errored  bool
		output   string
	}

	results := make([]result, len(infos))
	// Pre-populate results so cancelled goroutines still have names.
	for i, info := range infos {
		results[i] = result{patch: info.name, desc: descriptionLabel(info.desc)}
	}

	sem := make(chan struct{}, *jobs)
	var wg sync.WaitGroup

	testCmdStr := strings.Join(testCmd, " ")

	for i, info := range infos {
		wg.Add(1)
		go func(idx int, info patchInfo) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			num := strings.TrimSuffix(info.name, ".patch")
			wtPath := worktreeDir(wtRoot, num)

			// Register worktree before creation to avoid race with signal handler.
			worktreeMu.Lock()
			activeWorktrees[wtPath] = true
			worktreeMu.Unlock()

			if err := createWorktree(wtPath); err != nil {
				worktreeMu.Lock()
				delete(activeWorktrees, wtPath)
				worktreeMu.Unlock()
				results[idx].errored = true
				results[idx].output = "worktree creation failed: " + err.Error()
				return
			}

			defer func() {
				removeWorktree(wtPath)
				worktreeMu.Lock()
				delete(activeWorktrees, wtPath)
				worktreeMu.Unlock()
			}()

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
			var outBuf bytes.Buffer
			cmd.Stdout = &outBuf
			cmd.Stderr = &outBuf

			err := cmd.Run()
			if err == nil {
				// exit 0 = tests passed = mutation survived (BAD)
				results[idx].survived = true
				results[idx].output = outBuf.String()
			} else if ctx.Err() != nil {
				// Parent context cancelled (SIGINT/SIGTERM).
				return
			} else {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) &&
					exitErr.ExitCode() != 126 && exitErr.ExitCode() != 127 {
					// Non-zero exit = tests failed = mutation killed (GOOD).
					// Also treat timeout as killed.
					results[idx].output = outBuf.String()
				} else {
					// Infrastructure error: either not an ExitError (e.g.
					// working directory doesn't exist) or shell exit 126/127
					// (command not found or not executable).
					results[idx].errored = true
					results[idx].output = outBuf.String() + err.Error()
				}
			}
		}(i, info)
	}

	wg.Wait()

	signal.Stop(sigCh)

	// Clean up any remaining worktrees (from interrupted goroutines).
	worktreeMu.Lock()
	wts := make([]string, 0, len(activeWorktrees))
	for wt := range activeWorktrees {
		wts = append(wts, wt)
	}
	worktreeMu.Unlock()
	for _, wt := range wts {
		removeWorktree(wt)
	}

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
			fmt.Printf("%s  KILLED    %s\n", num, r.desc)
			killed++
		}
	}

	// Print output for survived and errored mutations.
	for _, r := range results {
		if (r.survived || r.errored) && r.output != "" {
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
