package main

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{Addr: ":9091", Handler: metricsMux,
		ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	go func() { log.Fatal(metricsServer.ListenAndServe()) }()

	mux := http.NewServeMux()
	ageEncryption(mux)
	dlFilippo(mux)
	blogFilippo(mux)
	mkcert(mux)
	c2sp(mux)
	filippoIO(mux)
	news(mux)

	s := &http.Server{
		Addr: ":8080",
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

var httpReqs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "HTTP requests processed, partitioned by handler.",
}, []string{"handler"})

func handleWithCounter(mux *http.ServeMux, pattern string, handler http.Handler) {
	mux.HandleFunc(pattern, func(rw http.ResponseWriter, r *http.Request) {
		httpReqs.WithLabelValues(pattern).Inc()
		handler.ServeHTTP(rw, r)
	})
}

func handleFuncWithCounter(mux *http.ServeMux, pattern string,
	handle func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(pattern, func(rw http.ResponseWriter, r *http.Request) {
		httpReqs.WithLabelValues(pattern).Inc()
		handle(rw, r)
	})
}
