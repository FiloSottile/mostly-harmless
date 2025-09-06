package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/mod/semver"
)

func main() {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{Addr: ":9091", Handler: metricsMux,
		ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	go func() { log.Fatal(metricsServer.ListenAndServe()) }()

	s := &http.Server{
		Addr:         ":8080",
		Handler:      handler(),
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		IdleTimeout:  10 * time.Minute,
	}

	log.Fatal(s.ListenAndServe())
}

func handler() http.Handler {
	mux := http.NewServeMux()

	dl(mux)
	redirects(mux)
	buttondown(mux)

	mux.Handle("filippo.io/", StaticHandler())

	mux.Handle("sunlight.dev/images/", StaticHandler())
	mux.Handle("sunlight.dev/{$}", HTMLHandler("sunlight.html"))

	mux.Handle("geomys.org/images/", StaticHandler())
	mux.Handle("geomys.org/{$}", HTMLHandler("geomys.html"))
	mux.Handle("geomys.org/fips140", HTMLHandler("fips140.html"))
	mux.Handle("geomys.org/fips140/essential/terms", HTMLHandler("tos.html"))
	mux.Handle("geomys.org/fips140/essential/subscribe", http.RedirectHandler(
		"https://buy.stripe.com/8wM4iufSY6q62as9AA", http.StatusFound))
	mux.Handle("geomys.org/fips140/essential/manage", http.RedirectHandler(
		"https://billing.stripe.com/p/login/8x29AU94R96B96dgeR2cg00", http.StatusFound))

	mux.Handle("age-encryption.org/{$}", http.RedirectHandler("https://github.com/FiloSottile/age", http.StatusFound))

	mux.Handle("age-encryption.org/design", http.RedirectHandler(
		"https://docs.google.com/document/d/11yHom20CrsuX8KQJXBBw04s80Unjv8zCg_A7sPAX_9Y/preview", http.StatusFound))

	mux.Handle("age-encryption.org/v1", http.RedirectHandler(
		"https://github.com/C2SP/C2SP/blob/main/age.md", http.StatusFound))

	mux.Handle("age-encryption.org/testkit", http.RedirectHandler(
		"https://github.com/C2SP/CCTV/tree/main/age", http.StatusFound))

	mux.Handle("mkcert.dev/{$}", http.RedirectHandler(
		"https://github.com/FiloSottile/mkcert", http.StatusFound))

	mux.Handle("mkcert.dev/mkcert-master.rb", http.RedirectHandler(
		"https://raw.githubusercontent.com/FiloSottile/mkcert/master/mkcert-master.rb", http.StatusMovedPermanently))

	mux.Handle("valsorda.com/{$}", http.RedirectHandler("https://filippo.io", http.StatusFound))
	mux.Handle("valsorda.org/{$}", http.RedirectHandler("https://filippo.io", http.StatusFound))
	mux.Handle("filosottile.info/{$}", http.RedirectHandler("https://filippo.io", http.StatusFound))

	mux.Handle("lycalopex.org/{$}", http.RedirectHandler("https://filippo.io", http.StatusFound))

	mux.Handle("ticketbleed.com/{$}", http.RedirectHandler("https://filippo.io/ticketbleed/", http.StatusFound))

	mux.Handle("geomys.dev/{$}", http.RedirectHandler("https://geomys.org", http.StatusFound))
	mux.Handle("geomys.it/{$}", http.RedirectHandler("https://geomys.org", http.StatusFound))

	mux.Handle("blog.filippo.io/", HostRedirectHandler("words.filippo.io", http.StatusMovedPermanently))

	mux.Handle("www.filippo.io/", HostRedirectHandler("filippo.io", http.StatusMovedPermanently))

	mux.Handle("c2sp.org/{$}", http.RedirectHandler("https://github.com/C2SP/C2SP/", http.StatusFound))
	mux.Handle("c2sp.org/CCTV", http.RedirectHandler("https://github.com/C2SP/CCTV/", http.StatusFound))
	mux.HandleFunc("c2sp.org/{name}", func(w http.ResponseWriter, r *http.Request) {
		if name, vers, ok := strings.Cut(r.PathValue("name"), "@"); ok {
			http.Redirect(w, r, "https://github.com/C2SP/C2SP/blob/"+name+"/"+vers+"/"+name+".md", http.StatusFound)
		} else {
			http.Redirect(w, r, "https://github.com/C2SP/C2SP/blob/main/"+name+".md", http.StatusFound)
		}
	})
	mux.HandleFunc("c2sp.org/CCTV/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		http.Redirect(w, r, "https://github.com/C2SP/CCTV/tree/main/"+name, http.StatusFound)
	})
	// Renamed test vectors and specs.
	mux.Handle("c2sp.org/CCTV/ed25519vectors", http.RedirectHandler("https://c2sp.org/CCTV/ed25519", http.StatusFound))
	mux.Handle("c2sp.org/sunlight", http.RedirectHandler("https://c2sp.org/static-ct-api", http.StatusFound))

	mux.Handle("mta-sts.filippo.io/.well-known/mta-sts.txt", MTASTSHandler())
	mux.Handle("mta-sts.bip.filippo.io/.well-known/mta-sts.txt", MTASTSHandler())
	mux.Handle("mta-sts.ml.filippo.io/.well-known/mta-sts.txt", MTASTSHandler())
	mux.Handle("mta-sts.geomys.org/.well-known/mta-sts.txt", MTASTSHandler())

	plausible := HostReverseProxyHandler("plausible.io")
	mux.Handle("filippo.io/js/script.js", plausible)
	mux.Handle("filippo.io/api/event", plausible)

	mux.Handle("filippo.io/age/info/refs", QueryPreservingRedirectHandler(
		"https://github.com/FiloSottile/age.git/info/refs", http.StatusFound))
	mux.Handle("filippo.io/yubikey-agent/info/refs", QueryPreservingRedirectHandler(
		"https://github.com/FiloSottile/yubikey-agent.git/info/refs", http.StatusFound))

	filippoIOModules := map[string]bool{
		"age":             true,
		"bigmod":          true,
		"cpace":           true,
		"csrf":            true,
		"edwards25519":    true,
		"hpke":            true,
		"intermediates":   true,
		"keygen":          true,
		"mkcert":          true,
		"mlkem768":        true,
		"mostly-harmless": true,
		"nistec":          true,
		"sunlight":        true,
		"torchwood":       true,
		"xaes256gcm":      true,
		"yubikey-agent":   true,
	}
	mux.Handle("filippo.io/{name}", PkgsiteHandler(filippoIOModules, StaticHandler()))
	mux.Handle("filippo.io/{name}/", PkgsiteHandler(filippoIOModules, StaticHandler()))
	goGetMux := http.NewServeMux()
	for name := range filippoIOModules {
		module := "filippo.io/" + name
		goGetMux.Handle(module+"/", GoImportHandler(module, "https://github.com/FiloSottile/"+name))
	}
	goGetMux.Handle("c2sp.org/", GoImportHandler("c2sp.org", "https://github.com/C2SP/C2SP"))
	goGetMux.Handle("c2sp.org/CCTV/", GoImportHandler("c2sp.org/CCTV", "https://github.com/C2SP/CCTV"))

	userAgents := NewTable(100)
	mux.Handle("filippo.io/heavy/{secret}/useragents", HeavyHitterHandler(userAgents))
	mux.Handle("filippo.io/heavy/{secret}/user-agents", HeavyHitterHandler(userAgents))

	notFound := NewTable(100)
	mux.Handle("filippo.io/heavy/{secret}/notfound", HeavyHitterHandler(notFound))
	mux.Handle("filippo.io/heavy/{secret}/404", HeavyHitterHandler(notFound))

	referrers := NewTable(500)
	mux.Handle("filippo.io/heavy/{secret}/referrers", HeavyHitterHandler(referrers))
	mux.Handle("filippo.io/heavy/{secret}/referers", HeavyHitterHandler(referrers))

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		w := &trackingResponseWriter{ResponseWriter: rw}
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")

		// Count popular User-Agents, and sample the pages they visit.
		userAgents.Count(r.UserAgent(), r.Host+r.URL.String())
		// Count popular external referers, and sample the pages they link to.
		if ref, err := url.Parse(r.Referer()); err == nil && r.Referer() != "" && ref.Host != r.Host {
			referrers.Count(r.Referer(), r.Host+r.URL.String())
		}
		// Track popular 404s, and sample their referrers.
		defer func() {
			if w.statusCode == http.StatusNotFound {
				notFound.Count(r.Host+r.URL.String(), r.Referer())
			}
		}()

		// Divert Go module downloads to the go-get handler.
		if r.URL.Query().Get("go-get") == "1" {
			goGetMux.ServeHTTP(w, r)
			return
		}

		_, pattern := mux.Handler(r)
		httpReqs.WithLabelValues(pattern).Inc()

		// Send browser navigation requests to Plausible Analytics.
		if r.Header.Get("Sec-Fetch-Mode") == "navigate" {
			defer func() {
				go plausiblePageview(r, w.statusCode, pattern)
			}()
		}

		mux.ServeHTTP(w, r)
	})
}

//go:embed filippo.io
var filippoIoContent embed.FS

func StaticHandler() http.Handler {
	content, err := fs.Sub(filippoIoContent, "filippo.io")
	if err != nil {
		log.Fatal(err)
	}
	handler := http.FileServer(http.FS(content))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tw := &trackingResponseWriter{ResponseWriter: w}
		handler.ServeHTTP(tw, r)
		if tw.statusCode == 200 {
			staticReqs.WithLabelValues(path.Clean(r.URL.Path)).Inc()
		}
	})
}

const mtsSTS = `version: STSv1
mode: enforce
mx: in1-smtp.messagingengine.com
mx: in2-smtp.messagingengine.com
max_age: 1209600
`

func MTASTSHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		io.WriteString(rw, mtsSTS)
	})
}

//go:embed *.html
var htmlContent embed.FS

func HTMLHandler(name string) http.Handler {
	content, err := htmlContent.ReadFile(name)
	if err != nil {
		log.Printf("Failed to read HTML file %q: %v", name, err)
		return http.NotFoundHandler()
	}
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/html; charset=UTF-8")
		rw.Write(content)
	})
}

func HeavyHitterHandler(table *Table) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.PathValue("secret") != os.Getenv("HEAVY_HITTER_SECRET") {
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}
		rw.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		for _, item := range table.Top(1000) {
			halfError := item.MaxError / 2
			fmt.Fprintf(rw, "%d (± %d)\t%q [%s]\n", item.Count-halfError, halfError, item.Value, item.Latest)
		}
	})
}

func PkgsiteHandler(names map[string]bool, fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		name, version, hasV := strings.Cut(name, "@")
		if !hasV {
			name, _, _ = strings.Cut(name, ".")
		}
		if hasV && !semver.IsValid(version) || !names[name] {
			fallback.ServeHTTP(rw, r)
			return
		}
		u := &url.URL{
			Scheme:   "https",
			Host:     "pkg.go.dev",
			Path:     "/filippo.io" + r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		if !hasV {
			path, symbol, hasSymbol := strings.Cut(r.URL.Path, ".")
			if hasSymbol {
				u.Path = "/filippo.io" + path
				u.Fragment = symbol
			}
		}
		http.Redirect(rw, r, u.String(), http.StatusFound)
	})
}

func GoImportHandler(module, repo string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		goGetReqs.WithLabelValues(module).Inc()
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		fmt.Fprintf(w, `<head><meta name="go-import" content="%s git %s">`, module, repo)
	})
}

func QueryPreservingRedirectHandler(target string, code int) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw, r, target+"?"+r.URL.RawQuery, code)
	})
}

func HostRedirectHandler(target string, code int) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		u := &url.URL{
			Scheme:   "https",
			Host:     target,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		http.Redirect(rw, r, u.String(), code)
	})
}

func HostReverseProxyHandler(target string) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.Header.Set("X-Forwarded-Host", pr.In.Host)
			pr.Out.Header.Set("X-Forwarded-Proto", pr.In.Header.Get("X-Forwarded-Proto"))
			pr.Out.Header.Set("X-Forwarded-For", pr.In.Header.Get("Fly-Client-IP"))
			pr.SetURL(&url.URL{Scheme: "https", Host: target})
		},
	}
}

type trackingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// Unwrap returns the original ResponseWriter for [http.ResponseController].
func (w *trackingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *trackingResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *trackingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

var plausibleClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 100,
	},
}

func plausiblePageview(r *http.Request, statusCode int, pattern string) {
	if r.PathValue("secret") != "" {
		return // Don't send Plausible events for secret paths.
	}
	// https://plausible.io/docs/events-api
	type Event struct {
		Domain   string         `json:"domain"`
		Name     string         `json:"name"`
		URL      string         `json:"url"`
		Referrer string         `json:"referrer"`
		Props    map[string]any `json:"props,omitempty"`
	}
	event := Event{
		Domain:   "filippo.io", // https://plausible.io/docs/subdomain-hostname-filter
		Name:     "pageview",
		URL:      r.Header.Get("X-Forwarded-Proto") + "://" + r.Host + r.URL.String(),
		Referrer: r.Referer(),
		Props: map[string]any{
			"HTTP Status Code": statusCode,
			"HTTP Method":      r.Method,
			"Mux Pattern":      pattern,
		},
	}
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal Plausible event: %v", err)
		plausibleEvents.WithLabelValues("false").Inc()
		return
	}
	req, err := http.NewRequest("POST", "https://plausible.io/api/event", bytes.NewReader(data))
	if err != nil {
		log.Printf("Failed to create Plausible event request: %v", err)
		plausibleEvents.WithLabelValues("false").Inc()
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", r.UserAgent())
	req.Header.Set("X-Forwarded-For", r.Header.Get("Fly-Client-IP"))
	if testing.Testing() {
		return
	}
	resp, err := plausibleClient.Do(req)
	if err != nil {
		log.Printf("Failed to send Plausible event: %v", err)
		plausibleEvents.WithLabelValues("false").Inc()
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Plausible event failed with status %d: %s", resp.StatusCode, body)
		plausibleEvents.WithLabelValues("false").Inc()
		return
	}
	plausibleEvents.WithLabelValues("true").Inc()
}

var goGetReqs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "goget_requests_total",
	Help: "go get requests processed, partitioned by repository name.",
}, []string{"name"})
var staticReqs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "static_requests_total",
	Help: "HTTP requests served from the FS.",
}, []string{"path"})
var httpReqs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "HTTP requests processed, partitioned by handler, excluding goget_requests_total.",
}, []string{"handler"})
var plausibleEvents = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "plausible_events_total",
	Help: "Plausible Analytics events sent.",
}, []string{"success"})
