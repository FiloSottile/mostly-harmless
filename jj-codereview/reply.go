package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func cmdReply(args []string) {
	file := flags.String("f", "", "read replies from `file` instead of standard input")
	flags.Usage = func() {
		fmt.Fprintf(stderr(), trim(`
Usage: %s reply %s [-f file] <query>

Stages draft replies on the CL matching the query, to be published
when the CL is next mailed (with publish_comments_on_push) or replied to.

The replies are read from standard input (or the -f file) as a JSON array:

	[
		{"id": "<comment-id>", "message": "Done", "resolved": true},
		{"message": "Thanks, PTAL!"}
	]

An entry with an id is a reply to that comment, attached to the same file,
line, and patchset. An entry without an id is a new patchset-level comment
on the current patchset. resolved defaults to true.
`), progName, globalFlags)
		exit(2)
	}
	flags.Parse(args)
	if flags.NArg() != 1 {
		flags.Usage()
	}

	c, err := GetChange(flags.Arg(0))
	if err != nil {
		dief("failed to fetch change: %v", err)
	}

	var src io.Reader = os.Stdin
	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			dief("%v", err)
		}
		defer f.Close()
		src = f
	}
	type replyInput struct {
		ID       string `json:"id"`
		Message  string `json:"message"`
		Resolved *bool  `json:"resolved"`
	}
	var replies []replyInput
	dec := json.NewDecoder(src)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&replies); err != nil {
		dief("failed to decode replies: %v", err)
	}

	comments, err := GetComments(c.Number)
	if err != nil {
		dief("failed to fetch comments: %v", err)
	}

	for _, r := range replies {
		if r.Message == "" {
			dief("reply with empty message")
		}
		draft := &CommentInput{
			Message:    r.Message,
			Unresolved: r.Resolved != nil && !*r.Resolved,
		}
		patchSet := c.Revisions[c.CurrentRevision].Number
		target := "patchset level"
		if r.ID != "" {
			parent, ok := comments[r.ID]
			if !ok {
				dief("comment %s not found on CL %d", r.ID, c.Number)
			}
			draft.Path = parent.Path
			draft.Line = parent.Line
			draft.Range = parent.Range
			draft.Side = parent.Side
			draft.InReplyTo = parent.ID
			patchSet = parent.PatchSet
			target = fmt.Sprintf("%s (%s)", parent.Path, parent.Author.Name)
		} else {
			draft.Path = patchSetLevel
		}
		if *noRun {
			printf("would stage draft on %s: %q", target, r.Message)
			continue
		}
		url := fmt.Sprintf("https://go-review.googlesource.com/a/changes/%d/revisions/%d/drafts", c.Number, patchSet)
		if _, err := authedRequest("PUT", url, draft); err != nil {
			dief("failed to stage draft on %s: %v", target, err)
		}
		verbosef("staged draft on %s: %q", target, r.Message)
	}
	if !*noRun {
		printf("staged %d draft(s) on CL %d, to be published at the next mail", len(replies), c.Number)
	}
}

const patchSetLevel = "/PATCHSET_LEVEL"

type CommentInput struct {
	Path       string           `json:"path"`
	Line       int              `json:"line,omitempty"`
	Range      *json.RawMessage `json:"range,omitempty"`
	Side       string           `json:"side,omitempty"`
	InReplyTo  string           `json:"in_reply_to,omitempty"`
	Message    string           `json:"message"`
	Unresolved bool             `json:"unresolved"`
}

type GerritComment struct {
	ID       string           `json:"id"`
	Path     string           `json:"-"`
	PatchSet int              `json:"patch_set"`
	Line     int              `json:"line"`
	Range    *json.RawMessage `json:"range"`
	Side     string           `json:"side"`
	Author   struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
	Message    string `json:"message"`
	Unresolved bool   `json:"unresolved"`
}

func GetComments(number int) (map[string]*GerritComment, error) {
	url := fmt.Sprintf("https://go-review.googlesource.com/changes/%d/comments", number)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	// Gerrit prepends a magic string to prevent XSSI.
	body, _ = bytes.CutPrefix(body, []byte(")]}'\n"))
	var byPath map[string][]*GerritComment
	if err := json.Unmarshal(body, &byPath); err != nil {
		return nil, fmt.Errorf("failed to decode comments: %v", err)
	}
	comments := map[string]*GerritComment{}
	for path, cc := range byPath {
		for _, c := range cc {
			c.Path = path
			comments[c.ID] = c
		}
	}
	return comments, nil
}

var gerritCredentials struct {
	username, password string
	approved           bool
}

// fillGerritCredentials asks git-credential for a go.googlesource.com token,
// which is also valid for go-review.googlesource.com. It may go through an
// interactive git-credential-oauth browser flow if no token is cached.
func fillGerritCredentials() {
	if gerritCredentials.password != "" {
		return
	}
	out, err := cmdOutputCredential("fill")
	if err != nil {
		dief("git credential fill failed: %v", err)
	}
	for _, line := range lines(out) {
		if v, ok := strings.CutPrefix(line, "username="); ok {
			gerritCredentials.username = v
		}
		if v, ok := strings.CutPrefix(line, "password="); ok {
			gerritCredentials.password = v
		}
	}
	if gerritCredentials.password == "" {
		dief("git credential fill returned no password")
	}
}

func cmdOutputCredential(action string) (string, error) {
	cmd := newCredentialCommand(action)
	b, err := cmd.Output()
	return string(b), err
}

func authedRequest(method, url string, body any) ([]byte, error) {
	fillGerritCredentials()
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(gerritCredentials.username, gerritCredentials.password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		newCredentialCommand("reject").Run()
		return nil, fmt.Errorf("%s %s: %s", method, url, resp.Status)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%s %s: %s\n%s", method, url, resp.Status, respBody)
	}
	if !gerritCredentials.approved {
		gerritCredentials.approved = true
		newCredentialCommand("approve").Run()
	}
	// Gerrit prepends a magic string to prevent XSSI.
	respBody, _ = bytes.CutPrefix(respBody, []byte(")]}'\n"))
	return respBody, nil
}

func newCredentialCommand(action string) *exec.Cmd {
	cmd := exec.Command("git", "credential", action)
	attrs := "protocol=https\nhost=go.googlesource.com\n"
	if action != "fill" {
		attrs += "username=" + gerritCredentials.username + "\n"
		attrs += "password=" + gerritCredentials.password + "\n"
	}
	cmd.Stdin = strings.NewReader(attrs + "\n")
	cmd.Stderr = stderr()
	setEnglishLocale(cmd)
	return cmd
}
