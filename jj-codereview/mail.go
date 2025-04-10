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
	"time"
)

func cmdMail(args []string) {
	var (
		multi = flags.Bool("m", false, "mail multiple heads")
		force = flags.Bool("f", false, "force mailing more than 10 commits")

		rList       = new(stringList) // installed below
		ccList      = new(stringList) // installed below
		hashtagList = new(stringList) // installed below

		trybot     = flags.Bool("trybot", false, "run trybots on the uploaded CLs")
		wip        = flags.Bool("wip", false, "set the status of a change to Work-in-Progress")
		autoSubmit = flags.Bool("autosubmit", false, "set autosubmit on the uploaded CLs")
	)
	flags.Var(rList, "r", "comma-separated list of reviewers")
	flags.Var(ccList, "cc", "comma-separated list of people to CC:")
	flags.Var(hashtagList, "hashtag", "comma-separated list of tags to set")

	flags.Usage = func() {
		fmt.Fprintf(stderr(), trim(`
Usage: %s mail %s [-r reviewer,...] [-cc mail,...]
	[-autosubmit] [-trybot] [-wip] [-hashtag tag,...]
	[revisions]

Mails all changes in "remote_bookmarks(remote=origin)..revisions".

If revisions is not specified, it's set to "@-".
`), progName, globalFlags)
		exit(2)
	}
	flags.Parse(args)
	if len(flags.Args()) > 1 {
		flags.Usage()
		exit(2)
	}

	revConfig := `--config=revset-aliases."jjcrmail()"=@-`
	if len(flags.Args()) == 1 {
		revConfig = `--config=revset-aliases."jjcrmail()"=` + flags.Arg(0)
	}

	config := jjConfig()
	defer os.Remove(config)
	jjLog := jjLog(config, revConfig)

	// A good definition for private() is:
	//
	//     conflicts() | (empty() ~ merges()) | description('substring-i:"DO NOT MAIL"')
	//
	if private := jjLog("-r", "jjcrmailpending() & private()"); len(private) > 0 {
		dief("the following changes are private:\n\n%s", strings.Join(private, "\n"))
	}

	commits := jjLog("-T", "commit_id ++ '\n'", "-r", "jjcrmailpending()")
	if len(commits) > 10 && !*force {
		dief("more than 10 commits; use -f to force")
	}

	heads := jjLog("-T", "commit_id ++ '\n'", "-r", "heads(jjcrmailpending())")
	if len(heads) > 1 && !*multi {
		dief("multiple heads; use -m to mail all of them")
	}

	printf("mailing the following changes:\n\n%s",
		cmdOutput("jj", "--quiet", revConfig, "--config-file", config, "log", "-r", "jjcrmailpending()"))

	for _, commit := range heads {
		branches := jjLog("-T", "bookmarks", "-r", "jjcrbranchhead("+commit+")")
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
			refSpec += start + "l=Commit-Queue+1"
			start = ","
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

	for _, commit := range commits {
		labelCommit(commit, 0)
	}
}

func labelCommit(commit string, tries int) {
	change, err := GetChange(commit)
	if err != nil {
		if tries < 5 {
			// Retry a few times in case the change is not yet visible.
			time.Sleep(1 * time.Second)
			fmt.Printf("%s\r", strings.Repeat(".", tries))
			labelCommit(commit, tries+1)
			return
		}
		printf("failed to fetch change for commit %s: %v", commit, err)
		return
	}
	patchSet := "?"
	if !*noRun {
		patchSet = fmt.Sprintf("%d", change.Revisions[commit].Number)
	}
	ref := fmt.Sprintf("refs/remotes/gerrit/cl/%d/%s", change.Number, patchSet)
	run("git", "update-ref", ref, commit)
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
