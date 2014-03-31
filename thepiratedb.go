package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
)

const runnersNum = 5
const maxTries = 3
const DEBUG = true

var notFoundText = []byte(`<title>Not Found | The Pirate Bay - The world's most resilient BitTorrent site</title>`)
var doctype = []byte(`<!DOCTYPE html PUBLIC`)

var LOG_INTERVAL = 10000
var START_OFFSET = 0

func runner(ci chan int, wg *sync.WaitGroup) {
	// Instantiate a client to keep a connection open
	client := &http.Client{}

	for i := range ci {
		if i%LOG_INTERVAL == 0 {
			log.Printf("Processing torrent %d", i)
		}

		tries := 0

	start:
		tries += 1
		if tries > maxTries {
			log.Printf("Failed torrent %d", i)
			continue
		}

		url := fmt.Sprintf("https://thepiratebay.se/torrent/%d", i)
		resp, err := client.Get(url)
		if err != nil {
			if DEBUG {
				log.Printf("Retry %d, %d-th time", i, tries)
			}
			goto start
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			if DEBUG {
				log.Printf("Retry torrent %d (%d)", i, tries)
			}
			goto start
		}
		resp.Body.Close()
		if !bytes.HasPrefix(body, doctype) {
			if DEBUG {
				log.Printf("Retry %d, %d-th time", i, tries)
			}
			goto start
		}

		if bytes.Index(body[:300], notFoundText) >= 0 {
			continue
		}

		log.Printf("Torrent number '%v' found", i)
	}

	log.Printf("Goroutine done.")
	wg.Done()
}

func main() {
	torrentLink := regexp.MustCompile(`<a href="/torrent/(\d+)/`)

	resp, err := http.Get("https://thepiratebay.se/recent")
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	latestMatch := torrentLink.FindSubmatch(body)
	resp.Body.Close()
	if latestMatch == nil {
		log.Fatal("latestMatch failed")
	}
	latest, _ := strconv.Atoi(string(latestMatch[1]))

	if DEBUG {
		log.Printf("Latest was %d", latest)
		latest = 100
		LOG_INTERVAL = 10
		START_OFFSET = 9850000
	}

	var wg sync.WaitGroup
	ci := make(chan int)
	for i := 0; i < runnersNum; i++ {
		wg.Add(1)
		go runner(ci, &wg)
	}
	for i := 1 + START_OFFSET; i <= latest+START_OFFSET; i++ {
		ci <- i
	}
	close(ci)
	wg.Wait()

	log.Print("Done.")
}
