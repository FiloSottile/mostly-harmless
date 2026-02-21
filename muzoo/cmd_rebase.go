package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdRebase(repoRoot, mutDir string, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: muzoo rebase")
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

		// Try three-way merge in a worktree.
		newDiff, err := rebaseThreeWay(wtRoot, num, diff)
		if err == nil && newDiff != "" {
			content := formatPatch(desc, newDiff)
			if err := os.WriteFile(filepath.Join(mutDir, p), []byte(content), 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", p, err)
			}
			fmt.Printf("%s  REBASED   %s (three-way merge)\n", num, descriptionLabel(desc))
			continue
		}
		if err == nil && newDiff == "" {
			fmt.Printf("%s  LOST      %s (mutation lost during rebase)\n", num, descriptionLabel(desc))
			anyFailed = true
			continue
		}

		// Try mergiraf if available.
		if hasMergiraf {
			newDiff, err := rebaseMergiraf(wtRoot, num, diff)
			if err == nil && newDiff != "" {
				content := formatPatch(desc, newDiff)
				if err := os.WriteFile(filepath.Join(mutDir, p), []byte(content), 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", p, err)
				}
				fmt.Printf("%s  REBASED   %s (mergiraf)\n", num, descriptionLabel(desc))
				continue
			}
			if err == nil && newDiff == "" {
				fmt.Printf("%s  LOST      %s (mutation lost during rebase)\n", num, descriptionLabel(desc))
				anyFailed = true
				continue
			}
		}

		// Failed to rebase.
		fmt.Printf("%s  CONFLICT  %s (could not rebase automatically)\n", num, descriptionLabel(desc))
		anyFailed = true
	}

	if anyFailed {
		return &exitError{code: 1, msg: "some patches could not be rebased"}
	}
	return nil
}

// rebaseThreeWay tries to rebase a patch using git apply -3 in a temporary worktree.
// Returns the new diff, or empty string if the mutation was lost.
func rebaseThreeWay(wtRoot, num, diff string) (string, error) {
	wtPath := worktreeDir(wtRoot, "rebase-"+num)

	if err := createWorktree(wtPath); err != nil {
		return "", err
	}
	defer removeWorktree(wtPath)

	if err := gitApplyThreeWay(wtPath, diff); err != nil {
		return "", err
	}

	// Generate new diff.
	newDiff, err := gitDiffHEAD(wtPath)
	if err != nil {
		return "", err
	}
	if newDiff == "" {
		return "", nil // mutation lost
	}
	return newDiff + "\n", nil
}

// rebaseMergiraf tries to rebase using mergiraf for each file in the patch.
func rebaseMergiraf(wtRoot, num, diff string) (string, error) {
	files := parseDiffFiles(diff)
	if len(files) == 0 {
		return "", fmt.Errorf("no files in diff")
	}

	wtPath := worktreeDir(wtRoot, "rebase-"+num)
	if err := createWorktree(wtPath); err != nil {
		return "", err
	}
	defer removeWorktree(wtPath)

	for _, f := range files {
		if f.oldBlob == "" || strings.TrimLeft(f.oldBlob, "0") == "" {
			return "", fmt.Errorf("no old blob for %s (new file)", f.path)
		}

		// Get base content (raw bytes, preserving trailing newline).
		baseContent, err := gitCatFile(f.oldBlob)
		if err != nil {
			return "", fmt.Errorf("getting base content for %s: %w", f.path, err)
		}

		// Get current HEAD content.
		headPath := filepath.Join(wtPath, f.path)
		headContent, err := os.ReadFile(headPath)
		if err != nil {
			return "", fmt.Errorf("reading HEAD content for %s: %w", f.path, err)
		}

		// Apply hunk to base to get mutated content.
		tmpDir, err := os.MkdirTemp("", "muzoo-mergiraf-*")
		if err != nil {
			return "", err
		}

		basePath := filepath.Join(tmpDir, "base")
		mutatedPath := filepath.Join(tmpDir, "mutated")
		headFilePath := filepath.Join(tmpDir, "head")
		resolvedPath := filepath.Join(tmpDir, "resolved")

		if err := os.WriteFile(basePath, baseContent, 0o644); err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}

		// Write base content to mutated, then apply hunk.
		if err := os.WriteFile(mutatedPath, baseContent, 0o644); err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}
		// Use git apply on just this file's diff.
		fileDiff := rewriteDiffPaths(f.diff, "mutated")
		applyCmd := exec.Command("git", "apply", "--unsafe-paths", "--directory="+tmpDir)
		applyCmd.Stdin = strings.NewReader(fileDiff)
		applyCmd.Dir = tmpDir
		if out, err := applyCmd.CombinedOutput(); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("applying hunk to base: %w\n%s", err, out)
		}
		if err := os.WriteFile(headFilePath, headContent, 0o644); err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}

		// Run mergiraf.
		mgCmd := exec.Command("mergiraf", "merge", basePath, headFilePath, mutatedPath, "-o", resolvedPath)
		if out, err := mgCmd.CombinedOutput(); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("mergiraf failed for %s: %w\n%s", f.path, err, out)
		}

		// Copy resolved content back to worktree.
		resolved, err := os.ReadFile(resolvedPath)
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}
		if err := os.WriteFile(headPath, resolved, 0o644); err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}

		// Clean up temp dir immediately, not deferred.
		os.RemoveAll(tmpDir)
	}

	// Generate new diff.
	newDiff, err := gitDiffHEAD(wtPath)
	if err != nil {
		return "", err
	}
	if newDiff == "" {
		return "", nil
	}
	return newDiff + "\n", nil
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
