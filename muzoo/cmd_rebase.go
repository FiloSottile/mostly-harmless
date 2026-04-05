package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdRebase(repoRoot, mutDir string, args []string) error {
	f := flag.NewFlagSet("muzoo rebase", flag.ContinueOnError)
	useLLM := f.Bool("llm", false, "use Claude to resolve conflicts that git and mergiraf cannot")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("usage: muzoo rebase [--llm]")
	}

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

	hasMergiraf := hasMergirafBinary()
	if !hasMergiraf {
		fmt.Fprintln(os.Stderr, "note: install mergiraf for better rebase success (https://mergiraf.org)")
	}

	var claudePath string
	if *useLLM {
		var err error
		claudePath, err = exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("--llm requires claude in $PATH (https://claude.ai/download)")
		}
	}

	anyFailed := false
	for _, p := range patches {
		desc, diff, err := readPatch(mutDir, p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}

		num := strings.TrimSuffix(p, ".patch")

		// Check if it already applies cleanly.
		if err := gitApplyCheck(repoRoot, diff); err == nil {
			fmt.Printf("%s  OK        %s\n", num, descriptionLabel(desc))
			continue
		}

		// Check if it's already applied (reverse-applies cleanly).
		if gitApplyCheckReverse(repoRoot, diff) == nil {
			fmt.Printf("%s  APPLIED   %s (mutation is part of the tree; remove with 'muzoo rm')\n", num, descriptionLabel(desc))
			anyFailed = true
			continue
		}

		newDiff, method, err := rebaseInWorktree(wtRoot, num, diff, desc, hasMergiraf, claudePath)
		if err != nil {
			fmt.Printf("%s  CONFLICT  %s (could not rebase automatically)\n", num, descriptionLabel(desc))
			anyFailed = true
			continue
		}
		if newDiff == "" {
			fmt.Printf("%s  LOST      %s (mutation lost during rebase)\n", num, descriptionLabel(desc))
			anyFailed = true
			continue
		}
		content := formatPatch(desc, newDiff)
		if err := os.WriteFile(filepath.Join(mutDir, p), []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", p, err)
		}
		fmt.Printf("%s  REBASED   %s (%s)\n", num, descriptionLabel(desc), method)
	}

	if anyFailed {
		return &exitError{code: 1, msg: "some patches could not be rebased"}
	}
	return nil
}

// rebaseInWorktree tries to rebase a conflicting patch in a single worktree,
// cascading through strategies: three-way merge, then mergiraf per-file
// (keeping partial successes), then LLM for any remaining failed files.
// Returns (newDiff, method, error).
func rebaseInWorktree(wtRoot, num, diff, desc string, hasMergiraf bool, claudePath string) (string, string, error) {
	wtPath := worktreeDir(wtRoot, "rebase-"+num)
	if err := createWorktree(wtPath); err != nil {
		return "", "", err
	}
	defer removeWorktree(wtPath)

	// Strategy 1: git apply -3 (handles the whole patch at once).
	if err := gitApplyThreeWay(wtPath, diff); err == nil {
		newDiff, err := gitDiffHEAD(wtPath)
		if err != nil {
			return "", "", err
		}
		if newDiff == "" {
			return "", "three-way merge", nil // lost
		}
		return newDiff + "\n", "three-way merge", nil
	}
	// Reset worktree after failed three-way merge (it leaves conflict markers).
	if _, err := gitOutputDir(wtPath, "checkout", "HEAD", "--", "."); err != nil {
		return "", "", fmt.Errorf("resetting worktree after failed three-way merge: %w", err)
	}

	// For per-file strategies, parse the diff into files.
	files := parseDiffFiles(diff)
	if len(files) == 0 {
		return "", "", fmt.Errorf("no files in diff")
	}

	// Track which files still need resolving.
	remaining := make(map[int]bool)
	for i := range files {
		remaining[i] = true
	}

	// Strategy 2: mergiraf per-file (partial successes are kept).
	usedMergiraf := false
	if hasMergiraf {
		for i, f := range files {
			if err := mergirafFile(wtPath, f); err == nil {
				delete(remaining, i)
				usedMergiraf = true
			} else {
				fmt.Fprintf(os.Stderr, "  mergiraf failed for %s: %v\n", f.path, err)
			}
		}
	}

	// Strategy 3: LLM per-file for anything mergiraf couldn't resolve.
	usedLLM := false
	if len(remaining) > 0 && claudePath != "" {
		for i, f := range files {
			if !remaining[i] {
				continue
			}
			if err := llmRebaseFile(claudePath, wtPath, f, desc); err == nil {
				delete(remaining, i)
				usedLLM = true
			} else {
				fmt.Fprintf(os.Stderr, "  llm failed for %s: %v\n", f.path, err)
			}
		}
	}

	// Build method string from what actually succeeded.
	var method string
	switch {
	case usedMergiraf && usedLLM:
		method = "mergiraf+llm"
	case usedLLM:
		method = "llm"
	case usedMergiraf:
		method = "mergiraf"
	}

	if len(remaining) > 0 {
		return "", "", fmt.Errorf("%d file(s) could not be rebased", len(remaining))
	}

	newDiff, err := gitDiffHEAD(wtPath)
	if err != nil {
		return "", "", err
	}
	if newDiff == "" {
		return "", method, nil // lost
	}
	return newDiff + "\n", method, nil
}

// mergirafFile tries to resolve a single file's conflict using mergiraf.
// It modifies the file in the worktree in place on success.
func mergirafFile(wtPath string, f diffFile) error {
	if f.oldBlob == "" || strings.TrimLeft(f.oldBlob, "0") == "" {
		return fmt.Errorf("no old blob for %s (new file)", f.path)
	}

	baseContent, err := gitCatFile(f.oldBlob)
	if err != nil {
		return fmt.Errorf("getting base content for %s: %w", f.path, err)
	}

	headPath := filepath.Join(wtPath, f.path)
	headContent, err := os.ReadFile(headPath)
	if err != nil {
		return fmt.Errorf("reading HEAD content for %s: %w", f.path, err)
	}

	tmpDir, err := os.MkdirTemp("", "muzoo-mergiraf-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	basePath := filepath.Join(tmpDir, "base")
	mutatedPath := filepath.Join(tmpDir, "mutated")
	headFilePath := filepath.Join(tmpDir, "head")
	resolvedPath := filepath.Join(tmpDir, "resolved")

	if err := os.WriteFile(basePath, baseContent, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(mutatedPath, baseContent, 0o644); err != nil {
		return err
	}

	// Apply hunk to base to get mutated content.
	fileDiff := rewriteDiffPaths(f.diff, "mutated")
	applyCmd := exec.Command("git", "apply", "--unsafe-paths", "--directory="+tmpDir)
	applyCmd.Stdin = strings.NewReader(fileDiff)
	applyCmd.Dir = tmpDir
	if out, err := applyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("applying hunk to base: %w\n%s", err, out)
	}
	if err := os.WriteFile(headFilePath, headContent, 0o644); err != nil {
		return err
	}

	mgCmd := exec.Command("mergiraf", "merge", basePath, headFilePath, mutatedPath, "-o", resolvedPath)
	if out, err := mgCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mergiraf failed for %s: %w\n%s", f.path, err, out)
	}

	resolved, err := os.ReadFile(resolvedPath)
	if err != nil {
		return err
	}
	return os.WriteFile(headPath, resolved, 0o644)
}

// llmRebaseFile uses Claude to resolve a single file's conflict.
// It reads the current worktree content (which may include mergiraf's
// partial work) and asks Claude to apply the mutation's intent.
func llmRebaseFile(claude, wtPath string, f diffFile, desc string) error {
	headPath := filepath.Join(wtPath, f.path)
	headContent, err := os.ReadFile(headPath)
	if err != nil {
		return fmt.Errorf("reading content for %s: %w", f.path, err)
	}

	var prompt strings.Builder
	prompt.WriteString("You are rebasing a mutation testing patch. The patch no longer applies cleanly to the current code.\n\n")
	if desc != "" {
		prompt.WriteString("The mutation is described as: ")
		prompt.WriteString(firstLine(desc))
		prompt.WriteString("\n\n")
	}
	prompt.WriteString("Here is the current version of the file ")
	prompt.WriteString(f.path)
	prompt.WriteString(":\n\n```\n")
	prompt.Write(headContent)
	prompt.WriteString("```\n\n")
	prompt.WriteString("Here is the patch that used to apply to an older version of this file:\n\n```diff\n")
	prompt.WriteString(f.diff)
	prompt.WriteString("```\n\n")
	prompt.WriteString("Apply the INTENT of this mutation to the current file. Output ONLY the complete file contents, nothing else. No markdown fences, no explanation.")

	cmd := exec.Command(claude, "--no-session-persistence", "-p", prompt.String())
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return fmt.Errorf("claude failed for %s: %w\n%s", f.path, err, exitErr.Stderr)
		}
		return fmt.Errorf("claude failed for %s: %w", f.path, err)
	}

	mutated := bytes.TrimSpace(out)
	// Strip markdown fences if Claude added them despite instructions.
	if bytes.HasPrefix(mutated, []byte("```")) {
		if i := bytes.IndexByte(mutated, '\n'); i != -1 {
			mutated = mutated[i+1:]
		}
		if bytes.HasSuffix(mutated, []byte("```")) {
			mutated = mutated[:len(mutated)-3]
		}
	}
	mutated = bytes.TrimRight(mutated, "\n \t")
	if len(mutated) == 0 {
		return fmt.Errorf("claude returned empty content for %s", f.path)
	}
	mutated = append(mutated, '\n')

	return os.WriteFile(headPath, mutated, 0o644)
}

type diffFile struct {
	path    string
	oldBlob string
	diff    string
}

// parseDiffFiles splits a unified diff into per-file sections.
func parseDiffFiles(diff string) []diffFile {
	lines := strings.Split(diff, "\n")
	var files []diffFile
	var current *diffFile
	var currentLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				current.diff = strings.Join(currentLines, "\n")
				files = append(files, *current)
			}
			current = &diffFile{}
			currentLines = []string{line}
			continue
		}
		if current != nil {
			// Parse path from "+++ b/path" which is more reliable than
			// splitting the "diff --git" line (handles paths with spaces).
			if strings.HasPrefix(line, "+++ b/") {
				current.path = strings.TrimPrefix(line, "+++ b/")
			}
			if strings.HasPrefix(line, "index ") {
				// Parse "index abc1234..def5678 100644"
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					blobs := strings.SplitN(parts[1], "..", 2)
					if len(blobs) >= 1 {
						current.oldBlob = blobs[0]
					}
				}
			}
			currentLines = append(currentLines, line)
		}
	}
	if current != nil {
		current.diff = strings.Join(currentLines, "\n")
		files = append(files, *current)
	}
	return files
}

// rewriteDiffPaths rewrites file paths in diff header lines only.
func rewriteDiffPaths(diff, newName string) string {
	var result strings.Builder
	for _, line := range strings.SplitAfter(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			// Reconstruct the diff --git line precisely.
			line = "diff --git a/" + newName + " b/" + newName + "\n"
		case strings.HasPrefix(line, "--- a/"):
			line = "--- a/" + newName + "\n"
		case strings.HasPrefix(line, "+++ b/"):
			line = "+++ b/" + newName + "\n"
		}
		result.WriteString(line)
	}
	return result.String()
}

func hasMergirafBinary() bool {
	_, err := exec.LookPath("mergiraf")
	return err == nil
}
