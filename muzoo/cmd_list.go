package main

import (
	"fmt"
	"strings"
)

func cmdList(mutDir string, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: muzoo list")
	}

	patches, err := listPatches(mutDir)
	if err != nil {
		return fmt.Errorf("listing patches: %w", err)
	}
	if len(patches) == 0 {
		fmt.Println("No mutations found.")
		return nil
	}

	for _, p := range patches {
		desc, _, err := readPatch(mutDir, p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		num := strings.TrimSuffix(p, ".patch")
		fmt.Printf("%s  %s\n", num, descriptionLabel(desc))
	}
	return nil
}
