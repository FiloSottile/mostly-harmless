package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitRepoRoot returns the repository root for the current directory.
func gitRepoRoot() (string, error) {
	return gitOutput("rev-parse", "--show-toplevel")
}

// gitCommonDir returns the git common dir (shared .git directory).
func gitCommonDir() (string, error) {
	out, err := gitOutput("rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	// --git-common-dir may return a relative path.
	if !filepath.IsAbs(out) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		out = filepath.Join(cwd, out)
	}
	return filepath.Clean(out), nil
}

// gitOutput runs a git command and returns its trimmed stdout.
func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// gitOutputDir runs a git command in a specific directory.
func gitOutputDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s (in %s): %w\n%s", strings.Join(args, " "), dir, err, stderr.String())
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// gitRun runs a git command, returning any error.
func gitRun(args ...string) error {
	_, err := gitOutput(args...)
	return err
}

// gitApplyCheck tests whether a patch applies cleanly.
func gitApplyCheck(repoRoot, diff string) error {
	cmd := exec.Command("git", "apply", "--check")
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader(diff)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

// gitApply applies a patch to a directory.
func gitApply(dir, diff string) error {
	cmd := exec.Command("git", "apply")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(diff)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

// gitApplyThreeWay applies a patch using three-way merge.
func gitApplyThreeWay(dir, diff string) error {
	cmd := exec.Command("git", "apply", "-3")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(diff)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

// gitDiffHEAD returns the diff of the working tree against HEAD.
func gitDiffHEAD(dir string) (string, error) {
	return gitOutputDir(dir, "diff", "HEAD")
}

// worktreeRoot returns the real repository root (parent of the git common dir).
// This is where .muzoo-worktrees/ should be created, even when running inside
// a git worktree.
func worktreeRoot() (string, error) {
	commonDir, err := gitCommonDir()
	if err != nil {
		return "", err
	}
	return filepath.Dir(commonDir), nil
}

// gitApplyCheckReverse tests whether a patch reverse-applies cleanly.
func gitApplyCheckReverse(repoRoot, diff string) error {
	cmd := exec.Command("git", "apply", "--check", "--reverse")
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader(diff)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

// gitCatFile returns the raw content of a git blob.
func gitCatFile(blob string) ([]byte, error) {
	cmd := exec.Command("git", "cat-file", "-p", blob)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git cat-file -p %s: %w\n%s", blob, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// worktreeDir returns the path for a mutation worktree.
func worktreeDir(wtRoot string, name string) string {
	return filepath.Join(wtRoot, ".muzoo-worktrees", name)
}

// ensureWorktreeParent creates the .muzoo-worktrees directory with a .gitignore.
func ensureWorktreeParent(wtRoot string) error {
	dir := filepath.Join(wtRoot, ".muzoo-worktrees")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ignorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		return os.WriteFile(ignorePath, []byte("*\n"), 0o644)
	}
	return nil
}

// createWorktree creates a detached worktree at the given path.
func createWorktree(path string) error {
	return gitRun("worktree", "add", "--detach", path, "HEAD")
}

// removeWorktree removes a worktree.
func removeWorktree(path string) error {
	return gitRun("worktree", "remove", "--force", path)
}
