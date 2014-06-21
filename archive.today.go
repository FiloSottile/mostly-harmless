package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
)

func commit(target string) string {
	resp, err := http.PostForm("https://archive.today/submit/",
		url.Values{"url": {target}})
	if err != nil {
		log.Fatal("Error doing a POST:", err)
	}
	resp.Body.Close()

	h := resp.Header.Get("Refresh")
	if h[:6] != "0;url=" {
		log.Fatal("Malformed answer while committing.")
	}

	return h[6:]
}

var loadingGif = []byte("https://archive.today/loading.gif")

func fetchZip(archiveURL string) (int64, io.ReadCloser, error) {
	zipURL := archiveURL + ".zip"

	for {
		resp, err := http.Get(archiveURL)
		if err != nil {
			log.Fatal("Error while checking", zipURL, "-", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()
		if bytes.Index(body, loadingGif) > -1 {
			time.Sleep(1 * time.Second)
			continue
		}

		resp, err = http.Get(zipURL)
		if err != nil {
			log.Fatal("Error while downloading", zipURL, "-", err)
		}
		if resp.StatusCode == 404 {
			resp.Body.Close()
			return 0, nil, errors.New("archive.today error")
		}

		size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			log.Fatal(err)
		}

		return size, resp.Body, nil
	}
}

func download(archiveURL string) {
	tokens := strings.Split(archiveURL, "/")
	fileName := tokens[len(tokens)-1] + ".zip"

	output, err := os.Create(fileName)
	if err != nil {
		log.Fatal("Error while creating", fileName, "-", err)
	}
	defer output.Close()

	_, respBody, err := fetchZip(archiveURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "archive.today failed to create an archive - check %s\n", archiveURL)
		return
	}
	defer respBody.Close()

	_, err = io.Copy(output, respBody)
	if err != nil {
		log.Fatal("Error while downloading", archiveURL, "-", err)
	}
}

func bundle(archiveURLs [][2]string) {
	type Site struct {
		Id  string
		Url string
	}
	type Context struct {
		Timestamp string
		Sites     []*Site
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")

	context := Context{
		Timestamp: timestamp,
	}

	output, err := os.Create("archive.today " + timestamp + ".tar")
	if err != nil {
		log.Fatal("Error while creating the tarball:", err)
	}
	defer output.Close()

	tw := tar.NewWriter(output)

	for _, URLs := range archiveURLs {
		archiveURL := URLs[0]
		originalURL := URLs[1]

		tokens := strings.Split(archiveURL, "/")
		id := tokens[len(tokens)-1]

		size, respBody, err := fetchZip(archiveURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "archive.today failed to create an archive - check %s\n", archiveURL)
			continue
		}

		body, err := ioutil.ReadAll(respBody)
		if err != nil {
			log.Fatal(err)
		}

		respBody.Close()

		r, err := zip.NewReader(bytes.NewReader(body), size)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range r.File {
			hdr := &tar.Header{
				Name:    id + "/" + f.Name,
				Size:    int64(f.UncompressedSize64),
				Mode:    0664,
				Uid:     1000,
				Gid:     1000,
				ModTime: time.Now(),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				log.Fatal(err)
			}
			rc, err := f.Open()
			if err != nil {
				log.Fatal(err)
			}
			_, err = io.Copy(tw, rc)
			if err != nil {
				log.Fatal(err)
			}
			rc.Close()
		}

		context.Sites = append(context.Sites, &Site{
			Id:  id,
			Url: originalURL,
		})
	}

	var index bytes.Buffer
	err = indexHTML.Execute(&index, context)
	if err != nil {
		log.Fatalln(err)
	}

	hdr := &tar.Header{
		Name:    "index.html",
		Size:    int64(index.Len()),
		Mode:    0664,
		Uid:     1000,
		Gid:     1000,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		log.Fatal(err)
	}
	_, err = index.WriteTo(tw)
	if err != nil {
		log.Fatalln(err)
	}

	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	usage := `archive.today client and bundler.

Usage: archive.today [--download|--bundle] [<url>...]

The urls can be fed as command line arguments or on stdin,
separated by newlines.

By default, it only commits the urls to the archive, print
the archive.today url and exit.

Options:
  --download    Also wait and download the zip archive.
  --bundle      Merge all the zip archives in a tarball with
                a HTML index.
  -h --help     Show this screen.
  --version     Show version.`

	arguments, _ := docopt.Parse(usage, nil, true, "archive.today 0.2", false)

	urls := make(chan string)
	go func() {
		cmdUrls := arguments["<url>"].([]string)
		if len(cmdUrls) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				urls <- scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				log.Fatal("Error reading standard input:", err)
			}
			close(urls)
		} else {
			for _, url := range cmdUrls {
				urls <- url
			}
			close(urls)
		}
	}()

	var toBundle [][2]string
	for url := range urls {
		archiveURL := commit(url)
		fmt.Println(archiveURL)

		if arguments["--download"].(bool) {
			download(archiveURL)
		}

		if arguments["--bundle"].(bool) {
			toBundle = append(toBundle, [2]string{archiveURL, url})
		}
	}

	if arguments["--bundle"].(bool) {
		bundle(toBundle)
	}
}
