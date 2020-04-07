// Copyright 2019 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package main

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gwatts/rootcerts/certparse"
)

func main() {
	var roots []*x509.Certificate
	switch len(os.Args) {
	case 1:
		roots = loadSystemRoots()
	case 2:
		data, err := ioutil.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		roots = appendFromPEM(roots, data)
	default:
		log.Fatal("usage: survey-roots [roots.pem]")
	}
	log.Printf("[+] Loaded %d roots", len(roots))

	// The loading logic, which intentionally matches the crypto/x509
	// one, ends up brining in a lot of duplicates because it does not
	// stop at the first source.
	uniqueRoots := make(map[[32]byte]*x509.Certificate)
	for _, root := range roots {
		fingerprint := spkiSubjectFingerprint(root)
		uniqueRoots[fingerprint] = root
	}
	log.Printf("[+] Found %d unique roots in target set", len(uniqueRoots))

	log.Printf("[ ] Fetching Mozilla root store...")
	c := &http.Client{Timeout: 20 * time.Second}
	resp, err := c.Get("https://hg.mozilla.org/releases/mozilla-release/raw-file/default/security/nss/lib/ckfw/builtins/certdata.txt")
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("hg.mozilla.org GET failed: %v", resp.Status)
	}
	mozillaCerts, err := certparse.ReadTrustedCerts(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	mozillaGood := make(map[[32]byte]bool)
	for _, c := range mozillaCerts {
		if c.Trust&certparse.ServerTrustedDelegator != 0 {
			mozillaGood[spkiSubjectFingerprint(c.Cert)] = true
		}
	}
	log.Printf("[+] Loaded %d Mozilla roots", len(mozillaGood))

	year := time.Now().Format("2006")
	log.Printf("[ ] Fetching Argon%s root store...", year)
	resp, err = c.Get("https://ct.googleapis.com/logs/argon" + year + "/ct/v1/get-roots")
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("/ct/v1/get-roots GET failed: %v", resp.Status)
	}
	var ctCerts struct {
		Certificates [][]byte
	}
	if err := json.NewDecoder(resp.Body).Decode(&ctCerts); err != nil {
		log.Fatal(err)
	}
	ctGood := make(map[[32]byte]bool)
	for _, der := range ctCerts.Certificates {
		c, err := x509.ParseCertificate(der)
		if err != nil {
			continue
		}
		ctGood[spkiSubjectFingerprint(c)] = true
	}
	log.Printf("[+] Loaded %d roots from Argon%s CT log", len(ctGood), year)

	for _, root := range uniqueRoots {
		fingerprint := spkiSubjectFingerprint(root)
		if mozillaGood[fingerprint] {
			continue
		}
		if ctGood[fingerprint] {
			log.Printf("[-] Root not in the Mozilla store")
		} else {
			log.Printf("[!] Unknown root")
		}
		log.Printf("\t%v", root.Subject)
		log.Printf("\t\thttps://censys.io/authorities/%x", fingerprint)
		log.Printf("\t\thttps://crt.sh/?q=%x", sha256.Sum256(root.Raw))
	}
}

func spkiSubjectFingerprint(c *x509.Certificate) [32]byte {
	h := sha256.New()
	h.Write(c.RawSubjectPublicKeyInfo)
	h.Write(c.RawSubject)
	var out [32]byte
	h.Sum(out[:0])
	return out
}

var certFiles = []string{
	"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
	"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
	"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
	"/etc/pki/tls/cacert.pem",                           // OpenELEC
	"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
	"/etc/ssl/cert.pem",                                 // Alpine Linux
}

var certDirectories = []string{
	"/etc/ssl/certs",               // SLES10/SLES11
	"/system/etc/security/cacerts", // Android
	"/usr/local/share/certs",       // FreeBSD
	"/etc/pki/tls/certs",           // Fedora/RHEL
	"/etc/openssl/certs",           // NetBSD
	"/var/ssl/certs",               // AIX
}

func loadSystemRoots() []*x509.Certificate {
	var roots []*x509.Certificate

	for _, file := range certFiles {
		if data, err := ioutil.ReadFile(file); err == nil {
			roots = appendFromPEM(roots, data)
			break
		}
	}

	for _, directory := range certDirectories {
		fis, err := ioutil.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, fi := range fis {
			if data, err := ioutil.ReadFile(directory + "/" + fi.Name()); err == nil {
				roots = appendFromPEM(roots, data)
			}
		}
	}

	return roots
}

func appendFromPEM(roots []*x509.Certificate, pemCerts []byte) []*x509.Certificate {
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}

		roots = append(roots, cert)
	}
	return roots
}
