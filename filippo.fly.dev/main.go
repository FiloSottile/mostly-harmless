package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	ageEncryption(mux)
	dlFilippo(mux)
	mkcert(mux)
	filippoIO(mux)

	s := &http.Server{
		Addr: ":" + port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Forwarded-Proto") == "http" {
				u := &url.URL{
					Scheme:   "https",
					Host:     r.Host,
					Path:     r.URL.Path,
					RawQuery: r.URL.RawQuery,
				}
				http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
				return
			}
			w.Header().Set("Strict-Transport-Security",
				"max-age=63072000; includeSubDomains; preload")
			w.Header().Set("Cache-Control", "public, max-age=300")
			mux.ServeHTTP(w, r)
		}),
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
	}

	log.Fatal(s.ListenAndServe())
}
