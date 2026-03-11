# muzoo — Curated Mutation Testing Tool

## Project Overview

`muzoo` is a CLI tool for curated mutation testing. Mutations are hand-crafted
(or LLM-assisted) patches stored alongside a repository. The tool helps create,
rebase, and test these mutations against a test suite, failing if any mutation
survives (tests still pass). Unlike automated mutation frameworks, mutations are
version-controlled and curated — no equivalent-mutant exclusion lists needed.

## Architecture

Single Go binary, no dependencies beyond the standard library and `git` in
`$PATH`. Optional `mergiraf` integration for better rebase success.

### File Layout

```
main.go          — entry point, CLI dispatch
patch.go         — patch parsing, numbering, file I/O
git.go           — git operations (apply, diff, worktrees)
cmd_capture.go   — muzoo capture
cmd_status.go    — muzoo status
cmd_rebase.go    — muzoo rebase (three-way merge, mergiraf)
cmd_testing.go   — muzoo test (parallel worktree execution)
cmd_list.go      — muzoo list
cmd_show.go      — muzoo show
cmd_rm.go        — muzoo rm
```

### Key Design Decisions

- All git operations shell out to `git` — no git library.
- Never uses the git index/staging area. Uses `git diff` for capture
  (unstaged changes only), ensuring `git restore .` cleanly undoes the capture.
- Patches are parsed by finding the first `diff --git ` line; everything before
  is the description, trailing blank lines stripped.
- `test` creates worktrees under `.muzoo-worktrees/` at the repo root (via
  `git rev-parse --git-common-dir`), with a self-ignoring `.gitignore`.
  Worktrees are reused across runs (updated to current HEAD) to preserve
  tool-managed directories like `.venv`. `git clean -fd -e .venv` is used
  between mutations to preserve virtual environments.
- Test commands run via `sh -c` for pipe/redirect support.
- `MUZOO_PATCH` and `MUZOO_DESCRIPTION` env vars set for each test invocation.

### Patch Format

Stored in the mutations directory (default `./testdata/mutations`), named
sequentially as `NNNN.patch` (zero-padded to 4 digits, 5 if >9999). A patch
file is either a bare `git diff` or a `git diff` preceded by an optional
description header (one or more lines before the first `diff --git` line,
separated by a blank line). Gaps in numbering are allowed (after `rm`).
`capture` finds the highest existing number and increments.

### Exit Codes

| Command   | 0                    | 1                      | 2            |
|-----------|----------------------|------------------------|--------------|
| `test`    | All killed           | Any survived/errored   | Setup error  |
| `rebase`  | All rebased          | Any failed/lost        | Setup error  |
| `status`  | All apply cleanly    | Any conflicts          | Setup error  |
| `capture` | Saved                | —                      | No changes   |

## Commands

### `capture [-m <message>]`

Saves current unstaged changes (`git diff`) as a new mutation. If `-m`
is not provided, tries to generate a description using Claude Code (`claude`
CLI with Sonnet model). If `claude` is not available or fails, falls back to
opening `$EDITOR` on the patch file with empty lines at the top for the
description. Staged changes are not captured.
After saving, restores the working tree to match the index (`git restore .`).

### `status`

Shows state of each mutation: `OK` (applies cleanly), `APPLIED` (already
applied — an error, since mutations should not be part of the tree), or
`CONFLICT`. Exits 1 if any patches are `APPLIED` or `CONFLICT`.

### `rebase`

Updates mutations to apply against current HEAD. For each conflicting patch,
tries: (1) `git apply -3` three-way merge, (2) `mergiraf` if available, (3)
flags as `CONFLICT`. Detects mutations lost during rebase (`LOST`). Already
applied patches are flagged as `APPLIED` errors (should be removed with
`muzoo rm`).

### `test [-j <jobs>] [--timeout <duration>] [--] [test-command...]`

Runs test command against each mutation in parallel worktrees. Pre-checks all
patches apply cleanly (exits 2 if not). Results: `KILLED` (test failed, good),
`SURVIVED` (test passed, bad), `ERROR` (worktree/apply error). Shows captured
stdout/stderr for errored mutations. Timeout expiry counts as
killed. Default `-j` is number of CPUs. Signal handling cleans up worktrees on
SIGINT/SIGTERM. With no test command, defaults to
`go test -json -failfast -short ./... && go test -json -failfast ./...` and prints the failed
test(s) next to each killed mutation. When the test command is `pytest` or
`uv run pytest` (with or without extra arguments), `-x -v --tb=short` flags are
added automatically and failed test names are shown next to each killed
mutation.

### `list`, `show <number>`, `rm <number>`

List mutations with descriptions, display a full patch, or delete a mutation
(does not renumber).

## Development

```sh
go build ./...          # compile check
go run . <command>      # run without installing
go build -o muzoo .     # build binary
```

## Testing Manually

Create a test git repo, add some code with tests, then:
```sh
# Make a change, capture it
sed -i 's/foo/bar/' file.go
muzoo capture -m "change foo to bar"

# Check status, test mutations
muzoo status
muzoo test -- go test ./...
```

## Future Directions (out of scope for v1)

- **`muzoo generate`**: LLM-proposed mutations for a file or diff, applied to
  working tree for review, then captured normally.
- **`muzoo rebase --llm`**: LLM-assisted conflict resolution when `git apply -3`
  and mergiraf both fail.
- **`muzoo cover`**: Generate coverage profile (Go/LCOV) marking lines touched
  by any mutation, for overlaying with test coverage.
- **PR-scoped mutations**: `muzoo generate --diff HEAD~1` to generate mutations
  only for changed code, for use in PR checks.
