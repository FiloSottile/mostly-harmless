# muzoo

A CLI tool for curated mutation testing.

A little zoo of mutations are hand-crafted (or LLM-generated) and stored
alongside a repository. `muzoo test` applies each mutation in a parallel git
worktree and runs tests to ensure they fail. Unlike automated mutation
frameworks, there are no equivalent-mutant exclusion lists to maintain.
`muzoo capture` makes it cheap to create new mutations.

## Install

```
go install filippo.io/mostly-harmless/muzoo@latest
```

Requires `git` in `$PATH`. Optionally uses [`mergiraf`](https://mergiraf.org)
for improved rebase conflict resolution.

## Usage

```
muzoo [-mutations <path>] <command> [args...]
```

The default mutations directory is `./testdata/mutations`.

### Capturing mutations

Save a change to the working tree that is supposed to break tests as a mutation
and restore tracked files to HEAD (`git restore .`).

```
muzoo capture -m "change >= to > in foo()"
```

If `-m` is omitted, `$EDITOR` opens the patch file directly. Type a description
above the diff and save.

If using Jujutsu, you can capture a mutation quickly by doing

```
jj new
# edit code to break tests
muzoo capture && jj squash -f @-
```

### Testing mutations

Run a test command against each mutation in parallel git worktrees. Mutations
that survive (tests still pass) indicate gaps in test coverage.

```
muzoo test
muzoo test -j 4 --timeout 30s -- make test
```

With no test command, `muzoo test` defaults to `go test -short ./... && go test
./...` — running short tests first, then full tests if needed — and prints the
name of the failed test(s) next to each killed mutation.

The working directory of the test command matches your current directory
relative to the repo root (e.g. if you run `muzoo test` from `src/foo`, the test
runs in `src/foo` inside the worktree).

Each test invocation has `MUZOO_PATCH` (patch file path) and `MUZOO_DESCRIPTION`
(description text) set.

### Other commands

```
muzoo status            # check which mutations apply cleanly
muzoo rebase            # update mutations to apply against current HEAD
muzoo list              # list mutations with descriptions
muzoo show <number>     # display a full patch
muzoo rm <number>       # delete a mutation
```

## Patch format

Patches are stored as `NNNN.patch` files. A patch file contains an optional
description (lines before the first `diff --git` line, separated by a blank
line) followed by a `git diff`.
