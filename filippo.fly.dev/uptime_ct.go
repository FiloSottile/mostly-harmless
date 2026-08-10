package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/asn1"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"filippo.io/sunlight"
)

// ctAddPreChain submits a test precertificate to a CT log, and verifies the
// returned SCT and its inclusion in the latest checkpoint.
//
// The precertificate's TBSCertificate is a deterministic function of the log
// and of the time truncated to the minute, so logs deduplicate concurrent
// submissions from multiple instances of this service. Successful results are
// cached for the rest of the minute, to rate-limit submissions.
func ctAddPreChain(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSuffix(r.PathValue("log"), "/")
	l, err := ctLog(name)
	if err != nil {
		msg := fmt.Sprintf("error fetching log list: %v", err)
		http.Error(w, msg, http.StatusBadGateway)
		return
	}
	if l == nil {
		http.Error(w, "unknown log", http.StatusBadRequest)
		return
	}
	// Shards are expected to go read-only at the end of their temporal
	// interval, don't alert on them.
	if !time.Now().Before(l.notAfterLimit) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "not submitting to read-only shard, temporal interval ended %s\n",
			l.notAfterLimit.Format(time.RFC3339))
		return
	}
	keys, err := ctUptimeKeys()
	if err != nil {
		msg := fmt.Sprintf("error loading key material: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	state := ctUptimeState(name)
	state.Lock()
	defer state.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	minute := time.Now().UTC().Truncate(time.Minute)
	if state.minute.Equal(minute) {
		w.Write(state.body)
		return
	}
	body, err := ctUptimeCheck(r.Context(), name, minute, l, keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	state.minute, state.body = minute, body
	w.Write(body)
}

func ctUptimeCheck(ctx context.Context, name string, minute time.Time, l *ctLogInfo, keys *ctUptimeKeyMaterial) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	precert, err := ctUptimePrecert(name, minute, l, keys)
	if err != nil {
		return nil, fmt.Errorf("error generating precertificate: %v", err)
	}
	reqBody, err := json.Marshal(struct {
		Chain [][]byte `json:"chain"`
	}{[][]byte{precert, keys.root.Raw}})
	if err != nil {
		return nil, fmt.Errorf("error encoding submission: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST",
		l.submissionURL+"ct/v1/add-pre-chain", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error preparing submission: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := uptimeClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error submitting precertificate: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("error submitting precertificate: status code %d: %q", resp.StatusCode, body)
	}
	var sct struct {
		SCTVersion int    `json:"sct_version"`
		ID         []byte `json:"id"`
		Timestamp  int64  `json:"timestamp"`
		Extensions []byte `json:"extensions"`
		Signature  []byte `json:"signature"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sct); err != nil {
		return nil, fmt.Errorf("error parsing SCT response: %v", err)
	}
	if sct.SCTVersion != 0 || len(sct.ID) != 32 || len(sct.Extensions) >= 1<<16 {
		return nil, fmt.Errorf("invalid SCT response")
	}
	if time.Since(time.UnixMilli(sct.Timestamp)) > 5*time.Minute {
		return nil, fmt.Errorf("stale SCT: timestamp %d", sct.Timestamp)
	}
	ext, err := sunlight.ParseExtensions(sct.Extensions)
	if err != nil {
		return nil, fmt.Errorf("error parsing SCT extensions: %v", err)
	}

	// Reassemble the JSON response into a TLS-encoded
	// SignedCertificateTimestamp for CheckInclusion.
	sctBytes := []byte{0 /* v1 */}
	sctBytes = append(sctBytes, sct.ID...)
	sctBytes = binary.BigEndian.AppendUint64(sctBytes, uint64(sct.Timestamp))
	sctBytes = binary.BigEndian.AppendUint16(sctBytes, uint16(len(sct.Extensions)))
	sctBytes = append(sctBytes, sct.Extensions...)
	sctBytes = append(sctBytes, sct.Signature...)

	client, err := sunlight.NewClient(&sunlight.ClientConfig{
		MonitoringPrefix: l.monitoringURL,
		PublicKey:        l.key,
		HTTPClient:       uptimeClient,
		UserAgent:        "https://uptime.geomys.org/ct/ (ct@geomys.org)",
	})
	if err != nil {
		return nil, fmt.Errorf("error creating monitoring client: %v", err)
	}
	checkpoint, n, err := client.Checkpoint(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching checkpoint: %v", err)
	}
	for checkpoint.N <= ext.LeafIndex {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for inclusion of leaf %d in tree of size %d", ext.LeafIndex, checkpoint.N)
		case <-time.After(1 * time.Second):
		}
		checkpoint, n, err = client.Checkpoint(ctx)
		if err != nil {
			return nil, fmt.Errorf("error fetching checkpoint: %v", err)
		}
	}
	t, err := sunlight.RFC6962SignatureTimestamp(n.Sigs[0])
	if err != nil {
		return nil, fmt.Errorf("error parsing checkpoint signature: %v", err)
	}
	if time.Since(time.UnixMilli(t)) > 5*time.Minute {
		return nil, fmt.Errorf("stale checkpoint: timestamp %d", t)
	}
	if t < sct.Timestamp {
		return nil, fmt.Errorf("checkpoint timestamp %d is older than SCT timestamp %d", t, sct.Timestamp)
	}
	entry, _, err := client.CheckInclusion(ctx, checkpoint.Tree, sctBytes)
	if err != nil {
		return nil, fmt.Errorf("error checking inclusion: %v", err)
	}
	if !entry.IsPrecert || entry.IssuerKeyHash != sha256.Sum256(keys.root.RawSubjectPublicKeyInfo) {
		return nil, fmt.Errorf("log entry does not match submission")
	}
	return fmt.Appendf(nil, "SCT verified and included, timestamp %d, leaf index %d, tree size %d\n",
		sct.Timestamp, ext.LeafIndex, checkpoint.N), nil
}

var ctPoisonOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 3}

func ctUptimePrecert(name string, minute time.Time, l *ctLogInfo, keys *ctUptimeKeyMaterial) ([]byte, error) {
	minute = minute.UTC()
	// Only the TBSCertificate needs to be deterministic, as logs deduplicate
	// on it (along with the issuer key hash), not on the signed certificate.
	serial := sha256.Sum256(fmt.Appendf(nil, "ct-uptime/%s/%d", name, minute.Unix()))
	label := fmt.Sprintf("%02d.%02d.%02d.%02d.%04d.%s.uptime.geomys.org",
		minute.Minute(), minute.Hour(), minute.Day(), minute.Month(), minute.Year(),
		ctDNSLabel(name))
	notAfter := minute.Add(24 * time.Hour)
	if notAfter.Before(l.notAfterStart) {
		notAfter = l.notAfterStart
	}
	if !notAfter.Before(l.notAfterLimit) {
		notAfter = l.notAfterLimit.Add(-1 * time.Second)
	}
	tmpl := &x509.Certificate{
		SerialNumber: new(big.Int).SetBytes(serial[:16]),
		Subject: pkix.Name{
			Organization: []string{"Geomys"},
			CommonName:   "uptime.geomys.org",
		},
		// A NotBefore older than 48 hours would make the submission low
		// priority, as it couldn't be a certificate issuance latency bottleneck.
		NotBefore:   minute.Add(-1 * time.Hour),
		NotAfter:    notAfter,
		DNSNames:    []string{"uptime.geomys.org", label},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		ExtraExtensions: []pkix.Extension{
			{Id: ctPoisonOID, Critical: true, Value: []byte{0x05, 0x00}},
		},
	}
	return x509.CreateCertificate(rand.Reader, tmpl, keys.root, keys.leafPub, keys.key)
}

func ctDNSLabel(name string) string {
	label := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, name)
	if len(label) > 63 {
		label = label[:63]
	}
	return label
}

// uptime_ct.pem holds the root certificate and the leaf public key.
//
//go:embed uptime_ct.pem
var ctUptimePEM []byte

type ctUptimeKeyMaterial struct {
	root    *x509.Certificate
	leafPub crypto.PublicKey
	key     *ecdsa.PrivateKey
}

var ctUptimeKeys = sync.OnceValues(func() (*ctUptimeKeyMaterial, error) {
	certBlock, rest := pem.Decode(ctUptimePEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid uptime_ct.pem")
	}
	root, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, err
	}
	pubBlock, _ := pem.Decode(rest)
	if pubBlock == nil || pubBlock.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("invalid uptime_ct.pem")
	}
	leafPub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, err
	}
	keyBlock, _ := pem.Decode([]byte(os.Getenv("CT_UPTIME_ROOT_KEY")))
	if keyBlock == nil || keyBlock.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("missing or invalid CT_UPTIME_ROOT_KEY")
	}
	k, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := k.(*ecdsa.PrivateKey)
	if !ok || !key.PublicKey.Equal(root.PublicKey) {
		return nil, fmt.Errorf("CT_UPTIME_ROOT_KEY does not match the root certificate")
	}
	return &ctUptimeKeyMaterial{root: root, leafPub: leafPub, key: key}, nil
})

type ctLogInfo struct {
	submissionURL string // with trailing slash
	monitoringURL string // with trailing slash
	key           crypto.PublicKey
	notAfterStart time.Time
	notAfterLimit time.Time
}

var ctLogList struct {
	sync.Mutex
	fetched time.Time
	logs    map[string]*ctLogInfo
}

// ctLog returns the tiled log with the given submission URL (without scheme
// and trailing slash) from the CT log list, or nil if there is none.
func ctLog(name string) (*ctLogInfo, error) {
	ctLogList.Lock()
	defer ctLogList.Unlock()
	if time.Since(ctLogList.fetched) > 6*time.Hour {
		logs, err := fetchCTLogList()
		if err != nil && ctLogList.logs == nil {
			return nil, err
		}
		if err == nil {
			ctLogList.logs, ctLogList.fetched = logs, time.Now()
		}
		// On error, keep serving the previous list.
	}
	return ctLogList.logs[name], nil
}

func fetchCTLogList() (map[string]*ctLogInfo, error) {
	resp, err := uptimeClient.Get("https://www.gstatic.com/ct/log_list/v3/all_logs_list.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}
	var list struct {
		Operators []struct {
			TiledLogs []struct {
				Key              []byte `json:"key"`
				SubmissionURL    string `json:"submission_url"`
				MonitoringURL    string `json:"monitoring_url"`
				TemporalInterval struct {
					StartInclusive time.Time `json:"start_inclusive"`
					EndExclusive   time.Time `json:"end_exclusive"`
				} `json:"temporal_interval"`
			} `json:"tiled_logs"`
		} `json:"operators"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	logs := make(map[string]*ctLogInfo)
	for _, op := range list.Operators {
		for _, l := range op.TiledLogs {
			key, err := x509.ParsePKIXPublicKey(l.Key)
			if err != nil || !strings.HasPrefix(l.SubmissionURL, "https://") ||
				l.TemporalInterval.StartInclusive.IsZero() || l.TemporalInterval.EndExclusive.IsZero() {
				continue
			}
			name := strings.TrimSuffix(strings.TrimPrefix(l.SubmissionURL, "https://"), "/")
			logs[name] = &ctLogInfo{
				submissionURL: strings.TrimSuffix(l.SubmissionURL, "/") + "/",
				monitoringURL: strings.TrimSuffix(l.MonitoringURL, "/") + "/",
				key:           key,
				notAfterStart: l.TemporalInterval.StartInclusive,
				notAfterLimit: l.TemporalInterval.EndExclusive,
			}
		}
	}
	return logs, nil
}

var ctUptimeStates struct {
	sync.Mutex
	m map[string]*ctUptimeStateEntry
}

type ctUptimeStateEntry struct {
	sync.Mutex
	minute time.Time
	body   []byte
}

func ctUptimeState(name string) *ctUptimeStateEntry {
	ctUptimeStates.Lock()
	defer ctUptimeStates.Unlock()
	if ctUptimeStates.m == nil {
		ctUptimeStates.m = make(map[string]*ctUptimeStateEntry)
	}
	if e, ok := ctUptimeStates.m[name]; ok {
		return e
	}
	e := &ctUptimeStateEntry{}
	ctUptimeStates.m[name] = e
	return e
}
