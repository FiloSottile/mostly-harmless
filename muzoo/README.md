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
for improved rebase conflict resolution. With `--llm`, `muzoo rebase` can also
use [Claude Code](https://claude.ai/download) (`claude`) for remaining conflicts.

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

If `-m` is omitted and [Claude Code](https://claude.ai/download) (`claude`) is
available in `$PATH`, a description is generated automatically. If `claude` is
not available, `$EDITOR` opens the patch file directly for
you to type a description above the diff.

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
muzoo test -- uv run pytest -x
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
muzoo rebase --llm      # also use Claude for unresolved conflicts
muzoo list              # list mutations with descriptions
muzoo show <number>     # display a full patch
muzoo rm <number>       # delete a mutation
```

## Patch format

Patches are stored as `NNNN.patch` files. A patch file contains an optional
description (lines before the first `diff --git` line, separated by a blank
line) followed by a `git diff`.

## Using LLMs to generate mutations

Here is an example prompt that yielded good results with Claude Code:

> ! muzoo -help

> make new mutations, focusing in particular on boundary conditions, e.g. everything that checks `> x` should be mutated to `> x + 1` and `> x - 1`
>
> make them by editing @src/mldsa/mldsa.py and then running
> ```
> muzoo -mutations ./tests/testdata/mutations/ capture -m "short description in 3-5 words"
> ```
>
> you can keep going in a loop
>
> look at @tests/testdata/mutations/ for examples
>
> at the end give me a summary of the ones you added and ideas for more

It helps to generate 2-3 examples manually to set the style.

You can ask the LLM to keep going and focus on security-relevant or plausible
human mistakes to perform differential mutation testing:

> can you think of more mutations that simulate plausible human mistakes or overlooked steps, or security-critical deviations, or bugs that are not likely in this implementation but might be in other
languages (like my case using strcmp instead of memcmp), or other mutations that are likely not to break naive test vectors (i.e. they don't break the whole computation but only some edge case)
