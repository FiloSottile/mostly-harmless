package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdCapture(repoRoot, mutDir string, args []string) error {
	f := flag.NewFlagSet("muzoo capture", flag.ContinueOnError)
	message := f.String("m", "", "mutation description")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("usage: muzoo capture [-m <message>]")
	}
	hasMessage := *message != ""

	// Get unstaged diff (working tree vs index). This matches what
	// "git restore ." will undo after capture.
	diff, err := gitOutputDir(repoRoot, "diff")
	if err != nil {
		return fmt.Errorf("getting diff: %w", err)
	}
	if diff == "" {
		return &exitError{code: 2, msg: "no changes to capture"}
	}
	diff += "\n" // Ensure trailing newline.

	// Create mutations directory if needed.
	if err := os.MkdirAll(mutDir, 0o755); err != nil {
		return fmt.Errorf("creating mutations directory: %w", err)
	}

	// Determine next number.
	num, err := nextPatchNumber(mutDir)
	if err != nil {
		return fmt.Errorf("determining patch number: %w", err)
	}

	// Write patch file.
	filename := patchFilename(num)
	patchPath := filepath.Join(mutDir, filename)
	content := formatPatch(*message, diff)
	if err := os.WriteFile(patchPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing patch: %w", err)
	}

	// If no -m, try to generate a description with Claude Code, then
	// fall back to opening $EDITOR.
	if !hasMessage {
		if desc := generateDescription(diff); desc != "" {
			*message = desc
			if err := os.WriteFile(patchPath, []byte(formatPatch(*message, diff)), 0o644); err != nil {
				return fmt.Errorf("writing patch: %w", err)
			}
		} else {
			if err := editPatch(patchPath, diff); err != nil {
				os.Remove(patchPath)
				return err
			}
		}
	}

	// Read back the final description for the confirmation message.
	data, err := os.ReadFile(patchPath)
	if err != nil {
		return fmt.Errorf("reading patch: %w", err)
	}
	description, _ := parsePatch(string(data))

	// Restore tracked files to HEAD.
	if _, err := gitOutputDir(repoRoot, "restore", "."); err != nil {
		return fmt.Errorf("restoring working tree: %w", err)
	}

	// Print confirmation.
	numStr := strings.TrimSuffix(filename, ".patch")
	if description != "" {
		fmt.Printf("Saved mutation %s: %s\n", numStr, firstLine(description))
	} else {
		fmt.Printf("Saved mutation %s\n", numStr)
	}
	return nil
}

// generateDescription tries to use Claude Code to generate a short description
// for the given diff. Returns empty string if claude is not available or fails.
func generateDescription(diff string) string {
	claude, err := exec.LookPath("claude")
	if err != nil {
		return ""
	}
	prompt := "Describe this mutation (a change to source code that should be caught by tests) " +
		"in two to five-ish words, first letter lowercase. " +
		"Output only the description, nothing else.\n\n" + diff
	cmd := exec.Command(claude, "--model", "sonnet", "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	desc := strings.TrimSpace(string(out))
	if desc == "" {
		return ""
	}
	return desc
}

func editPatch(patchPath, diff string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Write the patch with empty lines at the top for the description.
	if err := os.WriteFile(patchPath, []byte("\n\n"+diff), 0o644); err != nil {
		return err
	}

	// Open editor. Use sh -c to support EDITOR values with arguments
	// (e.g. "code --wait").
	cmd := exec.Command("sh", "-c", editor+` "$@"`, "--", patchPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}
	return nil
}
