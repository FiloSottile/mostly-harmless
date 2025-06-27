package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
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

	mux.Handle("buttondown.filippo.io/{$}",
		http.RedirectHandler("https://buttondown.com/cryptography-dispatches/", http.StatusFound))

	mux.Handle("buttondown.filippo.io/unsubscribe/", HostRedirectHandler("buttondown.com",
		http.StatusTemporaryRedirect)) // 307 to preserve POST from List-Unsubscribe-Post.

	mux.Handle("buttondown.filippo.io/static/", HostRedirectHandler("buttondown.com", http.StatusFound))

	mux.Handle("buttondown.filippo.io/archive/{slug}/{$}", SlugRedirectHandler())
	mux.Handle("buttondown.filippo.io/dispatches/{slug}/{$}", SlugRedirectHandler())
	mux.Handle("buttondown.filippo.io/{slug}/{rest...}", EmailHandler())
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

var buttondownClient = &http.Client{
	Timeout: 30 * time.Second,
}

func fetchMails() error {
	url := "https://api.buttondown.com/v1/emails?-email_type=private&-status=draft&-status=scheduled&-status=deleted"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+os.Getenv("BUTTONDOWN_API_KEY"))
	req.Header.Set("X-API-Version", "2025-06-01")
	resp, err := buttondownClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected status code: " + resp.Status)
	}
	var response struct {
		Next    *string           `json:"next"`
		Results []buttondownEmail `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}
	var allEmails []buttondownEmail
	allEmails = append(allEmails, response.Results...)
	for response.Next != nil {
		req, err = http.NewRequest("GET", *response.Next, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Token "+os.Getenv("BUTTONDOWN_API_KEY"))
		req.Header.Set("X-API-Version", "2025-06-01")
		resp, err = buttondownClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return errors.New("unexpected status code: " + resp.Status)
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return err
		}
		allEmails = append(allEmails, response.Results...)
	}
	if len(allEmails) < 90 {
		return errors.New("fetched less than 90 emails, something went wrong")
	}
	emailsMu.Lock()
	defer emailsMu.Unlock()
	emails = make(map[string]*buttondownEmail, len(allEmails)*2)
	for _, email := range allEmails {
		for _, slug := range []string{email.Slug, email.ID, email.Metadata.OverrideSlug} {
			if slug == "" {
				continue
			}
			if old, exists := emails[slug]; exists {
				log.Printf("warning: slug %q of email %q already exists as %q", slug, email.ID, old.ID)
			}
			emails[slug] = &email
		}
	}
	return nil
}
