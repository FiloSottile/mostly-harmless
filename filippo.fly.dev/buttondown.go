package main

import (
	"bytes"
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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/feeds"
)

type buttondownEmail struct {
	Body        template.HTML `json:"body"`
	Description string        `json:"description"`
	ID          string        `json:"id"`
	PublishDate string        `json:"publish_date"`
	Slug        string        `json:"slug"`
	Subject     string        `json:"subject"`
	Metadata    struct {
		OverrideSlug string `json:"override_slug"`
		OverrideGUID string `json:"override_guid"`
	} `json:"metadata"`
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
var emails map[string]*buttondownEmail
var feed []*buttondownEmail

func emailBySlug(slug string) *buttondownEmail {
	emailsMu.RLock()
	defer emailsMu.RUnlock()
	return emails[slug]
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

	redirectToNewsletter := http.RedirectHandler(
		"https://buttondown.com/cryptography-dispatches/", http.StatusFound)
	redirectToWords := http.RedirectHandler("https://words.filippo.io/", http.StatusFound)
	redirectToButtondown := HostRedirectHandler("buttondown.com", http.StatusFound)
	// 307 to preserve POST from List-Unsubscribe-Post.
	redirectToButtondown307 := HostRedirectHandler("buttondown.com", http.StatusTemporaryRedirect)
	redirectToButtondownWithPrefix := HostPrefixRedirectHandler(
		"buttondown.com", "/cryptography-dispatches", http.StatusFound)

	mux.Handle("buttondown.filippo.io/{$}", redirectToNewsletter)
	mux.Handle("buttondown.filippo.io/dispatches/{$}", redirectToWords)
	mux.Handle("buttondown.filippo.io/archive/{$}", redirectToWords)

	mux.Handle("buttondown.filippo.io/unsubscribe/", redirectToButtondown307)
	mux.Handle("buttondown.filippo.io/subscribers/", redirectToButtondownWithPrefix)
	mux.Handle("buttondown.filippo.io/management/", redirectToButtondownWithPrefix)
	mux.Handle("buttondown.filippo.io/static/", redirectToButtondown)

	mux.Handle("buttondown.filippo.io/rss/{$}", FeedHandler())
	mux.Handle("buttondown.filippo.io/dispatches/rss/{$}", FeedHandler())

	mux.Handle("buttondown.filippo.io/archive/{slug}/{$}", SlugRedirectHandler())
	mux.Handle("buttondown.filippo.io/dispatches/{slug}/{$}", SlugRedirectHandler())
	mux.Handle("buttondown.filippo.io/{slug}/{rest...}", EmailHandler())
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
		target := "https://buttondown.filippo.io/" + canonicalSlug(email) + "/"
		http.Redirect(w, r, target, http.StatusFound)
	})
}

//go:embed buttondown_email.html.tmpl
var buttondownEmailTemplate string

var buttondownEmailTmpl = template.Must(template.New("buttondown_email").Funcs(template.FuncMap{
	"dateFormat": func(t string, layout string) (string, error) {
		tm, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return "", err
		}
		return tm.Format(layout), nil
	},
	"canonicalSlug": func(e *buttondownEmail) string {
		return canonicalSlug(e)
	},
}).Parse(buttondownEmailTemplate))

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
		if canonical := canonicalSlug(email); canonical != slug && rest == "" {
			target := "https://buttondown.filippo.io/" + canonical + "/"
			http.Redirect(w, r, target, http.StatusFound)
			return
		}

		// For now, hide the new version behind a !, and redirect to Ghost.
		if rest == "!" {
			if err := buttondownEmailTmpl.Execute(w, email); err != nil {
				log.Printf("failed to execute buttondown email template: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}
		if rest == "" {
			target := "https://words.filippo.io/dispatches/" + slug + "/"
			http.Redirect(w, r, target, http.StatusFound)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
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
	for _, email := range feed {
		created, _ := time.Parse(time.RFC3339, email.PublishDate)
		if created.Year() < 2024 {
			// We set the override GUID only for 2024+ emails.
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
				Name: "Filippo Valsorda",
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

func fetchMails() error {
	var allEmails []*buttondownEmail
	r, err := buttondownGET("https://api.buttondown.com/v1/emails?-email_type=private&-status=draft&-status=scheduled&-status=deleted")
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

	for _, email := range allEmails {
		email.Subject = strings.TrimPrefix(email.Subject, "Cryptography Dispatches: ")
		email.Subject = strings.TrimPrefix(email.Subject, "Maintainer Dispatches: ")

		// https://docs.buttondown.com/using-markdown-rendering
		// markdown_py -x smarty -x tables -x footnotes -x fenced_code -x pymdownx.tilde -x toc
		cmd := exec.Command("markdown_py", "-x", "smarty", "-x", "tables", "-x", "footnotes",
			"-x", "fenced_code", "-x", "pymdownx.tilde", "-x", "toc")
		cmd.Stdin = bytes.NewReader([]byte(email.Body))
		out, err := cmd.Output()
		if err != nil {
			return errors.New("failed to render markdown: " + err.Error())
		}
		email.Body = template.HTML(out)

		if _, err := time.Parse(time.RFC3339, email.PublishDate); err != nil {
			return fmt.Errorf("failed to parse publish date %q of email %q: %w", email.PublishDate, email.ID, err)
		}
	}

	sort.Slice(allEmails, func(i, j int) bool {
		return allEmails[i].PublishDate > allEmails[j].PublishDate
	})

	emailsMu.Lock()
	defer emailsMu.Unlock()
	feed = allEmails[:10]
	emails = make(map[string]*buttondownEmail, len(allEmails)*2)
	for _, email := range allEmails {
		for _, slug := range []string{email.Slug, email.ID, email.Metadata.OverrideSlug} {
			if slug == "" {
				continue
			}
			if old, exists := emails[slug]; exists {
				log.Printf("warning: slug %q of email %q already exists as %q", slug, email.ID, old.ID)
			}
			emails[slug] = email
		}
	}
	return nil
}
