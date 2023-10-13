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

func dlFilippo(mux *http.ServeMux) {
	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	))
	tc.Timeout = 10 * time.Second
	gitHubClient = github.NewClient(tc)

	if _, err := getLatestVersion(context.Background(), "age"); err != nil {
		log.Println(err)
	}
	if _, err := getLatestVersion(context.Background(), "mkcert"); err != nil {
		log.Println(err)
	}

	handleFuncWithCounter(mux, "dl.filippo.io/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprintln(w, "User-agent: *")
			fmt.Fprintln(w, "Disallow: /")
			return
		}

		var version, project string
		switch {
		case strings.HasPrefix(r.URL.Path, "/age/"):
			project = "age"
			version = strings.TrimPrefix(r.URL.Path, "/age/")
		case strings.HasPrefix(r.URL.Path, "/mkcert/"):
			project = "mkcert"
			version = strings.TrimPrefix(r.URL.Path, "/mkcert/")
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

		dlReqs.WithLabelValues(GOOS, GOARCH, version, project).Inc()

		if version == "latest" {
			v, err := getLatestVersion(r.Context(), project)
			if err != nil {
				log.Println(err)
				http.Error(w, "Failed to retrieve latest version", http.StatusInternalServerError)
				dlErrs.WithLabelValues(project).Inc()
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

			http.Redirect(w, r, "https://github.com/FiloSottile/age/releases/download/"+version+"/age-"+version+"-"+GOOS+"-"+GOARCH+ext, http.StatusMovedPermanently)
		case "mkcert":
			ext := ""
			if GOOS == "windows" {
				ext = ".exe"
			}

			http.Redirect(w, r, "https://github.com/FiloSottile/mkcert/releases/download/"+version+"/mkcert-"+version+"-"+GOOS+"-"+GOARCH+ext, http.StatusMovedPermanently)
		}
	})
}

var dlReqs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "dl_requests_total",
	Help: "dl.filippo.io requests processed, partitioned by GOOS, GOARCH, and version.",
}, []string{"GOOS", "GOARCH", "version", "project"})
var dlErrs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "dl_errors_total",
	Help: "dl.filippo.io errors while retrieving latest version.",
}, []string{"project"})
