package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/feeds"
	"golang.org/x/net/html"
)

type buttondownEmail struct {
	Body        template.HTML `json:"body"`
	Description string        `json:"description"`
	ID          string        `json:"id"`
	PublishDate string        `json:"publish_date"`
	Slug        string        `json:"slug"`
	Subject     string        `json:"subject"`
	Status      string        `json:"status"`
	Metadata    struct {
		OverrideSlug  string        `json:"override_slug"`
		OverrideGUID  string        `json:"override_guid"`
		OverrideImage string        `json:"override_image"`
		FooterCode    template.HTML `json:"footer_code"`
	} `json:"metadata"`
	Image string `json:"-"`
}

func canonicalSlug(e *buttondownEmail) string {
	if e.Metadata.OverrideSlug != "" {
		return e.Metadata.OverrideSlug
	}
	if e.Slug != "" {
		return e.Slug
	}
	return e.ID
}

var emailsMu sync.RWMutex
var emailsBySlug map[string]*buttondownEmail
var emails []*buttondownEmail

func emailBySlug(slug string) *buttondownEmail {
	emailsMu.RLock()
	defer emailsMu.RUnlock()
	return emailsBySlug[slug]
}

func buttondown(mux *http.ServeMux) {
	if err := fetchMails(); err != nil {
		log.Fatalf("failed to fetch emails at startup: %v", err.Error())
	}
	go func() {
		for range time.NewTicker(1 * time.Minute).C {
			if err := fetchMails(); err != nil {
				log.Printf("failed to fetch emails: %v", err)
			}
		}
	}()

	redirectToIndex := http.RedirectHandler("/", http.StatusFound)
	redirectToFeed := http.RedirectHandler("/rss/", http.StatusFound)
	redirectToHome := http.RedirectHandler("https://filippo.io/", http.StatusFound)
	redirectToButtondown := HostRedirectHandler("buttondown.com", http.StatusFound)
	// 307 to preserve POST from List-Unsubscribe-Post.
	redirectToButtondown307 := HostRedirectHandler("buttondown.com", http.StatusTemporaryRedirect)
	redirectToButtondownWithPrefix := HostPrefixRedirectHandler(
		"buttondown.com", "/cryptography-dispatches", http.StatusFound)

	mux.Handle("words.filippo.io/{$}", IndexHandler())
	mux.Handle("words.filippo.io/dispatches/{$}", redirectToIndex)
	mux.Handle("words.filippo.io/archive/{$}", redirectToIndex)
	mux.Handle("words.filippo.io/hi/{$}", redirectToHome)

	mux.Handle("words.filippo.io/unsubscribe/", redirectToButtondown307)
	mux.Handle("words.filippo.io/subscribers/", redirectToButtondownWithPrefix)
	mux.Handle("words.filippo.io/management/", redirectToButtondownWithPrefix)
	mux.Handle("words.filippo.io/static/", redirectToButtondown)

	mux.Handle("words.filippo.io/rss/{$}", FeedHandler())
	mux.Handle("words.filippo.io/feed/{$}", redirectToFeed)
	mux.Handle("words.filippo.io/dispatches/rss/{$}", redirectToFeed)
	mux.Handle("words.filippo.io/dispatches/feed/{$}", redirectToFeed)

	mux.Handle("words.filippo.io/archive/{slug}/{$}", SlugRedirectHandler())
	mux.Handle("words.filippo.io/dispatches/{slug}/{$}", SlugRedirectHandler())
	mux.Handle("words.filippo.io/{slug}/{rest...}", EmailHandler())
}

func HostPrefixRedirectHandler(target, prefix string, code int) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		u := &url.URL{
			Scheme:   "https",
			Host:     target,
			Path:     prefix + r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		http.Redirect(rw, r, u.String(), code)
	})
}

func SlugRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		email := emailBySlug(slug)
		if email == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		target := "/" + canonicalSlug(email) + "/"
		http.Redirect(w, r, target, http.StatusFound)
	})
}

//go:embed buttondown_email.html.tmpl
var emailTemplate string

//go:embed buttondown_index.html.tmpl
var indexTemplate string

var funcs = template.FuncMap{
	"dateFormat": func(t string, layout string) (string, error) {
		tm, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return "", err
		}
		return tm.Format(layout), nil
	},
	"canonicalSlug": canonicalSlug,
	"embedBskyPost": embedBskyPost,
}

var emailTmpl = template.Must(template.New("email").Funcs(funcs).Parse(emailTemplate))
var indexTmpl = template.Must(template.New("index").Funcs(funcs).Parse(indexTemplate))

func EmailHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		email := emailBySlug(slug)
		if email == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		rest := r.PathValue("rest")
		if rest == "+" {
			target := "https://buttondown.com/emails/" + email.ID
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
		if rest != "" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if canonical := canonicalSlug(email); canonical != slug {
			target := "/" + canonical + "/"
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		emailTmpl.Execute(w, email)
	})
}

func IndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		indexTmpl.Execute(w, func() []*buttondownEmail {
			emailsMu.RLock()
			defer emailsMu.RUnlock()
			return emails
		}())
	})
}

func FeedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := &feeds.Feed{
			Title: "Filippo Valsorda",
			Link:  &feeds.Link{Href: "https://words.filippo.io/"},
			Items: feedItems(),
		}
		rss, err := f.ToRss()
		if err != nil {
			log.Printf("failed to generate RSS feed: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		w.Write([]byte(rss))
	})
}

func feedItems() []*feeds.Item {
	emailsMu.RLock()
	defer emailsMu.RUnlock()
	var items []*feeds.Item
	for _, email := range emails {
		created, _ := time.Parse(time.RFC3339, email.PublishDate)
		if created.Year() < 2024 {
			// We set the override GUID only for 2024+ emails.
			continue
		}
		if time.Since(created) < 1*time.Hour {
			// Delay emails by an hour in case I get something wrong and need to
			// fix it or roll back before it hits the clients forever.
			continue
		}
		item := &feeds.Item{
			Id:          email.ID,
			IsPermaLink: "false",
			Title:       email.Subject,
			Created:     created,
			Link: &feeds.Link{
				Href: "https://words.filippo.io/" + canonicalSlug(email) + "/",
			},
			Author: &feeds.Author{
				Name: "Filippo Valsorda <feed@filippo.io>",
			},
			Content: string(email.Body),
		}
		if email.Metadata.OverrideGUID != "" {
			item.Id = email.Metadata.OverrideGUID
		}
		if email.Description != "" {
			item.Description = email.Description
		}
		items = append(items, item)
		if len(items) == 10 {
			break
		}
	}
	return items
}

var buttondownClient = &http.Client{
	Timeout: 30 * time.Second,
}

type buttondownResponse struct {
	Next    *string            `json:"next"`
	Results []*buttondownEmail `json:"results"`
}

func buttondownGET(url string) (*buttondownResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+os.Getenv("BUTTONDOWN_API_KEY"))
	req.Header.Set("X-API-Version", "2025-06-01")
	resp, err := buttondownClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status code: " + resp.Status)
	}
	var response buttondownResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

var markdownCacheMu sync.Mutex
var markdownCache = make(map[string]template.HTML)

var ifRegexp = regexp.MustCompile(`(?s)\{%\s*if(.*?)\{%\s*endif\s*%\}`)

func buttondownMarkdown(text string) (template.HTML, error) {
	markdownCacheMu.Lock()
	defer markdownCacheMu.Unlock()
	if html, ok := markdownCache[text]; ok {
		return html, nil
	}

	// https://docs.buttondown.com/using-markdown-rendering
	cmd := exec.Command("markdown_py", "-x", "smarty", "-x", "tables", "-x", "footnotes",
		"-x", "fenced_code", "-x", "pymdownx.tilde", "-x", "toc", "-x", "mdx_truly_sane_lists")
	// Very crude removal of conditional email-only tags.
	cmd.Stdin = strings.NewReader(ifRegexp.ReplaceAllString(text, ""))
	out, err := cmd.Output()
	if err != nil {
		return "", errors.New("failed to render markdown: " + err.Error())
	}

	markdownCache[text] = template.HTML(out)
	return template.HTML(out), nil
}

func fetchMails() error {
	var allEmails []*buttondownEmail
	r, err := buttondownGET("https://api.buttondown.com/v1/emails")
	if err != nil {
		return err
	}
	allEmails = append(allEmails, r.Results...)
	for range 20 {
		if r.Next == nil {
			break
		}
		r, err = buttondownGET(*r.Next)
		if err != nil {
			return err
		}
		allEmails = append(allEmails, r.Results...)
	}
	if len(allEmails) < 90 {
		return errors.New("fetched less than 90 emails, something went wrong")
	}

	allEmails = slices.DeleteFunc(allEmails, func(e *buttondownEmail) bool {
		switch e.Status {
		case "draft", "scheduled", "managed_by_rss", "deleted", "paused", "errored", "transactional":
			return true
		case "about_to_send", "in_flight", "imported", "sent", "throttled", "resending":
			return false
		default:
			log.Printf("unknown status %q for email %q, hiding it", e.Status, e.ID)
			return true
		}
	})

	for _, email := range allEmails {
		email.Subject = strings.TrimPrefix(email.Subject, "Cryptography Dispatches: ")
		email.Subject = strings.TrimPrefix(email.Subject, "Maintainer Dispatches: ")

		email.Body, err = buttondownMarkdown(string(email.Body))
		if err != nil {
			return err
		}

		email.Image = extractLastImage(email.Body)
		if email.Metadata.OverrideImage != "" {
			email.Image = email.Metadata.OverrideImage
		}

		if _, err := time.Parse(time.RFC3339, email.PublishDate); err != nil {
			log.Printf("failed to parse publish date %q of email %q: %v", email.PublishDate, email.ID, err)
			email.PublishDate = time.Now().Format(time.RFC3339)
		}
	}

	sort.Slice(allEmails, func(i, j int) bool {
		return allEmails[i].PublishDate > allEmails[j].PublishDate
	})

	emailsMu.Lock()
	defer emailsMu.Unlock()
	emails = allEmails
	emailsBySlug = make(map[string]*buttondownEmail, len(allEmails)*2)
	for _, email := range allEmails {
		for _, slug := range []string{email.Slug, email.ID, email.Metadata.OverrideSlug} {
			if slug == "" {
				continue
			}
			if old, exists := emailsBySlug[slug]; exists {
				log.Printf("warning: slug %q of email %q already exists as %q", slug, email.ID, old.ID)
			}
			emailsBySlug[slug] = email
		}
	}
	return nil
}

func extractLastImage(htmlContent template.HTML) (url string) {
	doc, err := html.Parse(strings.NewReader(string(htmlContent)))
	if err != nil {
		return ""
	}
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					url = attr.Val
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)
	return
}

var bskyClient = &http.Client{
	Timeout: 5 * time.Second,
}

var embedBskyCacheMu sync.RWMutex
var embedBskyCache = make(map[string]template.HTML)
var embedBskyLastChecked = make(map[string]time.Time)

func embedBskyPost(u string) template.HTML {
	embedBskyCacheMu.RLock()
	html := embedBskyCache[u]
	lastChecked := embedBskyLastChecked[u]
	embedBskyCacheMu.RUnlock()
	if html != "" || time.Since(lastChecked) < 10*time.Minute {
		return html
	}

	atURL, err := findBskyPost(u)
	if err != nil {
		log.Printf("failed to find bsky post for %q: %v", u, err)
		return ""
	}
	if atURL == "" {
		atURL, err = findBskyPost(strings.TrimSuffix(u, "/"))
		if err != nil {
			log.Printf("failed to find bsky post for %q: %v", u, err)
			return ""
		}
	}
	if atURL == "" {
		embedBskyCacheMu.Lock()
		embedBskyCache[u] = ""
		embedBskyLastChecked[u] = time.Now()
		embedBskyCacheMu.Unlock()
		return ""
	}

	oembedURL := "https://embed.bsky.app/oembed?maxwidth=400&url=" + url.QueryEscape(atURL)
	req, err := http.NewRequest("GET", oembedURL, nil)
	if err != nil {
		log.Printf("failed to create oembed request for %q: %v", atURL, err)
		return ""
	}
	resp, err := bskyClient.Do(req)
	if err != nil {
		log.Printf("failed to fetch oembed for %q: %v", atURL, err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code fetching oembed for %q: %s", atURL, resp.Status)
		return ""
	}
	var result struct {
		HTML template.HTML `json:"html"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("failed to decode oembed response for %q: %v", atURL, err)
		return ""
	}

	embedBskyCacheMu.Lock()
	embedBskyCache[u] = result.HTML
	embedBskyCacheMu.Unlock()
	return result.HTML
}

func findBskyPost(u string) (string, error) {
	query := &url.Values{}
	query.Add("subject", u)
	query.Add("source", "app.bsky.feed.post:embed.external.uri")
	query.Add("did", "did:plc:x2nsupeeo52oznrmplwapppl")
	query.Add("limit", "10")
	constellationURL := "https://constellation.microcosm.blue/xrpc/blue.microcosm.links.getBacklinks?" + query.Encode()
	req, err := http.NewRequest("GET", constellationURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := bskyClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %s", resp.Status)
	}
	var result struct {
		Records []struct {
			DID  string `json:"did"`
			Rkey string `json:"rkey"`
		} `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Records) == 0 {
		return "", nil
	}
	record := result.Records[len(result.Records)-1] // oldest
	return "at://" + record.DID + "/app.bsky.feed.post/" + record.Rkey, nil
}
