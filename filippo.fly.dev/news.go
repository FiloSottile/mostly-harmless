package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"time"
)

//go:embed fakenews
var newsContent embed.FS

//go:embed hn.html
var hnTemplate string

func news(mux *http.ServeMux) {
	resultRe := regexp.MustCompile(`^ (.*)\((\d+) points, (\d+) comments\) $`)
	files := http.FileServer(http.FS(newsContent))
	htmlTmpl := template.Must(template.New("hn.html").Parse(hnTemplate))
	client := &http.Client{Timeout: 1 * time.Second}
	apiKey := os.Getenv("OPENAI_API_KEY")

	handleFuncWithCounter(mux, "filippo.io/fakenews/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fakenews/" {
			files.ServeHTTP(w, r)
			return
		}

		reqBody := map[string]interface{}{
			"model":       "ada:ft-personal:hntitles-2022-11-26-17-58-24",
			"prompt":      "A plausible Hacker News title:",
			"max_tokens":  50,
			"temperature": 0.9,
			"n":           15,
			"stop":        "END",
		}
		reqBytes := new(bytes.Buffer)
		json.NewEncoder(reqBytes).Encode(reqBody)

		req, err := http.NewRequestWithContext(r.Context(), "POST", "https://api.openai.com/v1/completions", reqBytes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()

		var result struct {
			Choices []struct {
				Text string
			}
			Usage struct {
				Total_tokens int
			}
		}
		err = json.NewDecoder(res.Body).Decode(&result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var entries []map[string]string
		for i, choice := range result.Choices {
			match := resultRe.FindStringSubmatch(choice.Text)
			if match == nil {
				continue
			}
			entries = append(entries, map[string]string{
				"Count":    fmt.Sprintf("%d", i+1),
				"Title":    match[1],
				"Points":   match[2],
				"Comments": match[3],
			})
		}

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		data := map[string]interface{}{
			"Cost":    fmt.Sprintf("0.%07d", result.Usage.Total_tokens*16),
			"Entries": entries,
		}
		htmlTmpl.Execute(w, data)
	})
}
