package main

import (
	"net/http"
	"strings"
)

func redirects(mux *http.ServeMux) {
	// Miscellaneous redirects
	for path, url := range map[string]string{
		"/fakenews/":          "https://web.archive.org/web/20221128012457/https://filippo.io/fakenews/",
		"/a-different-CT-log": "https://docs.google.com/document/d/1YsxLGZxYE1KTCTjDK2Ol-bcTzbrI313SZ1QWgqmnRDc/edit",
		"/rwc2023/talk":       "https://iacr.org/submit/files/slides/2023/rwc/rwc2023/131/slides.pdf",
		"/rwc2023":            "https://filippo.io/rwc2023/",
		"/ticketbleed/":       "https://filippo.io/Ticketbleed/",
		"/heartbleed/":        "https://filippo.io/Heartbleed/",
		"/cve-2016-2107/":     "https://filippo.io/CVE-2016-2107/",
		"/badfish/":           "https://filippo.io/Badfish/",
		"/hitb":               "https://imgur.com/a/3NkeN",
		"/hitb-slides":        "https://www.dropbox.com/s/bzptq3bvbwr0vqf/HITB.pdf?dl=0",
		"/hack.lu":            "https://speakerdeck.com/filosottile/the-heartbleed-test-adventure-at-hack-dot-lu-2014",
		"/fuzz-talk":          "https://speakerdeck.com/filosottile/automated-testing-with-go-fuzz",
		"/entropy-talk":       "https://speakerdeck.com/filosottile/the-plain-simple-reality-of-entropy",
		"/entropy-talk-ccc":   "https://speakerdeck.com/filosottile/the-plain-simple-reality-of-entropy-at-32c3",
		"/newsletter":         "https://buttondown.com/cryptography-dispatches/",
		"/age-design":         "https://docs.google.com/document/d/11yHom20CrsuX8KQJXBBw04s80Unjv8zCg_A7sPAX_9Y/preview",
		"/age/report":         "https://github.com/FiloSottile/age/issues/new/choose",
		"/age/age.1":          "https://htmlpreview.github.io/?https://github.com/FiloSottile/age/blob/main/doc/age.1.html",
		"/age/age-keygen.1":   "https://htmlpreview.github.io/?https://github.com/FiloSottile/age/blob/main/doc/age-keygen.1.html",
		"/CV/":                "https://blog.filippo.io/hi/",
		"/atom.xml":           "https://blog.filippo.io/rss/",
		"/maintenance":        "https://github.com/FiloSottile/FiloSottile/blob/main/maintenance.md",
		"/internship":         "https://docs.google.com/document/d/1e6dNtdTmHWmv2U6C407MV5N_wS5aX3LAJ6K_T6bShkE/edit",
		"/hpke-pq":            "https://github.com/FiloSottile/hpke/blob/main/hpke-pq.md",

		"/psa-enable-automatic-updates-please/":                 "https://blog.filippo.io/psa-enable-automatic-updates-please/",
		"/salt-and-pepper/":                                     "https://blog.filippo.io/salt-and-pepper/",
		"/the-heartbleed-test-at-owasp-slash-nyu-poly/":         "https://blog.filippo.io/the-heartbleed-test-at-owasp-slash-nyu-poly/",
		"/on-keybase-dot-io-and-encrypted-private-key-sharing/": "https://blog.filippo.io/on-keybase-dot-io-and-encrypted-private-key-sharing/",
		"/native-scrolling-and-iterm2/":                         "https://blog.filippo.io/native-scrolling-and-iterm2/",
		"/my-remote-shell-session-setup/":                       "https://blog.filippo.io/my-remote-shell-session-setup/",
		"/why-go-is-elegant-and-makes-my-code-elegant/":         "https://blog.filippo.io/why-go-is-elegant-and-makes-my-code-elegant/",
		// TODO: make this blog post URL shorter
		"/how-the-new-gmail-image-proxy-works-and-what-does-this-mean-for-you/": "https://blog.filippo.io/how-the-new-gmail-image-proxy-works-and-what-this-means-for-you/",
		"/the-ecb-penguin/": "https://blog.filippo.io/the-ecb-penguin/",

		// GothamGo 2023 QR codes
		"/gg0": "https://cs.opensource.google/go/go/+/refs/tags/go1.20rc1:src/crypto/ecdh/ecdh_test.go;l=423-489",
		"/gg1": "https://go.dev/blog/tls-cipher-suites",
		"/gg2": "https://words.filippo.io/dispatches/near-miss/",
		"/gg3": "https://words.filippo.io/dispatches/certificate-interning/",
		"/gg4": "https://words.filippo.io/full-time-maintainer/",
		"/gg5": "https://filippo.io/newsletter",
		"/gg6": "https://go-review.googlesource.com/c/go/+/276272",
	} {
		if strings.HasSuffix(path, "/") {
			path = path + "{$}"
		}
		mux.Handle("filippo.io"+path, http.RedirectHandler(url, http.StatusFound))
	}
}
