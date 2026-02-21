package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func cmdShow(mutDir string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: muzoo show <number>")
	}

	num, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid mutation number: %s", args[0])
	}

	filename := patchFilename(num)
	data, err := os.ReadFile(filepath.Join(mutDir, filename))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("mutation %s not found", args[0])
		}
		return err
	}

	fmt.Print(string(data))
	return nil
}
