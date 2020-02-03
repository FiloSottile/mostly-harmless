package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"expvar"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	bookmark, err := ioutil.ReadFile("bookmark")
	if err != nil {
		log.Fatalf("Failed to read bookmark: %v", err)
	}
	next, err := time.Parse("2006-01-02",
		strings.TrimSpace(string(bookmark)))
	if err != nil {
		log.Fatalf("Failed to parse bookmark: %v", err)
	}
	startInfluxDB()
	expvar.Publish("next", expvar.Func(func() interface{} {
		return next.Format("2006-01-02")
	}))
	r := &Rare{
		hc:      &http.Client{Timeout: 10 * time.Second},
		rate:    time.NewTicker(time.Second),
		fetches: expvar.NewInt("fetches"),
		errors:  expvar.NewInt("errors"),
	}
	for {
		time.Sleep(time.Until(next.Add(36 * time.Hour)))
		if err := r.fetchSitemap(next); err != nil {
			log.Printf("Error fetching sitemap %s: %v", next.Format("2006-01-02"), err)
			r.errors.Add(1)
			continue
		}
		next = next.AddDate(0, 0, 1)
		if err := ioutil.WriteFile("bookmark", []byte(next.Format("2006-01-02")), 0664); err != nil {
			log.Fatalf("Failed to write bookmark: %v", err)
		}
	}
}

const postsSitemap = "https://medium.com/sitemap/posts/%s/posts-%s.xml"

type Rare struct {
	hc   *http.Client
	rate *time.Ticker

	fetches *expvar.Int
	errors  *expvar.Int
}

func (r *Rare) fetchSitemap(t time.Time) error {
	date := t.Format("2006-01-02")
	log.Printf("Fetching %s sitemap...", date)
	<-r.rate.C
	resp, err := r.hc.Get(fmt.Sprintf(postsSitemap, t.Format("2006"), date))
	if err != nil {
		return err
	}
	buf := &bytes.Buffer{}
	f, err := createFile(fmt.Sprintf("sitemap/posts-%s.xml", date))
	if err != nil {
		return err
	}
	if _, err := io.Copy(io.MultiWriter(buf, f), resp.Body); err != nil {
		return err
	}
	resp.Body.Close()
	if err := f.Close(); err != nil {
		return err
	}

	var sitemap struct {
		URLs []string `xml:"url>loc"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &sitemap); err != nil {
		return err
	}

	for _, url := range sitemap.URLs {
		if err := r.fetchPost(url); err != nil {
			log.Printf("Error fetching post %q: %v", url, err)
			r.errors.Add(1)
			continue
		}
		r.fetches.Add(1)
	}
	return nil
}

func (r *Rare) fetchPost(url string) error {
	dir, file := filepath.Split(filepath.Clean(
		strings.TrimPrefix(url, "https://medium.com/")))
	if dir == ".." || dir == "" || dir == "." {
		return errors.New("invalid url")
	}
	if len(file) > 250 {
		parts := strings.Split(file, "-")
		file = parts[len(parts)-1]
	}

	<-r.rate.C
	resp, err := r.hc.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var prefix string
	if strings.HasPrefix(dir, "@") {
		prefix = dir[:min(len(dir), 3)]
	} else {
		prefix = dir[:min(len(dir), 2)]
	}

	f, err := createFile(prefix + "/" + dir + file + ".html")
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return f.Close()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func createFile(name string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(name), 0775); err != nil {
		return nil, err
	}
	return os.Create(name)
}
