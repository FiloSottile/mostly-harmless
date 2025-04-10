// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
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
