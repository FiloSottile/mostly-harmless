package main

import (
	"bytes"
	"encoding/xml"
	"errors"
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
		log.Fatal(err)
	}
	first, err := time.Parse("2006-01-02",
		strings.TrimSpace(string(bookmark)))
	if err != nil {
		log.Fatal(err)
	}
	r := &Rare{
		hc:   &http.Client{Timeout: 10 * time.Second},
		rate: time.NewTicker(time.Second),
	}
	r.fetchSitemaps(first)
}

const postsSitemap = "https://medium.com/sitemap/posts/%s/posts-%s.xml"

type Rare struct {
	hc   *http.Client
	rate *time.Ticker
}

func (r *Rare) fetchSitemaps(next time.Time) {
	for {
		if next.After(time.Now()) {
			return
		}
		r.fetchSitemap(next)
		next = next.AddDate(0, 0, 1)
		if err := ioutil.WriteFile("bookmark", []byte(next.Format("2006-01-02")), 0664); err != nil {
			log.Fatal(err)
		}
	}
}

func (r *Rare) fetchSitemap(t time.Time) {
	date := t.Format("2006-01-02")
	log.Printf("Fetching %s sitemap...", date)
	<-r.rate.C
	resp, err := r.hc.Get(fmt.Sprintf(postsSitemap, t.Format("2006"), date))
	if err != nil {
		log.Fatal(err)
	}
	buf := &bytes.Buffer{}
	f, err := createFile(fmt.Sprintf("sitemap/posts-%s.xml", date))
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(io.MultiWriter(buf, f), resp.Body); err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	var sitemap struct {
		URLs []string `xml:"url>loc"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &sitemap); err != nil {
		log.Fatal(err)
	}

	for _, url := range sitemap.URLs {
		if err := r.fetchPost(url); err != nil {
			log.Printf("error fetching post %q: %v", url, err)
		}
	}
}

func (r *Rare) fetchPost(url string) error {
	name := filepath.Clean(strings.TrimPrefix(url, "https://medium.com/"))
	dir, _ := filepath.Split(name)
	if dir == ".." || dir == "" || dir == "." {
		return errors.New("invalid url")
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
	f, err := createFile(prefix + "/" + name + ".html")
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
