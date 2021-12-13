package main

import (
	"context"
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

func getLatestVersion(ctx context.Context) (string, error) {
	rel, _, err := gitHubClient.Repositories.GetLatestRelease(ctx, "FiloSottile", "age")
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

	if _, err := getLatestVersion(context.Background()); err != nil {
		log.Println(err)
	}

	handleFuncWithCounter(mux, "dl.filippo.io/age/", func(w http.ResponseWriter, r *http.Request) {
		version := strings.TrimPrefix(r.URL.Path, "/age/")
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
		ext := ".tar.gz"
		if GOOS == "windows" {
			ext = ".zip"
		}

		dlReqs.WithLabelValues(GOOS, GOARCH, version).Inc()

		if version == "latest" {
			v, err := getLatestVersion(r.Context())
			if err != nil {
				log.Println(err)
				http.Error(w, "Failed to retrieve latest version", http.StatusInternalServerError)
				dlErrs.Inc()
				return
			}
			version = v
		}

		http.Redirect(w, r, "https://github.com/FiloSottile/age/releases/download/"+version+"/age-"+version+"-"+GOOS+"-"+GOARCH+ext, http.StatusMovedPermanently)
	})
}

var dlReqs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "dl_requests_total",
	Help: "dl.filippo.io requests processed, partitioned by GOOS, GOARCH, and version.",
}, []string{"GOOS", "GOARCH", "version"})
var dlErrs = promauto.NewCounter(prometheus.CounterOpts{
	Name: "dl_errors_total",
	Help: "dl.filippo.io errors while retrieving latest version.",
})
