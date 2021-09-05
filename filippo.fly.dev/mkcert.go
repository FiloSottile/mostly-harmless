package main

import "net/http"

func mkcert(mux *http.ServeMux) {
	mux.Handle("mkcert.dev/", http.RedirectHandler(
		"https://github.com/FiloSottile/mkcert", http.StatusFound))

	mux.Handle("mkcert.dev/mkcert-master.rb", http.RedirectHandler(
		"https://raw.githubusercontent.com/FiloSottile/mkcert/master/mkcert-master.rb",
		http.StatusMovedPermanently))
}
