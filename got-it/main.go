package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type entry struct {
	type_     string
	timestamp time.Time
}

var stateMu sync.Mutex
var state = make(map[string]entry)

func main() {
	serv := http.Server{
		Addr: os.Args[1],
	}

	http.HandleFunc("/submit", submit)
	http.HandleFunc("/stats", stats)
	http.HandleFunc("/speaker", serveHTML("speaker.html"))
	http.HandleFunc("/", serveHTML("index.html"))

	log.Fatal(serv.ListenAndServe())
}

func serveHTML(filename string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		contents, err := ioutil.ReadFile(filename)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(contents)
	}
}

func stats(w http.ResponseWriter, r *http.Request) {
	stateMu.Lock()
	defer stateMu.Unlock()

	gotit, confused := 0, 0
	for user := range state {
		if state[user].timestamp.Before(time.Now().Add(-time.Minute)) {
			continue
		}
		if state[user].type_ == "gotit" {
			gotit++
		} else {
			confused++
		}
	}

	json.NewEncoder(w).Encode(struct {
		Gotit    int `json:"gotit"`
		Confused int `json:"confused"`
	}{
		Gotit:    gotit,
		Confused: confused,
	})
}

func submit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	type_ := r.Form.Get("type")
	if type_ != "gotit" && type_ != "confused" {
		http.Error(w, "WRONG TYPE YO", 400)
		return
	}
	user := r.Form.Get("user")
	if user == "" {
		http.Error(w, "NO USER ID OH NO", 400)
		return
	}
	log.Printf("%s -> %s", user, type_)
	stateMu.Lock()
	defer stateMu.Unlock()
	state[user] = entry{
		type_:     type_,
		timestamp: time.Now(),
	}
}
