package main

import (
	"fmt"
	"strings"
)

func cmdStatus(repoRoot, mutDir string, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: muzoo status")
	}

	patches, err := listPatches(mutDir)
	if err != nil {
		return fmt.Errorf("listing patches: %w", err)
	}
	if len(patches) == 0 {
		fmt.Println("No mutations found.")
		return nil
	}

	hasProblems := false
	for _, p := range patches {
		desc, diff, err := readPatch(mutDir, p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}

		num := strings.TrimSuffix(p, ".patch")
		label := descriptionLabel(desc)

		if err := gitApplyCheck(repoRoot, diff); err == nil {
			fmt.Printf("%s  OK        %s\n", num, label)
		} else if gitApplyCheckReverse(repoRoot, diff) == nil {
			fmt.Printf("%s  APPLIED   %s\n", num, label)
			hasProblems = true
		} else {
			fmt.Printf("%s  CONFLICT  %s\n", num, label)
			hasProblems = true
		}
	}

	if hasProblems {
		return &exitError{code: 1, msg: "some patches have conflicts or are already applied"}
	}
	return nil
}
