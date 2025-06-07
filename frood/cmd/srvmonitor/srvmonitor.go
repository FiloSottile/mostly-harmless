// Command srvmonitor runs an HTTP server that executes scripts in
// /etc/monitor.d on demand.
//
// index.sh is executed for the index page, and other scripts are executed
// for the corresponding paths.
//
// The client is not trusted, but scripts are.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"filippo.io/mostly-harmless/zfspasskey"
	"gopkg.in/yaml.v3"
)

const header = `
<!DOCTYPE html>
<style>
pre {
	font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro',
		Menlo, Consolas, 'DejaVu Sans Mono', monospace;
}
:root {
	color-scheme: light dark;
}
.container {
	max-width: 800px;
	margin: 100px auto;
}
</style>
<div class="container">
<pre>
`

func main() {
	configYAML, err := os.ReadFile("/etc/zfs/passwords.yaml")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	var passwords map[string]string
	if err := yaml.Unmarshal(configYAML, &passwords); err != nil {
		log.Fatalf("Failed to unmarshal config file: %v", err)
	}
	unlockHandler, err := zfspasskey.NewHandler(passwords)
	if err != nil {
		log.Fatalf("Failed to —create unlock handler: %v", err)
	}

	h := http.NewServeMux()
	h.HandleFunc("GET /{$}", index)
	h.HandleFunc("GET /{script}", script)
	h.Handle("/unlock/", http.StripPrefix("/unlock", unlockHandler))
	srv := &http.Server{Handler: h}
	if err := srv.ListenAndServeTLS("/etc/ssl/frood.pem", "/etc/ssl/frood-key.pem"); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, header)

	out, err := exec.CommandContext(r.Context(), "/etc/monitor.d/index.sh").CombinedOutput()
	w.Write(out)
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
	}
}

func script(w http.ResponseWriter, r *http.Request) {
	script := r.PathValue("script")
	if !filepath.IsLocal(script) {
		http.Error(w, "invalid script", http.StatusBadRequest)
		return
	}
	script = filepath.Join("/etc/monitor.d", script)
	if _, err := os.Stat(script); err != nil {
		http.Error(w, "script not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	out, err := exec.CommandContext(r.Context(), script).CombinedOutput()
	w.Write(out)
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
	}
}
