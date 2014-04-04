package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var DEBUG = os.Getenv("DEBUG") != ""

var notFoundText = []byte(`<title>Not Found | The Pirate Bay - The world's most resilient BitTorrent site</title>`)
var doctype = []byte(`<!DOCTYPE html PUBLIC`)

var LOG_INTERVAL = 10000

type Torrent struct {
	Id          int
	Title       string
	Category    string
	Size        int64
	Seeders     int
	Leechers    int
	Uploaded    time.Time
	Uploader    string
	Files_num   int
	Description string
	Magnet      string
}

var regexes = struct {
	title, category, size,
	seeders, leechers,
	uploaded, uploader,
	files_num, description,
	magnet *regexp.Regexp
}{
	regexp.MustCompile(`(?s)<div id="title">\s*(.+?)\s*</div>`),
	regexp.MustCompile(`<dt>Type:</dt>\s*<dd><a[^>]*>(.+?)</a></dd>`),
	regexp.MustCompile(`(?s)<dt>Size:</dt>.*?\((\d+)&nbsp;Bytes\)</dd>`),
	regexp.MustCompile(`(?s)<dt>Seeders:</dt>.*?(\d+)</dd>`),
	regexp.MustCompile(`(?s)<dt>Leechers:</dt>.*?(\d+)</dd>`),
	regexp.MustCompile(`<dt>Uploaded:</dt>\s*<dd>(.+?)</dd>`),
	regexp.MustCompile(`<dt>By:</dt>\s*<dd>\s*<[ai][^>]*>(.+?)</[ai]>`),
	regexp.MustCompile(`(?s)<dt>Files:</dt>\s*<dd>.+?(\d+)</a></dd>`),
	regexp.MustCompile(`(?s)<div class="nfo">\s*<pre>(.+?)</pre>`),
	regexp.MustCompile(`href="(magnet:.+?)" title="Get this torrent"`),
}

const sqlInit = `
CREATE TABLE "Torrents" (
"Id" INTEGER PRIMARY KEY,
"Title" TEXT,
"Category" TEXT,
"Size" INTEGER,
"Seeders" INTEGER,
"Leechers" INTEGER,
"Uploaded" TEXT,
"Uploader" TEXT,
"Files_num" INTEGER,
"Description" TEXT,
"Magnet" TEXT
);`
const sqlIndex = `
CREATE INDEX "TITLE" ON "Torrents" ("Title");`
const sqlInsert = `
INSERT INTO "Torrents" VALUES
(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

var stripTagsRegexp = regexp.MustCompile(`(?s)<.+?>`)

func stripTags(s string) string {
	return stripTagsRegexp.ReplaceAllLiteralString(s, "")
}

func ParseTorrent(data []byte, t *Torrent) error {
	var err error

	match := regexes.title.FindSubmatch(data)
	if match == nil {
		return errors.New("title not found")
	}
	t.Title = html.UnescapeString(string(match[1]))

	match = regexes.category.FindSubmatch(data)
	if match == nil {
		return errors.New("category not found")
	}
	t.Category = html.UnescapeString(string(match[1]))

	match = regexes.size.FindSubmatch(data)
	if match == nil {
		return errors.New("size not found")
	}
	t.Size, err = strconv.ParseInt(string(match[1]), 10, 64)
	if err != nil {
		return errors.New("size malformed")
	}

	match = regexes.seeders.FindSubmatch(data)
	if match == nil {
		return errors.New("seeders not found")
	}
	t.Seeders, err = strconv.Atoi(string(match[1]))
	if err != nil {
		return errors.New("seeders malformed")
	}

	match = regexes.leechers.FindSubmatch(data)
	if match == nil {
		return errors.New("leechers not found")
	}
	t.Leechers, err = strconv.Atoi(string(match[1]))
	if err != nil {
		return errors.New("leechers malformed")
	}

	match = regexes.uploaded.FindSubmatch(data)
	if match == nil {
		return errors.New("uploaded not found")
	}
	t.Uploaded, err = time.Parse("2006-01-02 15:04:05 MST", string(match[1]))
	if err != nil {
		return errors.New("uploaded malformed")
	}

	match = regexes.uploader.FindSubmatch(data)
	if match == nil {
		return errors.New("uploader not found")
	}
	t.Uploader = string(match[1])

	match = regexes.files_num.FindSubmatch(data)
	if match == nil {
		return errors.New("files_num not found")
	}
	t.Files_num, err = strconv.Atoi(string(match[1]))
	if err != nil {
		return errors.New("files_num malformed")
	}

	match = regexes.description.FindSubmatch(data)
	if match == nil {
		return errors.New("description not found")
	}
	t.Description = html.UnescapeString(stripTags(string(match[1])))

	match = regexes.magnet.FindSubmatch(data)
	if match == nil {
		return errors.New("magnet not found")
	}
	t.Magnet = string(match[1])

	return nil
}

func runner(ci chan int, dbChan chan *Torrent, maxTries int, wg *sync.WaitGroup) {
	// Instantiate a client to keep a connection open
	client := &http.Client{}

	for i := range ci {
		if i%LOG_INTERVAL == 0 {
			log.Printf("Processing torrent %d", i)
		}

		tries := 0

	retry:
		tries += 1
		if tries > maxTries {
			if DEBUG {
				log.Fatalf("Failed torrent %d", i)
			} else {
				log.Printf("Failed torrent %d", i)
			}
			continue
		}

		url := fmt.Sprintf("https://thepiratebay.se/torrent/%d", i)
		resp, err := client.Get(url)
		if err != nil {
			if DEBUG {
				log.Printf("Retry torrent %d (%d)", i, tries)
			}
			time.Sleep(time.Duration(tries) * time.Second)
			goto retry
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			if DEBUG {
				log.Printf("Retry torrent %d (%d)", i, tries)
			}
			time.Sleep(time.Duration(tries) * time.Second)
			goto retry
		}
		resp.Body.Close()
		if !bytes.HasPrefix(body, doctype) {
			if DEBUG {
				log.Printf("Retry torrent %d (%d)", i, tries)
			}
			time.Sleep(time.Duration(tries) * time.Second)
			goto retry
		}

		if bytes.Index(body[:300], notFoundText) >= 0 {
			continue
		}

		t := new(Torrent)
		t.Id = i
		err = ParseTorrent(body, t)
		if err != nil {
			if DEBUG {
				log.Fatal(i, err)
			} else {
				log.Printf("ERROR: torrent %d: %v", i, err)
			}
		}

		dbChan <- t

		// log.Printf("%+v", t)
	}

	log.Printf("Goroutine done.")
	wg.Done()
}

func getLatest() int {
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

	return latest
}

func openDb(new bool) (*sql.DB, *sql.Stmt) {
	if new {
		os.Remove("./thepirate.db")
	}

	db, err := sql.Open("sqlite3", "./thepirate.db")
	if err != nil {
		log.Fatal(err)
	}

	if new {
		_, err = db.Exec(sqlInit)
		if err != nil {
			log.Fatal(err)
		}
		_, err = db.Exec(sqlIndex)
		if err != nil {
			log.Fatal(err)
		}
	}

	insertQuery, err := db.Prepare(sqlInsert)
	if err != nil {
		log.Fatal(err)
	}

	return db, insertQuery
}

func parseArgs() (maxTries int, runnersNum int) {
	if flag.NArg() < 2 {
		flag.Usage()
	}

	runnersNum, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		flag.Usage()
	}

	maxTries, err = strconv.Atoi(flag.Arg(1))
	if err != nil {
		flag.Usage()
	}

	return
}

func writer(dbChan chan *Torrent, insertQuery *sql.Stmt, lock *sync.Mutex) {
	var err error

	for t := range dbChan {
		tries := 0

	retry:
		tries += 1
		if tries > 3 {
			if DEBUG {
				log.Fatal(t.Id, err)
			} else {
				log.Printf("ERROR: torrent %d: sql %v", t.Id, err)
			}
			continue
		}

		_, err = insertQuery.Exec(
			t.Id, t.Title, t.Category, t.Size,
			t.Seeders, t.Leechers, t.Uploaded, t.Uploader,
			t.Files_num, t.Description, t.Magnet,
		)
		if err != nil && err.Error() == "UNIQUE constraint failed: Torrents.Id" {
			continue
		}
		if err != nil {
			time.Sleep(time.Duration(tries) * time.Second)
			goto retry
		}
	}

	lock.Unlock()
}

func main() {
	// sqlite3 thepirate.db "SELECT Id FROM Torrents ORDER BY Id DESC LIMIT 1;"
	var startOffset = flag.Int("start", 0, "the starting torrent number")
	// egrep -v "(Processing torrent|sql UNIQUE constraint failed)" thepirate.log |
	// egrep "^2014/04" | egrep -o "[0-9]{5,}" | thepiratedb -log
	var fromLog = flag.Bool("log", false, "read the numbers from stdin")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] runnersNum maxTries\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()
	maxTries, runnersNum := parseArgs()

	db, insertQuery := openDb(*startOffset == 0 && !*fromLog)
	defer db.Close()
	latest := getLatest()

	if DEBUG {
		// log.Printf("Latest was %d", latest)
		// latest = 50000
		LOG_INTERVAL = 10
	}

	go func(db *sql.DB) {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		_ = <-c
		db.Close()
		os.Exit(0)
	}(db)

	writerLock := new(sync.Mutex)
	writerLock.Lock()
	dbChan := make(chan *Torrent)
	go writer(dbChan, insertQuery, writerLock)

	var wg sync.WaitGroup
	ci := make(chan int)
	for i := 0; i < runnersNum; i++ {
		wg.Add(1)
		go runner(ci, dbChan, maxTries, &wg)
	}

	if *fromLog {
		if DEBUG {
			log.Print("Reading from log...")
		}
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			i, err := strconv.Atoi(scanner.Text())
			if err != nil {
				log.Fatal("log malformed")
			}
			ci <- i
		}
		if err := scanner.Err(); err != nil {
			log.Fatal("reading standard input:", err)
		}
	} else {
		for i := 1 + *startOffset; i <= latest; i++ {
			ci <- i
		}
	}

	close(ci)
	wg.Wait()

	close(dbChan)
	writerLock.Lock()

	log.Print("Done.")
}
