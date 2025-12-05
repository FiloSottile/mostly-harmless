package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/gorilla/feeds"
)

const plcREADME = `# PLC Operations Log Atom Feed

The feed at https://at.geomys.org/plc/{id}.atom provides an Atom feed of the
PLC operations log for the given AT identifier (DID or handle).

This service fetches the PLC log from https://plc.directory and generates
the Atom feed on-the-fly.

Example:

	https://at.geomys.org/plc/filippo.abyssdomain.expert.atom

`

func at(mux *http.ServeMux) {
	mux.HandleFunc("GET at.geomys.org/plc/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(plcREADME))
	})
	mux.Handle("GET at.geomys.org/plc/{id}", PLCAtomHandler())
}

func PLCAtomHandler() http.Handler {
	client := &http.Client{Timeout: 5 * time.Second}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		s := r.PathValue("id")

		s, ok := strings.CutSuffix(s, ".atom")
		if !ok {
			http.Error(w, "only .atom URLs are supported", http.StatusNotFound)
			return
		}

		id, err := syntax.ParseAtIdentifier(s)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid AT identifier: %v", err), http.StatusBadRequest)
			return
		}
		var did syntax.DID
		if id.IsDID() {
			did, err = id.AsDID()
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid DID: %v", err), http.StatusBadRequest)
				return
			}
		} else {
			hdl, err := id.AsHandle()
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid handle: %v", err), http.StatusBadRequest)
				return
			}
			did, err = (&identity.BaseDirectory{}).ResolveHandle(ctx, hdl)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to resolve handle: %v", err), http.StatusBadRequest)
				return
			}
		}

		if did.Method() != "plc" {
			http.Error(w, "only plc DIDs are supported", http.StatusBadRequest)
			return
		}

		url := fmt.Sprintf("https://plc.directory/%s/log", did)
		resp, err := client.Get(url)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to fetch PLC entry: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			http.Error(w, fmt.Sprintf("failed to fetch PLC entry: status %d", resp.StatusCode), http.StatusBadGateway)
			return
		}
		var ops []json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&ops); err != nil {
			http.Error(w, fmt.Sprintf("failed to decode PLC entry: %v", err), http.StatusBadGateway)
			return
		}

		f := &feeds.Feed{
			Title: fmt.Sprintf("PLC Log for %s", did),
		}
		for i, opRaw := range ops {
			indented, err := indentJSON(opRaw)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to indent PLC operation: %v", err), http.StatusBadGateway)
				return
			}
			f.Add(&feeds.Item{
				Id:          fmt.Sprintf("%s:%d", did, i),
				IsPermaLink: "false",
				Title:       "PLC operation",
				Content:     fmt.Sprintf("<pre>%s</pre>", html.EscapeString(string(indented))),
			})
		}
		atom, err := f.ToAtom()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to generate Atom feed: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		w.Write([]byte(atom))
	})
}

func indentJSON(b json.RawMessage) ([]byte, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "    ")
}
