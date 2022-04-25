package main

import (
	"net/http"
	"regexp"
)

func c2sp(mux *http.ServeMux) {
	specRe := regexp.MustCompile(`^/[a-z0-9-]+$`)

	handleFuncWithCounter(mux, "c2sp.org/", func(w http.ResponseWriter, r *http.Request) {
		if path := r.URL.Path; specRe.MatchString(path) {
			http.Redirect(w, r, "https://github.com/C2SP/C2SP/blob/main"+path+".md", http.StatusFound)
			return
		}
		http.Redirect(w, r, "https://github.com/C2SP/C2SP", http.StatusFound)
	})
}
