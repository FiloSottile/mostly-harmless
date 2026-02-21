package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// parsePatch splits a patch file into its description and diff portions.
// The diff starts at the first line beginning with "diff --git ".
func parsePatch(data string) (description, diff string) {
	lines := strings.SplitAfter(data, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			desc := strings.Join(lines[:i], "")
			desc = strings.TrimRight(desc, "\n")
			diff = strings.Join(lines[i:], "")
			return desc, diff
		}
	}
	// No diff --git line found; treat entire content as diff (unusual).
	return "", data
}

// firstLine returns the first line of s, trimmed of whitespace.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// descriptionLabel returns the first line of the description, or "(no description)".
func descriptionLabel(desc string) string {
	if desc == "" {
		return "(no description)"
	}
	return firstLine(desc)
}

// formatPatch combines a description and diff into a patch file.
func formatPatch(description, diff string) string {
	if description == "" {
		return diff
	}
	return description + "\n\n" + diff
}

// patchNumber extracts the number from a patch filename like "0001.patch".
func patchNumber(name string) (int, error) {
	name = strings.TrimSuffix(name, ".patch")
	n, err := strconv.Atoi(name)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, fmt.Errorf("invalid patch number: %d", n)
	}
	return n, nil
}

// patchFilename formats a number as a patch filename.
func patchFilename(n int) string {
	if n > 9999 {
		return fmt.Sprintf("%05d.patch", n)
	}
	return fmt.Sprintf("%04d.patch", n)
}

// listPatches returns sorted patch filenames in the mutations directory.
func listPatches(mutDir string) ([]string, error) {
	entries, err := os.ReadDir(mutDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var patches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".patch") {
			patches = append(patches, e.Name())
		}
	}
	sort.Slice(patches, func(i, j int) bool {
		ni, _ := patchNumber(patches[i])
		nj, _ := patchNumber(patches[j])
		return ni < nj
	})
	return patches, nil
}

// nextPatchNumber returns the next available patch number.
func nextPatchNumber(mutDir string) (int, error) {
	patches, err := listPatches(mutDir)
	if err != nil {
		return 0, err
	}
	if len(patches) == 0 {
		return 1, nil
	}
	highest := 0
	for _, p := range patches {
		n, err := patchNumber(p)
		if err != nil {
			continue
		}
		if n > highest {
			highest = n
		}
	}
	return highest + 1, nil
}

// readPatch reads and parses a patch file.
func readPatch(mutDir, name string) (description, diff string, err error) {
	data, err := os.ReadFile(filepath.Join(mutDir, name))
	if err != nil {
		return "", "", err
	}
	desc, d := parsePatch(string(data))
	return desc, d, nil
}
