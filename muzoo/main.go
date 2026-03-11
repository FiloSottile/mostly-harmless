package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "muzoo: %v\n", err)
		if exitErr, ok := err.(*exitError); ok {
			os.Exit(exitErr.code)
		}
		os.Exit(2)
	}
}

type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func run(args []string) error {
	// Parse global flags.
	f := flag.NewFlagSet("muzoo", flag.ContinueOnError)
	mutationsDir := f.String("mutations", "", "path to mutations directory")
	if err := f.Parse(args); err != nil {
		return err
	}
	args = f.Args()

	if len(args) == 0 {
		return fmt.Errorf("usage: muzoo [-mutations <path>] <command> [args...]")
	}

	cmd := args[0]
	args = args[1:]

	// Find repo root and resolve mutations directory.
	repoRoot, err := gitRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	if *mutationsDir == "" {
		*mutationsDir = repoRoot + "/testdata/mutations"
	}

	switch cmd {
	case "capture":
		return cmdCapture(repoRoot, *mutationsDir, args)
	case "status":
		return cmdStatus(repoRoot, *mutationsDir, args)
	case "rebase":
		return cmdRebase(repoRoot, *mutationsDir, args)
	case "test":
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		// Resolve symlinks to match git's --show-toplevel output.
		cwd, err = filepath.EvalSymlinks(cwd)
		if err != nil {
			return fmt.Errorf("resolving working directory: %w", err)
		}
		relDir, err := filepath.Rel(repoRoot, cwd)
		if err != nil {
			return fmt.Errorf("computing relative directory: %w", err)
		}
		return cmdTest(*mutationsDir, relDir, args)
	case "list":
		return cmdList(*mutationsDir, args)
	case "show":
		return cmdShow(*mutationsDir, args)
	case "rm":
		return cmdRm(*mutationsDir, args)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}
