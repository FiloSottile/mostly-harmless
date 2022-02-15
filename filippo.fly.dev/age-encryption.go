package main

import "net/http"

func ageEncryption(mux *http.ServeMux) {
	handleWithCounter(mux, "age-encryption.org/", http.RedirectHandler(
		"https://github.com/FiloSottile/age", http.StatusFound))

	handleWithCounter(mux, "age-encryption.org/v1", http.RedirectHandler(
		"https://docs.google.com/document/d/11yHom20CrsuX8KQJXBBw04s80Unjv8zCg_A7sPAX_9Y/preview",
		http.StatusFound))

	handleFuncWithCounter(mux, "age-encryption.org/stickers",
		func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Set("Content-Type", "text/plain")
			rw.Write([]byte("# age stickers! Send your address, including name and country,\n"))
			rw.Write([]byte("# to age@filippo.io encrypted to this recipient, and I'll mail\n"))
			rw.Write([]byte("# you some stickers in the next batch. Standing offer, no ETA.\n"))
			rw.Write([]byte("age1p6z5dwkjr4eu2rz6k6frumhxwc29q960fj7h0tajf0zywwylt5ysrwx8zr\n"))
		})
}
