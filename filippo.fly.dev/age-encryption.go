package main

import "net/http"

func ageEncryption(mux *http.ServeMux) {
	handleWithCounter(mux, "age-encryption.org/", http.RedirectHandler(
		"https://github.com/FiloSottile/age", http.StatusFound))

	handleWithCounter(mux, "age-encryption.org/v1", http.RedirectHandler(
		"https://docs.google.com/document/d/11yHom20CrsuX8KQJXBBw04s80Unjv8zCg_A7sPAX_9Y/preview",
		http.StatusFound))
}
