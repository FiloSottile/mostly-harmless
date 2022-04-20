package main

import "net/http"

func c2sp(mux *http.ServeMux) {
	handleWithCounter(mux, "c2sp.org/", http.RedirectHandler(
		"https://github.com/C2SP/C2SP", http.StatusFound))

	handleWithCounter(mux, "c2sp.org/age", http.RedirectHandler(
		"https://github.com/C2SP/C2SP/blob/main/age.md", http.StatusFound))
}
