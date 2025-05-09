package main

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/crypto/x509roots/nss"
)

const NSSCertdata = "https://hg.mozilla.org/mozilla-central/raw-file/tip/security/nss/lib/ckfw/builtins/certdata.txt"
const GoogleRoots = "https://chromium.googlesource.com/chromium/src/+/main/net/data/ssl/chrome_root_store/root_store.md"
const AppleRoots = "https://support.apple.com/en-us/121672"

var AppleFingerprintRegex = regexp.MustCompile(`([0-9A-F][0-9A-F] ){31}[0-9A-F][0-9A-F]`)
var GoogleFingerprintRegex = regexp.MustCompile(`[0-9a-f]{64}`)

func main() {
	res, err := http.Get(NSSCertdata)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Fatalf("failed to get roots: %s", res.Status)
	}

	certs, err := nss.Parse(res.Body)
	if err != nil {
		log.Fatalf("failed to parse certdata: %v", err)
	}
	if len(certs) == 0 {
		log.Fatal("certdata.txt appears to contain zero roots")
	}

	sort.Slice(certs, func(i, j int) bool {
		subjI, subjJ := certs[i].X509.Subject.String(), certs[j].X509.Subject.String()
		if subjI != subjJ {
			return subjI < subjJ
		}
		return string(certs[i].X509.Raw) < string(certs[j].X509.Raw)
	})

	fmt.Println("Mozilla Roots")
	fmt.Println("=============")
	fmt.Println("")

	rootsByFingerprint := make(map[[32]byte][]byte)
	for _, c := range certs {
		fingerprint := sha256.Sum256(c.X509.Raw)
		rootsByFingerprint[fingerprint] = c.X509.Raw
		fmt.Printf("# %s\n# %X\n# from Mozilla root program\n%s\n", c.X509.Subject, fingerprint,
			pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.X509.Raw}))
	}

	localRootsByFingerprint := make(map[[32]byte][]byte)
	files, err := filepath.Glob("library/*.crt")
	if err != nil {
		log.Fatalf("failed to read certificate files: %v", err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read %s: %v", file, err)
		}
		block, _ := pem.Decode(data)
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Fatalf("failed to parse certificate from %s: %v", file, err)
		}
		fingerprint := sha256.Sum256(cert.Raw)
		localRootsByFingerprint[fingerprint] = cert.Raw
	}

	fmt.Println("Google Roots")
	fmt.Println("============")
	fmt.Println("")

	res, err = http.Get(GoogleRoots)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Fatalf("failed to get roots: %s", res.Status)
	}
	googleRoots, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	matches := GoogleFingerprintRegex.FindAllString(string(googleRoots), -1)
	if len(matches) < 50 {
		log.Fatalf("expected at least 50 Google roots, got %d", len(matches))
	}
	for _, h := range matches {
		f, err := hex.DecodeString(h)
		if err != nil {
			log.Fatal(err)
		}
		fingerprint := [32]byte(f)
		if _, ok := rootsByFingerprint[fingerprint]; ok {
			continue
		}
		if der, ok := localRootsByFingerprint[fingerprint]; ok {
			c, err := x509.ParseCertificate(der)
			if err != nil {
				log.Fatalf("failed to parse certificate: %v", err)
			}
			fmt.Printf("# %s\n# %X\n# from Google root program\n%s\n", c.Subject, fingerprint,
				pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
			rootsByFingerprint[fingerprint] = der
			continue
		}
		log.Fatalf("missing root https://crt.sh/?q=%X", fingerprint)
	}

	fmt.Println("Apple Roots")
	fmt.Println("===========")
	fmt.Println("")

	res, err = http.Get(AppleRoots)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Fatalf("failed to get roots: %s", res.Status)
	}
	appleRoots, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	matches = AppleFingerprintRegex.FindAllString(string(appleRoots), -1)
	if len(matches) < 50 {
		log.Fatalf("expected at least 50 Apple roots, got %d", len(matches))
	}
	for _, h := range matches {
		f, err := hex.DecodeString(strings.ReplaceAll(h, " ", ""))
		if err != nil {
			log.Fatal(err)
		}
		fingerprint := [32]byte(f)
		if _, ok := rootsByFingerprint[fingerprint]; ok {
			continue
		}
		if der, ok := localRootsByFingerprint[fingerprint]; ok {
			c, err := x509.ParseCertificate(der)
			if err != nil {
				log.Fatalf("failed to parse certificate: %v", err)
			}
			fmt.Printf("# %s\n# %X\n# from Apple root program\n%s\n", c.Subject, fingerprint,
				pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
			rootsByFingerprint[fingerprint] = der
			continue
		}
		log.Fatalf("missing root https://crt.sh/?q=%X", fingerprint)
	}

	fmt.Println("Extra Roots")
	fmt.Println("===========")
	fmt.Println("")

	extra, err := os.Open("extra.pem")
	if err != nil {
		log.Fatalf("failed to open extra.pem: %v", err)
	}
	defer extra.Close()
	io.Copy(os.Stdout, extra)
}
