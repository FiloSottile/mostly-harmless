// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
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

	jjConfig := strings.ReplaceAll(jjConfigTemplate, "$REVISIONS$", "")
	jjLog := func(args ...string) []string {
		args = append([]string{"--quiet", "--config-toml", jjConfig, "log", "--no-graph"}, args...)
		return lines(cmdOutput("jj", args...))
	}

	for _, rev := range jjLog("-T", "commit_id ++ '\n'", "-r", "mutable()") {
		changeID := trim(cmdOutput("git", "show", "-s", `--format=%(trailers:key=Change-Id,valueonly)`, rev))
		if changeID == "" {
			continue
		}
		merged := jjLog("-r", `::remote_bookmarks(remote=origin) & description("Change-Id: `+changeID+`")`)
		if len(merged) == 0 {
			continue
		}
		run("jj", "abandon", "-r", rev)
		printf("merged as %s", merged[0])
	}
}
