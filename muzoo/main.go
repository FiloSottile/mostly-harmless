package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(os.Stderr, "muzoo: %v\n", err)
		}
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

const helpText = `muzoo — a CLI tool for curated mutation testing.

A little zoo of mutations are hand-crafted (or LLM-generated) and stored
alongside a repository. "muzoo test" applies each mutation in a parallel git
worktree and runs tests to ensure they fail. Unlike automated mutation
frameworks, there are no equivalent-mutant exclusion lists to maintain.
"muzoo capture" makes it cheap to create new mutations.

Usage: muzoo [-mutations <path>] <command> [args...]

The default mutations directory is ./testdata/mutations.

Commands:

  capture [-m <message>]
      Save current unstaged changes as a new mutation and restore tracked files
      to HEAD (git restore .). If -m is omitted and Claude Code (claude) is
      available in $PATH, a description is generated automatically. If claude is
      not available, $EDITOR opens the patch file for you to type a description
      above the diff.

  test [-j <jobs>] [--timeout <duration>] [--] [test-command...]
      Run a test command against each mutation in parallel git worktrees.
      Mutations that survive (tests still pass) indicate gaps in test coverage.

      With no test command, defaults to "go test -short ./... && go test ./..."
      running short tests first, then full tests if needed, and prints the name
      of the failed test(s) next to each killed mutation.

      Results: KILLED (test failed, good), SURVIVED (test passed, bad),
      ERROR (worktree/apply error).

      Each test invocation has MUZOO_PATCH (patch file path) and
      MUZOO_DESCRIPTION (description text) set as environment variables.

  status
      Check which mutations apply cleanly. Shows OK, APPLIED (error — mutation
      is already part of the tree), or CONFLICT for each patch.

  rebase [--llm]
      Update mutations to apply against current HEAD. Uses three-way merge and
      optionally mergiraf for conflict resolution. With --llm, falls back to
      Claude for conflicts that git and mergiraf cannot resolve.

  list
      List mutations with their descriptions.

  show <number>
      Display a full patch.

  rm <number>
      Delete a mutation (does not renumber remaining patches).

Flags:

  -mutations <path>
      Path to mutations directory (default: ./testdata/mutations).

Patch format:

  Patches are stored as NNNN.patch files. A patch file contains an optional
  description (lines before the first "diff --git" line, separated by a blank
  line) followed by a git diff.

Exit codes:

  test:    0 = all killed, 1 = any survived/errored, 2 = setup error
  rebase:  0 = all rebased, 1 = any failed/lost, 2 = setup error
  status:  0 = all apply cleanly, 1 = any conflicts, 2 = setup error
  capture: 0 = saved, 2 = no changes

The spirit of this tool:

  Each mutation should represent a specific, meaningful code change that ought
  to be caught by the test suite. Good mutations target critical logic:
  boundary conditions, sign flips, off-by-one errors, removed security checks,
  swapped arguments, or deleted error handling. If a mutation survives, it
  reveals a real gap in test coverage. The set of mutations is curated and
  version-controlled, not generated in bulk — quality over quantity.

Examples:

  # Capture a mutation with a description
  muzoo capture -m "change >= to > in validateAge()"

  # Capture a mutation with an auto-generated description
  muzoo capture

  # Run the default Go test command against all mutations
  muzoo test

  # Run a custom test command with 4 parallel workers and 30s timeout
  muzoo test -j 4 --timeout 30s -- make test

  # Run pytest
  muzoo test -- uv run pytest

  # Check which mutations still apply cleanly
  muzoo status

  # Rebase mutations after updating the code
  muzoo rebase
  muzoo rebase --llm

  # List all mutations
  muzoo list

  # Show or delete a specific mutation
  muzoo show 3
  muzoo rm 3

Requires git in $PATH. Optionally uses mergiraf (https://mergiraf.org)
for improved rebase conflict resolution. With --llm, muzoo rebase can also
use Claude Code (https://claude.ai/download) for remaining conflicts.

Install: go install filippo.io/mostly-harmless/muzoo@latest
`

func run(args []string) error {
	// Parse global flags.
	f := flag.NewFlagSet("muzoo", flag.ContinueOnError)
	f.Usage = func() { fmt.Fprint(os.Stderr, helpText) }
	mutationsDir := f.String("mutations", "", "path to mutations directory")
	if err := f.Parse(args); err != nil {
		return err
	}
	args = f.Args()

	if len(args) == 0 {
		f.Usage()
		return &exitError{code: 2, msg: "no command specified"}
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
