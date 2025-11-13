package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"filippo.io/torchwood"
	"golang.org/x/mod/sumdb/note"
)

const ctREADME = `# Certificate Transparency Uptime Alerts

This is a little service that checks

	https://www.gstatic.com/ct/compliance/endpoint_uptime.csv

and makes it possible to set up alerting for it.

e.g. https://ct-uptime-alerts.fly.dev/geomys.org will return a 503 if any lines
matching "geomys.org" have an uptime column below 99.95.

You can use it with any filter string, and it also takes a parameter like
"?threshold=99.8". You're welcome to use our instance (no guarantees!), or you
can run your own:

	https://github.com/FiloSottile/mostly-harmless/tree/main/ct-uptime-alerts

There is also a witness monitoring service at https://ct-uptime-alerts.fly.dev/witness/.
`

const witnessREADME = `# Witness Uptime Monitoring

This is a little service that submits the following checkpoint to a witness

	geomys.org/witness/test-log
	1
	BCml5C32yqMcl0gjTrcSOeNVx59oPnSdytBzDGBO5k0=

	— geomys.org/witness/test-log wHh/9BPsoBNr2x0Ol3qPBYasIN0HI2ZiBg5ac0v3LQq/7F+YO7U4oWbDeJn1VaWVrlbSEM30Gr7WWYQjj2SBxRoJ/Ao=

and checks that the witness responds with a fresh, valid signature.

The witness can be configured with the following log list

	https://ct-uptime-alerts.fly.dev/witness/log-list

or directly with this log vkey

	geomys.org/witness/test-log+c0787ff4+AeMb5VOzy60PTGdGmLPxOKGAa0jNyDGsgv2rnprGju1t

so it will accept the checkpoint.

To run a check, request /witness/add-checkpoint/ followed by the witness vkey, e.g.

	https://ct-uptime-alerts.fly.dev/witness/add-checkpoint/witness.navigli.sunlight.geomys.org+a3e00fe2+BNy/co4C1Hn1p+INwJrfUlgz7W55dSZReusH/GhUhJ/G

The vkey name must be the submission prefix of the witness.
`

func main() {
	s := &http.Server{
		Addr:         ":8080",
		Handler:      handler(),
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		IdleTimeout:  10 * time.Minute,
	}
	log.Fatal(s.ListenAndServe())
}

var client = &http.Client{
	Timeout: 5 * time.Second,
}

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, ctREADME)
	})
	mux.HandleFunc("/{filter}", func(w http.ResponseWriter, r *http.Request) {
		filter := r.PathValue("filter")
		if filter == "" {
			msg := "missing filter in path"
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		threshold := 99.95
		if t := r.URL.Query().Get("threshold"); t != "" {
			parsed, err := strconv.ParseFloat(t, 64)
			if err != nil {
				msg := fmt.Sprintf("invalid threshold: %v", err)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			threshold = parsed
		}

		resp, err := client.Get("https://www.gstatic.com/ct/compliance/endpoint_uptime.csv")
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

		if alerted {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(output.Bytes())
	})
	mux.HandleFunc("/witness/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, witnessREADME)
	})
	mux.HandleFunc("/witness/log-list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "logs/v0")
		fmt.Fprintln(w, "vkey geomys.org/witness/test-log+c0787ff4+AeMb5VOzy60PTGdGmLPxOKGAa0jNyDGsgv2rnprGju1t")
		fmt.Fprintln(w, "qpd 86400")
		fmt.Fprintln(w, "contact https://ct-uptime-alerts.fly.dev/witness/")
	})
	mux.HandleFunc("/witness/add-checkpoint/{vkey...}", func(w http.ResponseWriter, r *http.Request) {
		vkey := r.PathValue("vkey")
		v, err := torchwood.NewCosignatureVerifier(vkey)
		if err != nil {
			msg := fmt.Sprintf("invalid vkey: %v", err)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		url := "https://" + v.Name() + "/add-checkpoint"
		resp, err := client.Post(url, "text/plain", tlogWitnessBody(1))
		if err != nil {
			msg := fmt.Sprintf("error submitting checkpoint: %v", err)
			http.Error(w, msg, http.StatusBadGateway)
			return
		}
		if resp.StatusCode == http.StatusConflict {
			// Might be the first time we submit to this witness, try again with
			// old size zero.
			resp.Body.Close()
			resp, err = client.Post(url, "text/plain", tlogWitnessBody(0))
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

	return mux
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
