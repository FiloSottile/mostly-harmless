package main

import (
	"log"
	"net/http"
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
		Addr: ":8080",
	}

	http.HandleFunc("/submit", submit)

	log.Fatal(serv.ListenAndServe())
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
	log.Printf("received vote %s from %s", type_, user)
	stateMu.Lock()
	defer stateMu.Unlock()
	state[user] = entry{
		type_:     type_,
		timestamp: time.Now(),
	}
}
