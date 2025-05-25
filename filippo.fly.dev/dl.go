package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v38/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/oauth2"
)

var gitHubClient *github.Client

func getLatestVersion(ctx context.Context, project string) (string, error) {
	rel, _, err := gitHubClient.Repositories.GetLatestRelease(ctx, "FiloSottile", project)
	if err != nil {
		return "", err
	}
	return *rel.TagName, nil
}

func dl(mux *http.ServeMux) {
	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	))
	tc.Timeout = 10 * time.Second
	gitHubClient = github.NewClient(tc)

	// Print an error at startup if we can't fetch the latest versions.
	if _, err := getLatestVersion(context.Background(), "age"); err != nil {
		log.Println(err)
	}
	if _, err := getLatestVersion(context.Background(), "mkcert"); err != nil {
		log.Println(err)
	}

	mux.HandleFunc("dl.filippo.io/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "User-agent: *")
		fmt.Fprintln(w, "Disallow: /")
	})

	mux.HandleFunc("dl.filippo.io/{project}/{version}", func(w http.ResponseWriter, r *http.Request) {
		project, version := r.PathValue("project"), r.PathValue("version")

		switch project {
		case "age", "mkcert":
		default:
			http.Error(w, "Unknown project", http.StatusNotFound)
			return
		}

		if version != "latest" && !strings.HasPrefix(version, "v") {
			http.Error(w, "Invalid download path", http.StatusNotFound)
			return
		}

		parts := strings.Split(r.URL.Query().Get("for"), "/")
		if len(parts) != 2 {
			http.Error(w, "Invalid or missing 'for' value", http.StatusBadRequest)
			return
		}
		GOOS, GOARCH := parts[0], parts[1]

		proof := ""
		if r.URL.Query().Has("proof") {
			proof = ".proof"
		}

		dlReqs.WithLabelValues(GOOS, GOARCH, version, project, proof).Inc()

		if version == "latest" {
			v, err := getLatestVersion(r.Context(), project)
			if err != nil {
				log.Printf("Failed to retrieve latest version for %s: %v", project, err)
				http.Error(w, "Failed to retrieve latest version", http.StatusInternalServerError)
				return
			}
			version = v
		}

		switch project {
		case "age":
			ext := ".tar.gz"
			if GOOS == "windows" {
				ext = ".zip"
			}

			http.Redirect(w, r, "https://github.com/FiloSottile/age/releases/download/"+version+"/age-"+version+"-"+GOOS+"-"+GOARCH+ext+proof, http.StatusFound)
		case "mkcert":
			ext := ""
			if GOOS == "windows" {
				ext = ".exe"
			}

			http.Redirect(w, r, "https://github.com/FiloSottile/mkcert/releases/download/"+version+"/mkcert-"+version+"-"+GOOS+"-"+GOARCH+ext+proof, http.StatusFound)
		}
	})
}

var dlReqs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "dl_requests_total",
	Help: "dl.filippo.io requests processed, partitioned by GOOS, GOARCH, and version.",
}, []string{"GOOS", "GOARCH", "version", "project", "proof"})
