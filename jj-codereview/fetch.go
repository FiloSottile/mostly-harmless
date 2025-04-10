// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
)

func cmdFetch(args []string) {
	flags.Usage = func() {
		fmt.Fprintf(stderr(), trim(`
Usage: %s fetch %s <query>

Fetches a single CL by number, Change-Id, or other Gerrit query.
`), progName, globalFlags)
		exit(2)
	}
	flags.Parse(args)
	if flags.NArg() != 1 {
		flags.Usage()
	}

	config := jjConfig()
	defer os.Remove(config)
	jjLog := jjLog(config)

	c, err := GetChange(flags.Arg(0))
	if err != nil {
		dief("failed to fetch change: %v", err)
	}

	run("git", "fetch", c.Revisions[c.CurrentRevision].Fetch.HTTP.URL, c.Revisions[c.CurrentRevision].Fetch.HTTP.Ref)
	ref := fmt.Sprintf("refs/remotes/gerrit/cl/%d/%d", c.Number, c.Revisions[c.CurrentRevision].Number)
	if !*noRun {
		run("git", "update-ref", ref, "FETCH_HEAD")
		for _, c := range jjLog("-T", "commit_id ++ '\n'", "-r", "::"+c.CurrentRevision+" ~ ::remote_bookmarks(remote=origin) ~ "+c.CurrentRevision) {
			labelCommit(c, 5)
		}
		printf("%s", jjLog("-r", c.CurrentRevision)[0])
	}
}
