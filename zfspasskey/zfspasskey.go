package zfspasskey

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

//go:embed zfspasskey.html
var webUITmpl string
var webUI = template.Must(template.New("main").Parse(webUITmpl))

//go:embed age-0.2.3-1-g59d2c94.js scure-base-1.2.5.js
var js embed.FS

// NewHandler returns a handler that serves a web interface for remotely
// unlocking ZFS datasets.
//
// The datasets map must have dataset names as keys, and armored age files
// containing the password as values. The age files can be generated using the
// web interface itself, but currently must be manually configured.
//
// Clients are sent the age file headers only, decrypt them using passkeys, and
// send back the file key. The handler decrypts the password and mounts the
// dataset with "zfs mount -l -R".
func NewHandler(datasets map[string]string) (http.Handler, error) {
	headers := make(map[string]string)
	for name, ageFile := range datasets {
		hdr, err := age.ExtractHeader(armor.NewReader(strings.NewReader(ageFile)))
		if err != nil {
			return nil, fmt.Errorf("failed to extract header for %s: %w", name, err)
		}
		headers[name] = base64.StdEncoding.EncodeToString(hdr)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(js))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		webUI.Execute(w, headers)
	})
	mux.HandleFunc("POST /{$}", func(w http.ResponseWriter, r *http.Request) {
		var res struct {
			FileKey []byte
			Name    string
		}
		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		ageFile, ok := datasets[res.Name]
		if !ok {
			http.Error(w, "file not found", 404)
			return
		}
		i := age.NewInjectedFileKeyIdentity(res.FileKey)
		ar, err := age.Decrypt(armor.NewReader(strings.NewReader(ageFile)), i)
		if err != nil {
			http.Error(w, err.Error(), 403)
			return
		}
		password, err := io.ReadAll(ar)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		cmd := exec.Command("zfs", "mount", "-l", "-R", res.Name)
		cmd.Stdin = bytes.NewReader(password)
		out, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to mount %s: %v", res.Name, err), 500)
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "mounted %s\n", res.Name)
		}
		w.Write(out)
	})
	return mux, nil
}
