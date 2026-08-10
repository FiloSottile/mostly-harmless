package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var bothConfigFiles = []string{
	"/etc/sunlight/sunlight.yaml",
	"/etc/sunlight/sunlight-staging.yaml",
	"/etc/sunlight/skylight.yaml",
	"/usr/local/bin/debug",
	"/etc/systemd/system/sunlight.service",
	"/etc/systemd/system/sunlight-staging.service",
	"/etc/systemd/system/skylight.service",
	"/etc/systemd/system/partial-aftersun.service",
	"/etc/systemd/system/partial-aftersun.timer",
	"/etc/systemd/system/partial-aftersun-staging.service",
	"/etc/systemd/system/partial-aftersun-staging.timer",
	"/etc/systemd/system/public-config.service",
	"/etc/logrotate.d/jsonl",
	"/etc/caddy/Caddyfile",
	"/etc/systemd/system/caddy.service.d/override.conf",
	"/usr/local/bin/zfs-textfile",
	"/etc/systemd/system/zfs-textfile.service",
	"/etc/systemd/system/zfs-textfile.timer",
}

var tuscoloConfigFiles = []string{
	"/etc/sunlight/sunlight-golf.yaml",
	"/etc/systemd/system/sunlight-golf.service",
	"/etc/systemd/system/heliograph-dashboard.service",
	"/etc/systemd/system/heliograph-dashboard.timer",
	"/etc/systemd/system/heliograph-dashboard-trastevere.service",
	"/etc/systemd/system/heliograph-dashboard-trastevere.timer",
	"/etc/systemd/system/age-keyserver.service",
	"/etc/systemd/system/pkg-geomys-dev.service",
	"/etc/systemd/system/plc-replica.service",
	"/etc/prometheus/prometheus.yml",
}

func main() {
	nameFlag := flag.String("name", "", "Name of the server to display configuration for (tuscolo or trastevere)")
	flag.Parse()
	var configFiles []string
	switch *nameFlag {
	case "tuscolo":
		configFiles = append(bothConfigFiles, tuscoloConfigFiles...)
	case "trastevere":
		configFiles = bothConfigFiles
	default:
		log.Fatalf("Unknown server name: %s", *nameFlag)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		type File struct {
			Path     string
			Contents string
		}
		files := make([]File, 0, len(configFiles))
		for _, path := range configFiles {
			contents, err := os.ReadFile(path)
			if err != nil {
				http.Error(w, "Error reading file: "+path, http.StatusInternalServerError)
				return
			}
			files = append(files, File{Path: path, Contents: string(contents)})
		}
		tmpl.Execute(w, struct {
			Files []File
			Name  string
		}{
			Files: files,
			Name:  strings.Title(*nameFlag),
		})
	})
	server := &http.Server{
		Addr:         "localhost:8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

var tmpl = template.Must(template.New("config").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="canonical" href="https://config.sunlight.geomys.org/">
    <title>Geomys {{ .Name }} CT Log Server Public Configuration</title>
    <style>
        :root {
            font-family: Avenir, Montserrat, Corbel, 'URW Gothic', source-sans-pro, sans-serif;
            color-scheme: light dark;
        }
        code, pre {
            font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, Consolas, 'DejaVu Sans Mono', monospace;
        }
        p {
            line-height: 1.5em;
        }
        a {
            color: inherit;
        }
        .container {
            width: auto;
            max-width: 700px;
            padding: 0 15px;
            margin: 80px auto;
        }
        img {
            max-width: 100%;
            height: auto;
        }
        @media print {
            .container {
                margin: 0;
                padding: 0;
            }
        }
    </style>
</head>
<body>
<div class="container">
    <h1>Geomys {{ .Name }} CT Log Server Public Configuration</h1>
    <p>
        This is the <em>live</em> public configuration for the
        <a href="https://groups.google.com/a/chromium.org/g/ct-policy/c/KCzYEIIZSxg/m/zD26fYw4AgAJ">
        Geomys {{ .Name }} CT Log Server</a>, a <a href="https://sunlight.dev/">Sunlight</a> instance.

    <p>
        See also the <a href="https://docs.google.com/document/d/1ID8dX5VuvvrgJrM0Re-jt6Wjhx1eZp-trbpSIYtOhRE/edit?tab=t.0#heading=h.v39dw5r67cif">
        public playbooks</a>.

    {{ range .Files }}
        <h2 id="{{ .Path }}"><a href="#{{ .Path }}">{{ .Path }}</a></h2>
        <pre><code>{{ .Contents }}</code></pre>
    {{ end }}
`))
