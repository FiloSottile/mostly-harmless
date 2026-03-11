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

	// Compute a pathspec exclude for the mutations directory so that
	// git diff and git restore don't interact with mutation patch files.
	absMutDir, err := filepath.Abs(mutDir)
	if err != nil {
		return fmt.Errorf("resolving mutations dir: %w", err)
	}
	relMutDir, err := filepath.Rel(repoRoot, absMutDir)
	if err != nil {
		return fmt.Errorf("computing relative mutations dir: %w", err)
	}
	exclude := fmt.Sprintf(":(exclude)%s", relMutDir)

	// Get unstaged diff (working tree vs index). This matches what
	// "git restore ." will undo after capture. The mutations directory
	// is excluded so we don't capture changes to patch files themselves.
	diff, err := gitOutputDir(repoRoot, "diff", "--", ".", exclude)
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

	// Check if Claude Code is available for description generation before
	// restoring, so we know whether to fall back to $EDITOR (which needs
	// the working tree intact for context).
	needDescription := !hasMessage
	hasClaude := false
	if needDescription {
		_, err := exec.LookPath("claude")
		hasClaude = err == nil
	}

	// If we need $EDITOR (no -m and no claude), open it before restoring
	// so the user can see the working tree for context.
	if needDescription && !hasClaude {
		if err := editPatch(patchPath, diff); err != nil {
			os.Remove(patchPath)
			return err
		}
	}

	// Restore tracked files to match the index, excluding the mutations
	// directory so git restore doesn't overwrite or recreate patch files.
	if _, err := gitOutputDir(repoRoot, "restore", "--", ".", exclude); err != nil {
		return fmt.Errorf("restoring working tree: %w", err)
	}

	numStr := strings.TrimSuffix(filename, ".patch")

	// Generate description with Claude Code after restoring, so the user
	// can start working on the next mutation while this runs.
	if needDescription && hasClaude {
		fmt.Printf("Saved mutation %s\n", numStr)
		if desc := generateDescription(mutDir, diff); desc != "" {
			if err := os.WriteFile(patchPath, []byte(formatPatch(desc, diff)), 0o644); err != nil {
				return fmt.Errorf("writing patch: %w", err)
			}
			fmt.Printf("Mutation %s: %s\n", numStr, desc)
		}
	} else {
		// Read back description from -m or $EDITOR for confirmation.
		data, err := os.ReadFile(patchPath)
		if err != nil {
			return fmt.Errorf("reading patch: %w", err)
		}
		description, _ := parsePatch(string(data))
		if description != "" {
			fmt.Printf("Saved mutation %s: %s\n", numStr, firstLine(description))
		} else {
			fmt.Printf("Saved mutation %s\n", numStr)
		}
	}

	return nil
}

// generateDescription tries to use Claude Code to generate a short description
// for the given diff. Returns empty string if claude is not available or fails.
func generateDescription(mutDir, diff string) string {
	claude, err := exec.LookPath("claude")
	if err != nil {
		return ""
	}

	// Collect existing descriptions as examples for consistency.
	examples := existingDescriptions(mutDir)
	if len(examples) == 0 {
		examples = []string{
			"z norm check reduced to last element only",
			"skip non-zero hint padding check",
		}
	}

	var prompt strings.Builder
	prompt.WriteString("Describe this mutation (a change to source code that should be caught by tests) " +
		"in two to five-ish words, first letter lowercase. " +
		"Output only the description, nothing else.\n\n")
	prompt.WriteString("Examples of existing descriptions for reference:\n")
	for _, ex := range examples {
		prompt.WriteString("- ")
		prompt.WriteString(ex)
		prompt.WriteString("\n")
	}
	prompt.WriteString("\n")
	prompt.WriteString(diff)

	cmd := exec.Command(claude, "--model", "sonnet", "--no-session-persistence", "-p", prompt.String())
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

// existingDescriptions returns the first line of each non-empty description
// from existing patches in the mutations directory.
func existingDescriptions(mutDir string) []string {
	patches, err := listPatches(mutDir)
	if err != nil {
		return nil
	}
	var descs []string
	for _, p := range patches {
		desc, _, err := readPatch(mutDir, p)
		if err != nil || desc == "" {
			continue
		}
		descs = append(descs, firstLine(desc))
	}
	return descs
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
