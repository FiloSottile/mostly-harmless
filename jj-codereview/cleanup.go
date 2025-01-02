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

Abandons all visible mailed revisions (in "remote_bookmarks(remote=gerrit)")
that were merged (in "remote_bookmarks(remote=origin)").
`), progName, globalFlags)
		exit(2)
	}
	flags.Parse(args)

	jjConfig := strings.ReplaceAll(jjConfigTemplate, "$REVISIONS$", "")
	jjLog := func(args ...string) []string {
		args = append([]string{"--config-toml", jjConfig, "log", "--no-graph"}, args...)
		return lines(cmdOutput("jj", args...))
	}

	for _, rev := range jjLog("-T", "commit_id ++ '\n'", "-r", "all() & remote_bookmarks(remote=gerrit)") {
		changeID := trim(cmdOutput("git", "show", "-s", `--format=%(trailers:key=Change-Id,valueonly)`, rev))
		merged := jjLog("-r", `::remote_bookmarks(remote=origin) & description("Change-Id: `+changeID+`")`)
		if len(merged) == 0 {
			continue
		}
		printf("%s\nwas merged as\n%s", jjLog("-r", rev)[0], merged[0])
		run("jj", "abandon", "-r", rev)
	}
}
