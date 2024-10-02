package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

func c2sp(mux *http.ServeMux) {
	specRe := regexp.MustCompile(`^/([A-Za-z0-9-]+)$`)
	versRe := regexp.MustCompile(`^/([A-Za-z0-9-]+)@(v[a-z0-9-.]+)$`)
	cctvRe := regexp.MustCompile(`^/CCTV/([A-Za-z0-9-]+)$`)

	handleFuncWithCounter(mux, "c2sp.org/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("go-get") {
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			if strings.HasPrefix(r.URL.Path, "/CCTV/") || r.URL.Path == "/CCTV" {
				goGetReqs.WithLabelValues("CCTV", r.URL.Query().Get("go-get")).Inc()
				fmt.Fprint(w, `<head><meta name="go-import" content="c2sp.org/CCTV git https://github.com/C2SP/CCTV">`)
			} else {
				goGetReqs.WithLabelValues("C2SP", r.URL.Query().Get("go-get")).Inc()
				fmt.Fprint(w, `<head><meta name="go-import" content="c2sp.org git https://github.com/C2SP/C2SP">`)
			}
			return
		}

		if r.URL.Path == "/CCTV" {
			http.Redirect(w, r, "https://github.com/C2SP/CCTV/", http.StatusFound)
			return
		}

		if r.URL.Path == "/CCTV/ed25519vectors" { // legacy name
			http.Redirect(w, r, "https://c2sp.org/CCTV/ed25519", http.StatusFound)
			return
		}

		if r.URL.Path == "/sunlight" { // renamed spec
			http.Redirect(w, r, "https://c2sp.org/static-ct-api", http.StatusFound)
			return
		}

		if match := cctvRe.FindStringSubmatch(r.URL.Path); match != nil {
			http.Redirect(w, r, "https://github.com/C2SP/CCTV/tree/main/"+match[1], http.StatusFound)
			return
		}

		if match := versRe.FindStringSubmatch(r.URL.Path); match != nil {
			http.Redirect(w, r, "https://github.com/C2SP/C2SP/blob/"+match[1]+"/"+match[2]+"/"+match[1]+".md", http.StatusFound)
			return
		}

		if match := specRe.FindStringSubmatch(r.URL.Path); match != nil {
			http.Redirect(w, r, "https://github.com/C2SP/C2SP/blob/main/"+match[1]+".md", http.StatusFound)
			return
		}

		http.Redirect(w, r, "https://github.com/C2SP/C2SP", http.StatusFound)
	})
}
