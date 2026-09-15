package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
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

		var queryThreshold float64
		if t := r.URL.Query().Get("threshold"); t != "" {
			parsed, err := strconv.ParseFloat(t, 64)
			if err != nil {
				msg := fmt.Sprintf("invalid threshold: %v", err)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			queryThreshold = parsed
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
				var threshold float64
				switch {
				case queryThreshold != 0:
					threshold = queryThreshold
				case fields[1] == "add-chain", fields[1] == "add-pre-chain":
					threshold = 95
				default:
					threshold = 99
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
	mux.HandleFunc("uptime.geomys.org/ct/add-pre-chain/{log...}", ctAddPreChain)

	mux.Handle("uptime.geomys.org/witness/{$}", witnessHomeHandler())
	mux.HandleFunc("uptime.geomys.org/witness/log-list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "logs/v0")
		fmt.Fprintln(w, "vkey geomys.org/witness/test-log+c0787ff4+AeMb5VOzy60PTGdGmLPxOKGAa0jNyDGsgv2rnprGju1t")
		fmt.Fprintln(w, "qpd 86400")
		fmt.Fprintln(w, "contact https://uptime.geomys.org/witness/")
	})
	mux.HandleFunc("uptime.geomys.org/witness/add-checkpoint/{vkey...}", func(w http.ResponseWriter, r *http.Request) {
		// The path is either a full vkey (name+hash+key) or a short reference
		// (name+hash), which is looked up in knownVKeys. If it's not there,
		// the cosignature is checked for freshness without being verified.
		ref := r.PathValue("vkey")
		var name string
		var hash uint32
		var v *torchwood.CosignatureVerifier
		if strings.Count(ref, "+") >= 2 {
			var err error
			v, err = torchwood.NewCosignatureVerifier(ref)
			if err != nil {
				msg := fmt.Sprintf("invalid vkey: %v", err)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			name, hash = v.Name(), v.KeyHash()
		} else {
			n, h, ok := strings.Cut(ref, "+")
			hh, err := strconv.ParseUint(h, 16, 32)
			if !ok || n == "" || len(h) != 8 || err != nil {
				msg := "invalid witness reference: expected name+hash or vkey"
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			name, hash = n, uint32(hh)
			v = knownVerifiers[ref]
		}

		url := "https://" + name + "/add-checkpoint"
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

		var cosig note.Signature
		if v != nil {
			n, err := note.Open(concatSig(sig), note.VerifierList(v))
			if err != nil {
				msg := fmt.Sprintf("invalid witness signature: %v", err)
				http.Error(w, msg, http.StatusBadGateway)
				return
			}
			cosig = n.Sigs[0]
		} else {
			var n *note.Note
			_, err := note.Open(concatSig(sig), note.VerifierList())
			var unverified *note.UnverifiedNoteError
			if errors.As(err, &unverified) {
				n = unverified.Note
			} else {
				msg := fmt.Sprintf("invalid witness response: %v", err)
				http.Error(w, msg, http.StatusBadGateway)
				return
			}
			var found bool
			for _, s := range n.UnverifiedSigs {
				if s.Name == name && s.Hash == hash {
					cosig, found = s, true
					break
				}
			}
			if !found {
				msg := fmt.Sprintf("witness response has no signature by %s+%08x", name, hash)
				http.Error(w, msg, http.StatusBadGateway)
				return
			}
		}

		t, err := torchwood.CosignatureTimestamp(cosig)
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
		if v != nil {
			fmt.Fprintf(w, "witness signature valid, timestamp %d\n", t)
		} else {
			fmt.Fprintf(w, "witness signature present but NOT verified (unknown key), timestamp %d\n", t)
		}
	})
}

// knownVKeys are the vkeys that can be referenced by name and key hash in
// the add-checkpoint URL, e.g. witness.example.com+a3e00fe2, without the key
// itself. This keeps the URL short enough for uptime monitoring services even
// with ML-DSA keys.
var knownVKeys = []string{
	"witness.navigli.sunlight.geomys.org+6bc44249+BvKbJzZZb2L5kdgqACS68/hxJoQQ88Zhp7esVL81HYRPQAurY9cfky201kuNm1oSO71BiNPWC8oNHunBo/PnRD/eWIuu/ZPIsiG0Cq9xg8dJsoKuQGXJ2Hu1QLBoyFWUFWAtmURCRi5bXFYJjVy4fFFh5ClS477HTy7av3C6VvnOdYTH1jISdXyIblBTWkQEq24gMc2JZKiS5YleWbXjlVuDKaU3yQjDhyWTd8AZuCvrcC9rE8wOZ7etpkAEf2pPINQ4LnMEHCWTuEE5NZwGWwqcekXvNQvv1jjfWQkhSzXbFvbLmifpWxRFrHn9OSEPGfaCksRUvKoTZzOrIPuFAhAraDIXX0DLwc7xY9hcg2lGpCaDoOXoyAS6m4wGiqerd0kLjFitkgJ7A/tpIcO+m255eyWnZwrevWaFgx+xwN7OjdJTlGMK9Qgd7QJW1Qilmt/h1SqZGpithCfje+/MiUea2eyTKbPBCf5c206/5Axu789uL1v22XZRO5tfDPLpU5lTPYdD0dt9M4I02G/2mHsZ43dbcsSYe0i+oM6GgIV0CoTx833GJMCjOOZbo3Qmlxwf3QFhkwGtc/db16iDfCW0U2NINxuSyDGahEIudOyNldfh+oe3tLyICiNGF3kGRQ1MRLD+e6AakJsjat9wMgjORHWx1rE7r14wCZtxFkSjOjSMpN8DiBXtGySSnWv2jKWnYnLPYhdqTJMEAZLLtd5Elaq+DukrJfmR5Jx348lx0E8AjALfg1oYQSCBKYlUN7h4kJhZhYGu6ahVPp1OBlA+DuBfoMtE6IiTfm11LseSOyNOtNTFe5V508Dw/APhoKuxy0mTAmEDmPnXTBvHhX1PVhl1qHTmfNhpukzxBRPKzsVN7y/FuJuqrX208k7SihRJGtDsXkQK/cFzTmSGvyHEKK+IHoKFpRK0dxEV6muqPJEEzehwZrJ7Rz0ZfkjuKVCiP98md+yD+r7GD2BjIb3IEzhrsbf5Ak7f5NImP4UoL6vy48ZfwmEo6tvn6StSVd912GYobRKHzboTBhUIn8c93VIX7PAQ64exm4l9wUBr3KMzHuykTW9m7C1xw1+Oqf3PGWN9rLHXNMPaR08xnImkTd+W74ls1sHH1CzJiNJB5QZg44zCajgL131L9G62AuTth/zJeGovddDU1m7N3i+e2Qz/omDKILlfUtcwPyyWeSvW/XX0ECFp/o9cDh/l6S5mcaT2Mtiiyn+owXl4Mx85Yppi2ZvVpZudLyhM0AGeoDdRRlfdeaJsvfC7pzBKzbj/1CKs7Ts962kvpCoDyZ5rMBK5PhFLvNXhl9ntKSTIAqUoM3vPgouzrA57Y9G5cWEJ741G4siABCFedWcDKT5zXKO682fncspp2MLjGCqNEWbSRRJ957q/W5mUyI1sDiXsKaaxGq0Xg86qu1AaEd7EBgOLj3i762ANsbZqtButkOdGD+MVeELpgNiOttWPW6Xq0ERM5NcJgezQJghy+y7IKf6zqrud4Ql6pklvjTuzCeLVsfpWQsqfgF4W0ZADldLfPCJIDBby1sxjh32i6hIp2R7QC6Mp87k9PAa5fiFDOqJXqDtGLxKPKu6UPdvbJHfHx4M37F6mfQnst4Bkjgf0kLGkW9isZ7bnids+QclIgfKFjZ5mD+ekGKo835+G44y7zAauKZzj8nBiNMSYwdAEMn6TYHciHn1VyetNEtzSMHInsV30TOnahKfkNgvAuNCuxrIyNvFHCw9t3vRr/J0=",
	"witness-golf.sunlight.geomys.org+285a8d9f+BmXb1ItjzDOpqzbdlBDvu5HD+lM3cZMMdPpnBsskhj99lnA4KDncLD5ck/YXkOgNjani4pFnX4PFKtM5WefU/+3TQv4qyOpXtUVLsExml5qdmiqwaMYb7Oz8WPntgJ+PWLECCOQWhMdgvj50O2MdbaSHZI74MrE2ARVcxalUf16awdvwZD8qLVgZATxMP56NtEmbRmRIHrakSzHLL93iM1RElqjoBW7njk6sBgUpHrY6ljMXRvrL2rLBlZj7zZ6pz6NoKR5fYhw63YAPLTBwf4m2YarWtcPZ7gNHN/4n9I5LheySxcu47cKsOdzKCIsa9+sXbZUEbKW1CBu8Tl3nd/MuPKC/5F3zccaUFURZlqgnS1PKOqeCc1xuXnnUOmulYJbMPosYMmSuxqbuO56q+haO4ZsYYlBeeslN73EoKW2iF7CDxtQCWwBrnKn2WZxf5/3EVTweL4kYXZ+G4DmJkfGHlyUJj/dJRi0V1rj/F8sFLwZNRc/LoNBqhY+CJw5+KdItNVBM3RTNQFoGwAa9oTFAIfBJA2qjlJQ1L1NxxfD1G6MwUvOkDeZpghyVU1fDiyeKdsfxtWBlOeJYd64v2el/Z2CqegQaiWTw5rbWz7EPxNs0f/z2Qg7V8x9warJW7o0ta1YR4MvIUKJy2aIy3kE/94qBF2ai95j9wkazIPxZJjBVjKz+mHGsRw5yH6zC22xD4NLulD9IBWGsbpPL5WGLfVdcYyjelLLdFcwNogAwr25bQHRrW51WCdfxfJp0XubgaXKhV92LyQ5YEwe7YDMi+aOZBaI73Y6Tg7oxsXRTyUBUR6vU0vtkfrWI1PkI35sdK7bDBbZNsPQ0tbtM4MhJIcwUzv8sPnrZMpu7w+LVgvAxhIbd49ITjRz0JA8NomcHX0zCvES/hVtAZmQoH2K4cYTrCX9wbg9jx9Vz02cX5szL6UIf6W0fOI4h8cjTsI1lw/U9rDJxmIfVIatkvLkmJHqzOBfDFe7nY7ASrnMTqr3Ei08eiRVvLgLkqgWyekb1x4GnaOE1OrumCrP1b9dPDQF9XUPZWHRqq4S0lhlGVs0FnykXTRxxF0tF2NrR+66J9z4HFX5PX+4MiwmX/E0Bs6TM8Yr2/loVF4ipzXsKkDI1wfid5i8Xx3PRBtbhFUQ87Oytjyu5L1cdXbhy/iDFBUN5bJ2WEubbKm6f35NAuXdW355B2p6iksgqacARu25vTHDIL26AVT80J18CrJ0Loq1Q3IYIPGWSlS5Eg2mOM2M4JGIPPJEv0+QJ/t+y5+hrjlJNiDbdwtJIUIMv/Gk+NQb2arRahhDzrraqG7VK7yme6B3u3CeNwyrgrfbQXSE4Ikq8HevZcUKuwHWJmeHdjxB1fDVWzDYJoXzLxiXL95iE9esi65I/eFaeL43Rw6Uxu0draErdZTuLnMbRKDw0U4rab5oJIf/AboJmeu7xLCAXP0I9c++S8Ko0djwdsJHRovP9pbRA0zC/Q1Km9gaG5tuzmn+CAH0FuK0Egpx6sIXNpC/hMFDz2Oq/F6mqtxel6DYDfoA6Eim3jqfjtDQfrpCPVrbiA4Cj/zuI2Alb5EJDnWWXY8eunVAJz+P9h0F4ththRna1mvpmh7k00R7UzKfcZdClh9nE65T21RUloSfwV6k/A3Gz7+uUKMr8f6tjfi66DOSxZAsquwG7yoiaGxY7dP3o8bZeXQLcXq3bvO9/LtS6PshWNi/WQX40rAWLTKU1XbjtibbqwudbB6k=",
}

// knownVerifiers is knownVKeys indexed by "name+hash".
var knownVerifiers = func() map[string]*torchwood.CosignatureVerifier {
	m := make(map[string]*torchwood.CosignatureVerifier)
	for _, vkey := range knownVKeys {
		v, err := torchwood.NewCosignatureVerifier(vkey)
		if err != nil {
			panic(fmt.Sprintf("invalid known vkey %q: %v", vkey, err))
		}
		m[fmt.Sprintf("%s+%08x", v.Name(), v.KeyHash())] = v
	}
	return m
}()

// witnessHomeHandler serves uptime_witness.html with the list of known
// vkeys in place of the <!-- known-vkeys --> placeholder.
func witnessHomeHandler() http.Handler {
	content, err := htmlContent.ReadFile("uptime_witness.html")
	if err != nil {
		log.Printf("Failed to read HTML file %q: %v", "uptime_witness.html", err)
		return http.NotFoundHandler()
	}
	list := &bytes.Buffer{}
	list.WriteString("<pre><code>")
	for _, vkey := range knownVKeys {
		list.WriteString(html.EscapeString(vkey))
		list.WriteString("\n")
	}
	list.WriteString("</code></pre>")
	content = bytes.Replace(content, []byte("<!-- known-vkeys -->"), list.Bytes(), 1)
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/html; charset=UTF-8")
		rw.Write(content)
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
