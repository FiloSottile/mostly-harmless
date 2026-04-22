// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strings"
)

func cmdCleanup(args []string) {
	flags.Usage = func() {
		fmt.Fprintf(stderr(), trim(`
Usage: %s cleanup %s

Abandons mutable changes that were merged in a origin branch.
`), progName, globalFlags)
		exit(2)
	}
	flags.Parse(args)

	config := jjConfig()
	defer os.Remove(config)
	jjLog := jjLog(config)

	mergedByChangeID := map[string]string{}
	for _, line := range nonBlankLines(cmdOutput("git", "log",
		"--format=%H %(trailers:key=Change-Id,valueonly)",
		"--since=45 days ago", "--remotes=origin")) {
		commit, changeID, ok := strings.Cut(line, " ")
		if !ok || changeID == "" {
			continue
		}
		if _, ok := mergedByChangeID[changeID]; !ok {
			mergedByChangeID[changeID] = commit
		}
	}

	for _, rev := range jjLog("-T", "commit_id ++ '\n'", "-r", "mutable()") {
		changeID := trim(cmdOutput("git", "show", "-s", `--format=%(trailers:key=Change-Id,valueonly)`, rev))
		if changeID == "" {
			continue
		}
		mergedRev, ok := mergedByChangeID[changeID]
		if !ok {
			continue
		}
		for _, child := range jjLog("-T", "commit_id ++ '\n'", "-r", "children("+rev+")") {
			run("jj", "rebase", "-s", child, "-d", mergedRev)
		}
		run("jj", "abandon", "-r", rev)
		printf("merged as %s", mergedRev)
	}
}
