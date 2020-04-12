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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gwatts/rootcerts/certparse"
)

type Root struct {
	c      *x509.Certificate
	source []string
}

type Fingerprint [32]byte

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: survey-roots [roots.pem]")
	}
	verbose := flag.Bool("v", false, "print source and hashes of roots")
	flag.Parse()

	var verboseOut, verboseErr io.Writer = os.Stdout, os.Stderr
	if !*verbose {
		verboseOut, verboseErr = ioutil.Discard, ioutil.Discard
	}

	var roots []*Root
	switch len(flag.Args()) {
	case 0:
		roots = loadSystemRoots()
	case 1:
		data, err := ioutil.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		roots = appendFromPEM(roots, data, os.Args[1])
	default:
		flag.Usage()
		os.Exit(1)
	}
	fmt.Fprintf(verboseErr, "[+] Loaded %d roots\n", len(roots))

	// The loading logic, which intentionally matches the crypto/x509
	// one, ends up brining in a lot of duplicates because it does not
	// stop at the first source.
	var uniqueRoots []*Root
	seen := make(map[Fingerprint]*Root)
	for _, root := range roots {
		fingerprint := spkiSubjectFingerprint(root.c)
		r, ok := seen[fingerprint]
		if !ok {
			uniqueRoots = append(uniqueRoots, root)
			seen[fingerprint] = root
		} else {
			r.source = append(r.source, root.source...)
		}
	}
	sort.Slice(uniqueRoots, func(i, j int) bool {
		return uniqueRoots[i].c.Subject.String() < uniqueRoots[j].c.Subject.String()
	})
	fmt.Fprintf(verboseErr, "[+] Found %d unique roots in target set\n", len(uniqueRoots))

	fmt.Fprintf(verboseErr, "[ ] Fetching Mozilla root store...\n")
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
	mozillaGood := make(map[Fingerprint]bool)
	for _, c := range mozillaCerts {
		if c.Trust&certparse.ServerTrustedDelegator != 0 {
			mozillaGood[spkiSubjectFingerprint(c.Cert)] = true
		}
	}
	fmt.Fprintf(verboseErr, "[+] Loaded %d Mozilla roots\n", len(mozillaGood))

	year := time.Now().Format("2006")
	fmt.Fprintf(verboseErr, "[ ] Fetching Argon%s root store...\n", year)
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
	ctGood := make(map[Fingerprint]bool)
	for _, der := range ctCerts.Certificates {
		c, err := x509.ParseCertificate(der)
		if err != nil {
			continue
		}
		ctGood[spkiSubjectFingerprint(c)] = true
	}
	fmt.Fprintf(verboseErr, "[+] Loaded %d roots from Argon%s CT log\n", len(ctGood), year)

	var notInMozilla, unknown int
	for _, root := range uniqueRoots {
		fingerprint := spkiSubjectFingerprint(root.c)
		if mozillaGood[fingerprint] {
			continue
		}
		if ctGood[fingerprint] {
			notInMozilla++
			fmt.Printf(" - %v\n", root.c.Subject)
		} else {
			unknown++
			fmt.Printf("!! %v\n", root.c.Subject)
		}
		fmt.Fprintf(verboseOut, "\tfrom %s\n", strings.Join(root.source, ", "))
		fmt.Fprintf(verboseOut, "\thttps://censys.io/authorities/%x\n", fingerprint)
		fmt.Fprintf(verboseOut, "\thttps://crt.sh/?q=%x\n", sha256.Sum256(root.c.Raw))
		fmt.Fprintf(verboseOut, "\n")
	}
	if notInMozilla+unknown > 0 && !*verbose {
		fmt.Printf("\n")
	}

	fmt.Printf("Found %d root(s) not in the Mozilla store, and %d completely unknown one(s).\n", notInMozilla, unknown)
}

func spkiSubjectFingerprint(c *x509.Certificate) Fingerprint {
	h := sha256.New()
	h.Write(c.RawSubjectPublicKeyInfo)
	h.Write(c.RawSubject)
	var out Fingerprint
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

func loadSystemRoots() []*Root {
	var roots []*Root

	for _, file := range certFiles {
		if data, err := ioutil.ReadFile(file); err == nil {
			roots = appendFromPEM(roots, data, file)
			break
		}
	}

	for _, directory := range certDirectories {
		fis, err := ioutil.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, fi := range fis {
			file := directory + "/" + fi.Name()
			if data, err := ioutil.ReadFile(file); err == nil {
				roots = appendFromPEM(roots, data, file)
			}
		}
	}

	return roots
}

func appendFromPEM(roots []*Root, pemCerts []byte, source string) []*Root {
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

		roots = append(roots, &Root{c: cert, source: []string{source}})
	}
	return roots
}
