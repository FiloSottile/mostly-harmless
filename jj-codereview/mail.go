// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var jjConfigTemplate = `
[revset-aliases]
"jjcrmail()" = '''$REVISIONS$'''
"jjcrmailpending()" = "remote_bookmarks(remote=origin)..jjcrmail()"
"jjcrbranch(x)" = "remote_bookmarks(remote=origin) & roots(remote_bookmarks(remote=origin)..x)-::"

[templates]
bookmarks = "separate('\n', remote_bookmarks.map(|b| if(b.remote() == 'origin', b.name()))) ++ '\n'"

[ui]
color = "never"
paginate = "never"
`

func cmdMail(args []string) {
	// NOTE: New flags should be added to the usage message below as well as doc.go.
	var (
		rList  = new(stringList) // installed below
		ccList = new(stringList) // installed below

		all         = flags.Bool("a", false, "mail multiple heads")
		hashtagList = new(stringList) // installed below
		trybot      = flags.Bool("trybot", false, "run trybots on the uploaded CLs")
		wip         = flags.Bool("wip", false, "set the status of a change to Work-in-Progress")
		autoSubmit  = flags.Bool("autosubmit", false, "set autosubmit on the uploaded CLs")
	)
	flags.Var(rList, "r", "comma-separated list of reviewers")
	flags.Var(ccList, "cc", "comma-separated list of people to CC:")
	flags.Var(hashtagList, "hashtag", "comma-separated list of tags to set")

	flags.Usage = func() {
		fmt.Fprintf(stderr(),
			"Usage: %s mail %s [-r reviewer,...] [-cc mail,...]\n"+
				"\t[-autosubmit] [-trybot] [-wip] [-hashtag tag,...]\n"+
				"\t[revisions]\n", progName, globalFlags)
		fmt.Fprintf(stderr(), "\n")
		fmt.Fprintf(stderr(), "Mails all changes in remote_bookmarks(remote=origin)..revisions.\n")
		fmt.Fprintf(stderr(), "If revisions is not specified, it's @-.\n")
		exit(2)
	}
	flags.Parse(args)
	if len(flags.Args()) > 1 {
		flags.Usage()
		exit(2)
	}

	var trybotVotes []string
	switch os.Getenv("GIT_CODEREVIEW_TRYBOT") {
	case "", "luci":
		trybotVotes = []string{"Commit-Queue+1"}
	case "farmer":
		trybotVotes = []string{"Run-TryBot"}
	case "both":
		trybotVotes = []string{"Commit-Queue+1", "Run-TryBot"}
	default:
		fmt.Fprintf(stderr(), "GIT_CODEREVIEW_TRYBOT must be unset, blank, or one of 'luci', 'farmer', or 'both'\n")
		exit(2)
	}

	jjc := strings.ReplaceAll(jjConfigTemplate, "$REVISIONS$", "@-")
	if len(flags.Args()) == 1 {
		jjc = strings.ReplaceAll(jjConfigTemplate, "$REVISIONS$", flags.Arg(0))
	}

	if log := trim(cmdOutput("jj", "--config-toml", jjc, "log", "--no-graph",
		"-r", "jjcrmailpending() & conflicts()")); log != "" {
		dief("the following pending changes have conflicts:\n%s", log)
	}
	if log := trim(cmdOutput("jj", "--config-toml", jjc, "log", "--no-graph",
		"-r", "jjcrmailpending() & (empty() ~ merges())")); log != "" {
		dief("the following pending changes are empty:\n%s", log)
	}
	if log := trim(cmdOutput("jj", "--config-toml", jjc, "log", "--no-graph",
		"-r", `jjcrmailpending() & description('substring-i:"DO NOT MAIL"')`)); log != "" {
		dief("the following pending changes say DO NOT MAIL:\n%s", log)
	}
	private, _ := cmdOutputErr("jj", "--config-toml", jjc, "config", "get", "git.private-commits")
	if private != "" {
		if log := trim(cmdOutput("jj", "--config-toml", jjc, "log", "--no-graph",
			"-r", "jjcrmailpending() & ("+private+")")); log != "" {
			dief("the following pending changes are private:\n%s", log)
		}
	}

	commits := lines(cmdOutput("jj", "--config-toml", jjc, "log",
		"--no-graph", "-T", "commit_id ++ '\n'", "-r", "jjcrmailpending()"))
	if len(commits) > 10 && !*all {
		dief("more than 10 commits; use -a to mail all of them")
	}

	heads := lines(cmdOutput("jj", "--config-toml", jjc, "log",
		"--no-graph", "-T", "commit_id ++ '\n'", "-r", "heads(jjcrmailpending())"))
	if len(heads) > 1 && !*all {
		dief("multiple heads; use -a to mail all of them")
	}

	printf("mailing the following changes:\n\n%s",
		cmdOutput("jj", "--config-toml", jjc, "log", "-r", "jjcrmailpending()"))

	for _, commit := range heads {
		branches := lines(cmdOutput("jj", "--config-toml", jjc, "log",
			"--no-graph", "-T", "bookmarks", "-r", "jjcrbranch("+commit+")"))
		if len(branches) != 1 {
			dief("cannot determine branch for commit %s, got %v", commit, branches)
		}

		refSpec := commit + ":refs/for/" + strings.TrimSuffix(branches[0], "@origin")
		start := "%"
		if *rList != "" {
			refSpec += mailList(start, "r", string(*rList))
			start = ","
		}
		if *ccList != "" {
			refSpec += mailList(start, "cc", string(*ccList))
			start = ","
		}
		if *hashtagList != "" {
			for _, tag := range strings.Split(string(*hashtagList), ",") {
				if tag == "" {
					dief("hashtag may not contain empty tags")
				}
				refSpec += start + "hashtag=" + tag
				start = ","
			}
		}
		if *trybot {
			for _, v := range trybotVotes {
				refSpec += start + "l=" + v
				start = ","
			}
		}
		if *wip {
			refSpec += start + "wip"
			start = ","
		}
		if *autoSubmit {
			refSpec += start + "l=Auto-Submit"
		}
		args := []string{"push", "-q", "-o", "nokeycheck", "origin"}
		args = append(args, refSpec)
		run("git", args...)
	}

	// TODO(filippo): create refs/remote/gerrit/cl/NNNNNN/M refs.
}

// mailAddressRE matches the mail addresses we admit. It's restrictive but admits
// all the addresses in the Go CONTRIBUTORS file at time of writing (tested separately).
var mailAddressRE = regexp.MustCompile(`^([a-zA-Z0-9][-_.a-zA-Z0-9]*)(@[-_.a-zA-Z0-9]+)?$`)

// mailList turns the list of mail addresses from the flag value into the format
// expected by gerrit. The start argument is a % or , depending on where we
// are in the processing sequence.
func mailList(start, tag string, flagList string) string {
	errors := false
	spec := start
	short := ""
	long := ""
	for i, addr := range strings.Split(flagList, ",") {
		m := mailAddressRE.FindStringSubmatch(addr)
		if m == nil {
			printf("invalid reviewer mail address: %s", addr)
			errors = true
			continue
		}
		if m[2] == "" {
			email := mailLookup(addr)
			if email == "" {
				printf("unknown reviewer: %s", addr)
				errors = true
				continue
			}
			short += "," + addr
			long += "," + email
			addr = email
		}
		if i > 0 {
			spec += ","
		}
		spec += tag + "=" + addr
	}
	if short != "" {
		verbosef("expanded %s to %s", short[1:], long[1:])
	}
	if errors {
		exit(1)
	}
	return spec
}

// reviewers is the list of reviewers for the current repository,
// sorted by how many reviews each has done.
var reviewers []reviewer

type reviewer struct {
	addr  string
	count int
}

// mailLookup translates the short name (like adg) into a full
// email address (like adg@golang.org).
// It returns "" if no translation is found.
// The algorithm for expanding short user names is as follows:
// Look at the git commit log for the current repository,
// extracting all the email addresses in Reviewed-By lines
// and sorting by how many times each address appears.
// For each short user name, walk the list, most common
// address first, and use the first address found that has
// the short user name on the left side of the @.
func mailLookup(short string) string {
	loadReviewers()

	short += "@"
	for _, r := range reviewers {
		if strings.HasPrefix(r.addr, short) && !shortOptOut[r.addr] {
			return r.addr
		}
	}
	return ""
}

// shortOptOut lists email addresses whose owners have opted out
// from consideration for purposes of expanding short user names.
var shortOptOut = map[string]bool{
	"dmitshur@google.com": true, // My @golang.org is primary; @google.com is used for +1 only.
}

// loadReviewers reads the reviewer list from the current git repo
// and leaves it in the global variable reviewers.
// See the comment on mailLookup for a description of how the
// list is generated and used.
func loadReviewers() {
	if reviewers != nil {
		return
	}
	countByAddr := map[string]int{}
	for _, line := range nonBlankLines(cmdOutput("git", "log", "--format=format:%B", "-n", "1000")) {
		if strings.HasPrefix(line, "Reviewed-by:") {
			f := strings.Fields(line)
			addr := f[len(f)-1]
			if strings.HasPrefix(addr, "<") && strings.Contains(addr, "@") && strings.HasSuffix(addr, ">") {
				countByAddr[addr[1:len(addr)-1]]++
			}
		}
	}

	reviewers = []reviewer{}
	for addr, count := range countByAddr {
		reviewers = append(reviewers, reviewer{addr, count})
	}
	sort.Sort(reviewersByCount(reviewers))
}

type reviewersByCount []reviewer

func (x reviewersByCount) Len() int      { return len(x) }
func (x reviewersByCount) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x reviewersByCount) Less(i, j int) bool {
	if x[i].count != x[j].count {
		return x[i].count > x[j].count
	}
	return x[i].addr < x[j].addr
}

// stringList is a flag.Value that is like flag.String, but if repeated
// keeps appending to the old value, inserting commas as separators.
// This allows people to write -r rsc,adg (like the old hg command)
// but also -r rsc -r adg (like standard git commands).
// This does change the meaning of -r rsc -r adg (it used to mean just adg).
type stringList string

func (x *stringList) String() string {
	return string(*x)
}

func (x *stringList) Set(s string) error {
	if *x != "" && s != "" {
		*x += ","
	}
	*x += stringList(s)
	return nil
}
