package main

import "net/http"

func blogFilippo(mux *http.ServeMux) {
	handleFuncWithCounter(mux, "blog.filippo.io/", func(rw http.ResponseWriter, r *http.Request) {
		newURL := *r.URL
		newURL.Scheme = "https"
		newURL.Host = "words.filippo.io"
		http.Redirect(rw, r, newURL.String(), http.StatusMovedPermanently)
	})
}
