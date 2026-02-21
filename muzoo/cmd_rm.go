package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func cmdRm(mutDir string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: muzoo rm <number>")
	}

	num, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid mutation number: %s", args[0])
	}

	filename := patchFilename(num)
	path := filepath.Join(mutDir, filename)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("mutation %s not found", args[0])
		}
		return err
	}

	fmt.Printf("Removed mutation %s\n", strings.TrimSuffix(filename, ".patch"))
	return nil
}
