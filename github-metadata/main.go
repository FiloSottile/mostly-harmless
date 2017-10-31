// Command github-metadata downloads everything available about
// a repository, except the git data itself and the wiki.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/peterhellberg/link"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: github-metadata owner repo")
	}
	owner, repo := os.Args[1], os.Args[2]

	hc := &http.Client{
		Timeout: 10 * time.Second,
	}
	download := func(filename, path string, obj interface{}) {
		os.MkdirAll(filepath.Dir(filename), 0775)
		path = strings.Replace(path, ":owner", owner, -1)
		path = strings.Replace(path, ":repo", repo, -1)
		path = strings.TrimPrefix(path, "https://api.github.com")
		log.Printf("%s => %s", path, filename)
		path = "https://api.github.com" + path
		req, err := http.NewRequest("GET", path, nil)
		fatalIfErr(err)
		req.Header.Set("Accept", "application/vnd.github.full+json, application/vnd.github.mercy-preview+json, application/vnd.github.squirrel-girl-preview, application/vnd.github.thor-preview+json, application/vnd.github.machine-man-preview, application/vnd.github.mockingbird-preview")
		req.Header.Set("User-Agent", "github.com/FiloSottile/mostly-harmless/github-metadata")
		if os.Getenv("USER") != "" {
			req.SetBasicAuth(os.Getenv("USER"), os.Getenv("TOKEN"))
		}
		resp, err := hc.Do(req)
		fatalIfErr(err)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fatalIfErr(errors.New("remote error: " + resp.Status))
		}
		if _, ok := link.ParseResponse(resp)["next"]; ok {
			// https://developer.github.com/v3/guides/traversing-with-pagination/
			fatalIfErr(errors.New("pagination required"))
		}
		file, err := os.Create(filename)
		fatalIfErr(err)
		if obj != nil {
			buf := &bytes.Buffer{}
			err = json.NewDecoder(io.TeeReader(resp.Body, buf)).Decode(obj)
			fatalIfErr(err)
			_, err = io.Copy(file, buf)
			fatalIfErr(err)
		} else {
			_, err = io.Copy(file, resp.Body)
		}
		fatalIfErr(err)
		fatalIfErr(file.Close())
	}

	var pulls []struct {
		Number int
		Links  struct {
			Self           struct{ Href string }
			Comments       struct{ Href string }
			Commits        struct{ Href string }
			Issue          struct{ Href string }
			ReviewComments struct{ Href string } `json:"review_comments"`
		} `json:"_links"`
	}
	download("pulls.json", "/repos/:owner/:repo/pulls?state=all", &pulls)
	for _, pull := range pulls {
		n := func(s string) string {
			return strings.Replace(s, ":number", strconv.Itoa(pull.Number), -1)
		}
		download(n("pulls/:number/pull.json"), pull.Links.Self.Href, nil)
		// download(n("pulls/:number/issue.json"), pull.Links.Issue.Href, nil)
		// download(n("pulls/:number/comments.json"), pull.Links.Comments.Href, nil)
		download(n("pulls/:number/review_comments.json"), pull.Links.ReviewComments.Href, nil)
		download(n("pulls/:number/commits.json"), pull.Links.Commits.Href, nil)
		download(n("pulls/:number/reviews.json"), n("/repos/:owner/:repo/pulls/:number/reviews"), nil)
		download(n("pulls/:number/requested_reviewers.json"), n("/repos/:owner/:repo/pulls/:number/requested_reviewers"), nil)
		// download(n("pulls/:number/timeline.json"), n("/repos/:owner/:repo/issues/:number/timeline"), nil)
		// download(n("pulls/:number/events.json"), n("/repos/:owner/:repo/issues/:number/events"), nil)
		// download(n("pulls/:number/reactions.json"), n("/repos/:owner/:repo/issues/:number/reactions"), nil)
	}

	var issues []struct {
		Number int
	}
	download("issues.json", "/repos/:owner/:repo/issues?state=all", &issues)
	for _, issue := range issues {
		n := func(s string) string {
			return strings.Replace(s, ":number", strconv.Itoa(issue.Number), -1)
		}
		download(n("issues/:number/issue.json"), n("/repos/:owner/:repo/issues/:number"), nil)
		download(n("issues/:number/comments.json"), n("/repos/:owner/:repo/issues/:number/comments"), nil)
		download(n("issues/:number/timeline.json"), n("/repos/:owner/:repo/issues/:number/timeline"), nil)
		download(n("issues/:number/events.json"), n("/repos/:owner/:repo/issues/:number/events"), nil)
		download(n("issues/:number/reactions.json"), n("/repos/:owner/:repo/issues/:number/reactions"), nil)
	}

	// TODO: graphs and traffic
	download("stargazers.json", "/repos/:owner/:repo/stargazers", nil)
	download("subscribers.json", "/repos/:owner/:repo/subscribers", nil)
	// TODO: https://developer.github.com/v3/projects/
	download("branches.json", "/repos/:owner/:repo/branches", nil)
	// download("collaborators.json", "/repos/:owner/:repo/collaborators", nil)
	download("comments.json", "/repos/:owner/:repo/comments", nil)
	download("forks.json", "/repos/:owner/:repo/forks", nil)
	download("releases.json", "/repos/:owner/:repo/releases", nil)
	download("languages.json", "/repos/:owner/:repo/languages", nil)
	download("tags.json", "/repos/:owner/:repo/tags", nil)
	download("contributors.json", "/repos/:owner/:repo/contributors", nil)
	download("topics.json", "/repos/:owner/:repo/topics", nil)
	download("labels.json", "/repos/:owner/:repo/labels", nil)
	download("milestones.json", "/repos/:owner/:repo/milestones?state=all", nil)
	download("repo.json", "/repos/:owner/:repo", nil)
}

func fatalIfErr(err error) {
	if err != nil {
		debug.PrintStack()
		log.Fatal(err)
	}
}
