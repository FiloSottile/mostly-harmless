package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"filippo.io/torchwood"
	"golang.org/x/mod/sumdb/note"
)

var uptimeClient = &http.Client{
	Timeout: 5 * time.Second,
}

func uptime(mux *http.ServeMux) {
	mux.Handle("uptime.geomys.org/ct/{$}", HTMLHandler("uptime_ct.html"))
	mux.HandleFunc("uptime.geomys.org/ct/24h/{filter}", func(w http.ResponseWriter, r *http.Request) {
		filter := r.PathValue("filter")
		if filter == "" {
			msg := "missing filter in path"
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		threshold := 99.5
		if t := r.URL.Query().Get("threshold"); t != "" {
			parsed, err := strconv.ParseFloat(t, 64)
			if err != nil {
				msg := fmt.Sprintf("invalid threshold: %v", err)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			threshold = parsed
		}

		resp, err := uptimeClient.Get("https://www.gstatic.com/ct/compliance/endpoint_uptime_24h.csv")
		if err != nil {
			msg := fmt.Sprintf("error fetching data: %v", err)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			msg := fmt.Sprintf("error fetching data: status code %d", resp.StatusCode)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}

		var alerted bool
		output := &bytes.Buffer{}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			// https://tuscolo2025h2.sunlight.geomys.org/,add-chain,100.0000
			line := scanner.Text()
			fields := strings.Split(line, ",")
			if len(fields) != 3 {
				msg := fmt.Sprintf("error parsing data: invalid line %q", line)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
			if strings.Contains(fields[0], filter) {
				uptime, err := strconv.ParseFloat(fields[2], 64)
				if err != nil {
					msg := fmt.Sprintf("error parsing data: invalid line %q", line)
					http.Error(w, msg, http.StatusInternalServerError)
					return
				}
				if uptime < threshold {
					alerted = true
				}
				fmt.Fprintln(output, line)
			}
		}
		if err := scanner.Err(); err != nil {
			msg := fmt.Sprintf("error reading data: %v", err)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if alerted {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		w.Write(output.Bytes())
	})

	mux.Handle("uptime.geomys.org/witness/{$}", HTMLHandler("uptime_witness.html"))
	mux.HandleFunc("uptime.geomys.org/witness/log-list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "logs/v0")
		fmt.Fprintln(w, "vkey geomys.org/witness/test-log+c0787ff4+AeMb5VOzy60PTGdGmLPxOKGAa0jNyDGsgv2rnprGju1t")
		fmt.Fprintln(w, "qpd 86400")
		fmt.Fprintln(w, "contact https://uptime.geomys.org/witness/")
	})
	mux.HandleFunc("uptime.geomys.org/witness/add-checkpoint/{vkey...}", func(w http.ResponseWriter, r *http.Request) {
		vkey := r.PathValue("vkey")
		v, err := torchwood.NewCosignatureVerifier(vkey)
		if err != nil {
			msg := fmt.Sprintf("invalid vkey: %v", err)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		url := "https://" + v.Name() + "/add-checkpoint"
		resp, err := uptimeClient.Post(url, "text/plain", tlogWitnessBody(1))
		if err != nil {
			msg := fmt.Sprintf("error submitting checkpoint: %v", err)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}
		if resp.StatusCode == http.StatusConflict {
			// Might be the first time we submit to this witness, try again with
			// old size zero.
			resp.Body.Close()
			resp, err = uptimeClient.Post(url, "text/plain", tlogWitnessBody(0))
			if err != nil {
				msg := fmt.Sprintf("error submitting checkpoint: %v", err)
				http.Error(w, msg, http.StatusBadGateway)
				return
			}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			msg := fmt.Sprintf("error submitting checkpoint: status code %d", resp.StatusCode)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}
		sig, err := io.ReadAll(resp.Body)
		if err != nil {
			msg := fmt.Sprintf("error reading witness response: %v", err)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}
		n, err := note.Open(concatSig(sig), note.VerifierList(v))
		if err != nil {
			msg := fmt.Sprintf("invalid witness signature: %v", err)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}
		t, err := torchwood.CosignatureTimestamp(n.Sigs[0])
		if err != nil {
			msg := fmt.Sprintf("error parsing witness signature: %v", err)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}
		if time.Since(time.Unix(t, 0)) > 1*time.Minute {
			msg := fmt.Sprintf("stale witness signature: timestamp %d", t)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}
		fmt.Fprintf(w, "witness signature valid, timestamp %d\n", t)
	})
}

func tlogWitnessBody(old int64) io.Reader {
	return bytes.NewReader([]byte(fmt.Sprintf(`old %d

geomys.org/witness/test-log
1
BCml5C32yqMcl0gjTrcSOeNVx59oPnSdytBzDGBO5k0=

— geomys.org/witness/test-log wHh/9BPsoBNr2x0Ol3qPBYasIN0HI2ZiBg5ac0v3LQq/7F+YO7U4oWbDeJn1VaWVrlbSEM30Gr7WWYQjj2SBxRoJ/Ao=
`, old)))
}

func concatSig(sig []byte) []byte {
	return []byte(`geomys.org/witness/test-log
1
BCml5C32yqMcl0gjTrcSOeNVx59oPnSdytBzDGBO5k0=

` + string(sig))
}
