package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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

	return mux
}
