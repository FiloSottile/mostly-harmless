package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func buttondown(mux *http.ServeMux) {
	redirect := func(fragment, target string) {
		mux.HandleFunc("buttondown.filippo.io/archive/"+fragment+"/",
			func(rw http.ResponseWriter, r *http.Request) {
				httpReqs.WithLabelValues("[buttondown]").Inc()
				redirectReqs.WithLabelValues(fragment).Inc()
				http.Redirect(rw, r, target, http.StatusFound)
			})
	}
	email := func(new string, old ...string) {
		for _, slug := range old {
			redirect(slug, "https://words.filippo.io/dispatches/"+new+"/")
		}
	}
	handleWithCounter(mux, "buttondown.filippo.io/{$}", http.RedirectHandler("https://words.filippo.io/"))

	email("openpgp-is-broken",
		"626bb4fe-7d48-4cb7-a88d-206f9c38a921", "cryptography-dispatches-hello-world-and-openpgp")
	email("linux-csprng",
		"0b662832-a9a7-4e07-82fa-b97804618b98", "cryptography-dispatches-the-linux-csprng-is-now")
	email("go-1-14-crypto",
		"7909a70e-45b9-417d-b094-f0ec6b94b605", "cryptography-dispatches-new-crypto-in-go-114")
	email("openssh-fido2",
		"9cd031bc-4b83-4ba6-9134-98678f9abf74", "cryptography-dispatches-openssh-82-just-works")
	email("x25519-associative",
		"1ad8d5d9-7531-4ed4-bf21-f3c13d8be128", "cryptography-dispatches-is-x25519-associative")
	email("dsa",
		"557475c5-9781-47e0-a640-5734bc849bc7", "cryptography-dispatches-dsa-is-past-its-prime")
	email("replace-pgp-with-https",
		"505a859e-964d-4d15-9ad8-7ad0f45e1345", "cryptography-dispatches-replace-pgp-with-an-https")
	email("registries-considered-harmful",
		"8ea4e389-f8ca-4643-8416-4311436b090b", "cryptography-dispatches-registries-considered")
	email("nacl-api",
		"59c99d2e-9fd7-4859-934b-f52cf254e6b2", "cryptography-dispatches-nacl-is-not-a-high-level")
	email("reconstruct-vs-validate",
		"9efd8ad0-2b31-4319-b71b-34bf356836a9", "cryptography-dispatches-reconstruct-instead-of")
	email("edwards25519-formulas",
		"e625193c-7981-4295-9df5-f94fa694064a", "cryptography-dispatches-re-deriving-the")
	email("telegram-ecdh",
		"45cace9a-4f74-4591-8fd1-8ae54d14e156", "cryptography-dispatches-the-most-backdoor-looking")
	email("cipher-suite-ordering",
		"de1e8d9e-186d-4ae3-bea6-09a9a90f0ffa", "from-the-go-blog-automatic-cipher-suite-ordering")
}

var buttondownReqs = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "buttondown_requests_total",
	Help: "Buttondown redirects, partitioned by path.",
}, []string{"path"})
