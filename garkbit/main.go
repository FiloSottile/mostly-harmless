package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

const headerHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Door Control</title>
    <style>
        :root {
            font-family: Avenir, Montserrat, Corbel, 'URW Gothic', source-sans-pro, sans-serif;
            color-scheme: light dark;
        }

        body {
            margin: 0;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 1rem;
            gap: 2rem;
            text-align: center;
            padding-left: max(1rem, calc((100vw - 65ch) / 2));
            padding-right: max(1rem, calc((100vw - 65ch) / 2));
        }

        p {
            text-align: justify;
            max-width: 65ch;
            margin: 0 auto;
        }

        button {
            font-size: 1.5rem;
            padding: 1rem 2rem;
            border-radius: 0.5rem;
            border: 2px solid currentColor;
            background: transparent;
            color: currentColor;
            cursor: pointer;
        }

        a {
            text-decoration: none;
        }
    </style>
</head>
<body>
`

const homeHTML = `
    <form action="open" method="post">
        <button type="submit">🚪 ✨</button>
    </form>
    <p><i>"Ghastly," continued Marvin, "it all is. Absolutely ghastly. Just don't even talk about it. Look at this door," he said, stepping through it. The irony circuits cut into his voice modulator as he mimicked the style of the sales brochure. "All the doors in this spaceship have a cheerful and sunny disposition. It is their pleasure to open for you, and their satisfaction to close again with the knowledge of a job well done."</i> — The Hitchhiker's Guide to the Galaxy</p>
    </form>
`

const adminLinkHTML = `
    <a href="/admin">🦊</a>
`

const adminHTML = `
    <form action="newuser" method="post">
        <input type="text" name="username">
    </form>
`

type accessLogEntry struct {
	User string
	Time time.Time
}

type friend struct {
	Name string
	Key  string
}

type data struct {
	AccessLog []accessLogEntry
	Friends   []friend
}

func openDoor(pin rpio.Pin) {
	pin.High()
	defer pin.Low()
	time.Sleep(100 * time.Millisecond)
}

func main() {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		slog.Error("could not get user config dir", "err", err)
		return
	}
	dataPath := filepath.Join(dataDir, "garkbit.json")
	var dataStore data
	dataJSON, err := os.ReadFile(dataPath)
	if err == nil {
		err = json.Unmarshal(dataJSON, &dataStore)
	}
	slog.Info("db loaded", "path", dataPath, "err", err)

	// Can we have SQLite?
	// No, we have a database at home.
	// The database at home:
	var dbLock sync.RWMutex
	readData := func(f func(data *data)) {
		dbLock.RLock()
		defer dbLock.RUnlock()
		f(&dataStore)
	}
	writeData := func(f func(data *data)) {
		dbLock.Lock()
		defer dbLock.Unlock()
		f(&dataStore)
		dataJSON, err := json.Marshal(dataStore)
		if err != nil {
			slog.Error("could not marshal data", "err", err)
			return
		}
		err = os.WriteFile(dataPath, dataJSON, 0644)
		if err != nil {
			slog.Error("could not write data", "path", dataPath, "err", err)
		}
	}

	rpio.Open()
	defer rpio.Close()

	// https://pinout.xyz/pinout/pin11_gpio17/
	pin := rpio.Pin(17)
	pin.Output()
	pin.Low()

	home := func(w http.ResponseWriter, r *http.Request) {
		if authorized(r, readData) == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(headerHTML))
		w.Write([]byte(homeHTML))
		if isAdmin(r) {
			w.Write([]byte(adminLinkHTML))
		}
	}
	http.HandleFunc("GET /{$}", home)
	http.HandleFunc("GET /friend/{key}/", home)

	open := func(w http.ResponseWriter, r *http.Request) {
		u := authorized(r, readData)
		if u == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		openDoor(pin)
		slog.Info("door opened", "user", u)
		writeData(func(data *data) {
			data.AccessLog = slices.Insert(data.AccessLog, 0,
				accessLogEntry{User: u, Time: time.Now()})
		})
		http.Redirect(w, r, "", http.StatusSeeOther)
	}
	http.HandleFunc("POST /open", open)
	http.HandleFunc("POST /friend/{key}/open", open)

	http.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(headerHTML))
		w.Write([]byte(adminHTML))
		readData(func(data *data) {
			for _, friend := range data.Friends {
				fmt.Fprintf(w, `<p><a href="/friend/%s/">%s</a>`, friend.Key,
					html.EscapeString(friend.Name))
			}

			fmt.Fprintf(w, "<pre>")
			for _, entry := range data.AccessLog {
				fmt.Fprintf(w, "[%s] %s<br>", entry.Time.Format(time.RFC3339),
					html.EscapeString(entry.User))
			}
		})
	})

	http.HandleFunc("POST /newuser", func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		username := r.FormValue("username")
		if username == "" {
			http.Error(w, "No username", http.StatusBadRequest)
			return
		}
		key := rand.Text()
		writeData(func(data *data) {
			data.Friends = append(data.Friends, friend{
				Name: username, Key: key,
			})
		})
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})

	slog.Info("server starting", "addr", "localhost:3000")
	err = http.ListenAndServe("localhost:3000", nil)
	slog.Error("server stopped", "err", err)
}

func isAdmin(r *http.Request) bool {
	// adminUserHash is easy to reverse, but might save me some spam.
	const adminUserHash = "d6f5687af9ddd8ceb0921a85e4e392e2b7021e6f5210af36b3ce97733dac0f52"
	userHash := sha256.Sum256([]byte(r.Header.Get("Tailscale-User-Login")))
	return hex.EncodeToString(userHash[:]) == adminUserHash
}

func authorized(r *http.Request, readData func(f func(data *data))) string {
	var friends []friend
	readData(func(data *data) { friends = data.Friends })
	for _, friend := range friends {
		if r.PathValue("key") == friend.Key {
			return friend.Name
		}
	}

	// Anyone in the Tailnet or with thom this node was shared.
	// https://tailscale.com/kb/1312/serve#identity-headers
	return r.Header.Get("Tailscale-User-Login")
}
