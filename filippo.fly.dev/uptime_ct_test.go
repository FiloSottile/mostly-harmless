package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"slices"
	"testing"
	"time"
)

func TestCTUptimePrecert(t *testing.T) {
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test root"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keys := &ctUptimeKeyMaterial{root: root, leafPub: &leafKey.PublicKey, key: rootKey}

	l := &ctLogInfo{
		notAfterStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		notAfterLimit: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	name := "tuscolo2026h2.sunlight.geomys.org"
	minute := time.Date(2026, 8, 10, 20, 45, 0, 0, time.UTC)

	parse := func(minute time.Time) *x509.Certificate {
		t.Helper()
		der, err := ctUptimePrecert(name, minute, l, keys)
		if err != nil {
			t.Fatal(err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		return cert
	}

	a, b := parse(minute), parse(minute)
	if !bytes.Equal(a.RawTBSCertificate, b.RawTBSCertificate) {
		t.Error("TBSCertificate is not deterministic")
	}
	c := parse(minute.Add(time.Minute))
	if bytes.Equal(a.RawTBSCertificate, c.RawTBSCertificate) {
		t.Error("TBSCertificate does not change with the minute")
	}
	if a.SerialNumber.Cmp(c.SerialNumber) == 0 {
		t.Error("serial number does not change with the minute")
	}

	if !a.NotBefore.Equal(minute.Add(-1 * time.Hour)) {
		t.Errorf("NotBefore = %v", a.NotBefore)
	}
	if !a.NotAfter.Equal(minute.Add(24 * time.Hour)) {
		t.Errorf("NotAfter = %v", a.NotAfter)
	}
	want := []string{"uptime.geomys.org", "45.20.10.08.2026.tuscolo2026h2-sunlight-geomys-org.uptime.geomys.org"}
	if !slices.Equal(a.DNSNames, want) {
		t.Errorf("DNSNames = %q", a.DNSNames)
	}
	if !slices.ContainsFunc(a.Extensions, func(e pkix.Extension) bool {
		return e.Id.Equal(ctPoisonOID) && e.Critical
	}) {
		t.Error("missing critical poison extension")
	}
	if !slices.Contains(a.ExtKeyUsage, x509.ExtKeyUsageServerAuth) {
		t.Error("missing ServerAuth EKU")
	}
	if a.SerialNumber.Sign() != 1 {
		t.Errorf("SerialNumber = %v", a.SerialNumber)
	}

	early := parse(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	if !early.NotAfter.Equal(l.notAfterStart) {
		t.Errorf("NotAfter = %v, want clamped to %v", early.NotAfter, l.notAfterStart)
	}
	late := parse(time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC))
	if !late.NotAfter.Equal(l.notAfterLimit.Add(-1 * time.Second)) {
		t.Errorf("NotAfter = %v, want clamped to %v", late.NotAfter, l.notAfterLimit.Add(-1*time.Second))
	}
}
