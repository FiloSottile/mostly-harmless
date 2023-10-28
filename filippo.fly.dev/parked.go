package main

import "net/http"

func parked(mux *http.ServeMux) {
	handleWithCounter(mux, "valsorda.com/", http.RedirectHandler("https://filippo.io", http.StatusFound))
	handleWithCounter(mux, "valsorda.org/", http.RedirectHandler("https://filippo.io", http.StatusFound))
	handleWithCounter(mux, "filosottile.info/", http.RedirectHandler("https://filippo.io", http.StatusFound))

	handleWithCounter(mux, "ticketbleed.com/",
		http.RedirectHandler("https://filippo.io/ticketbleed/", http.StatusFound))
}
