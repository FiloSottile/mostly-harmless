package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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

func download(archiveURL string) {
	zipURL := archiveURL + ".zip"
	tokens := strings.Split(zipURL, "/")
	fileName := tokens[len(tokens)-1]

	output, err := os.Create(fileName)
	if err != nil {
		log.Fatal("Error while creating", fileName, "-", err)
	}
	defer output.Close()

	for {
		resp, err := http.Get(zipURL)
		if err != nil {
			log.Fatal("Error while downloading", zipURL, "-", err)
		}
		if resp.StatusCode == 404 {
			resp.Body.Close()
			time.Sleep(1 * time.Second)
			continue
		}
		defer resp.Body.Close()

		_, err = io.Copy(output, resp.Body)
		if err != nil {
			log.Fatal("Error while downloading", zipURL, "-", err)
		}

		break
	}
}

func main() {
	usage := `archive.today client and bundler.

Usage: archive.today [--download] [<url>...]

The urls can be fed as command line arguments or on stdin,
separated by newlines.

By default, it only commits the urls to the archive, print
the archive.today url and exit.

Options:
  --download    Also wait and download the zip archive.
  -h --help     Show this screen.
  --version     Show version.`

	arguments, _ := docopt.Parse(usage, nil, true, "archive.today 0.1", false)

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

	for url := range urls {
		archiveURL := commit(url)
		fmt.Println(archiveURL)

		if arguments["--download"].(bool) {
			download(archiveURL)
		}
	}
}
