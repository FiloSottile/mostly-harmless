package main

import (
	"bytes"
	"html/template"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"rsc.io/markdown"
)

var htmlPrefixTemplate = template.Must(template.New("md2html").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="canonical" href="{{ .Canonical }}">
    <title>{{ .Title }}</title>
    <style>
        :root {
            font-family: Avenir, Montserrat, Corbel, 'URW Gothic', source-sans-pro, sans-serif;
            color-scheme: light dark;
        }
        code, pre {
            font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, Consolas, 'DejaVu Sans Mono', monospace;
            -webkit-font-smoothing: antialiased;
        }
        sub, sup {
            line-height: 0;
        }
        p, li {
            line-height: 1.8em;
        }
        a {
            color: inherit;
        }
        header {
            margin: 4rem auto;
            max-width: 400px;
            padding: 0 10px;
        }
        main {
            width: auto;
            max-width: 700px;
            padding: 0 15px;
            margin: 5rem auto;
        }
        pre {
            padding: 1em;
            overflow-x: auto;
            color: Canvas;
            background-color: CanvasText;
            font-size: 0.9em;
            line-height: 1.5em;
        }
		img {
			max-width: 100%;
			height: auto;
            margin: 0 auto;
            display: block;
		}
        li {
            margin-top: 1em;
            margin-bottom: 1em;
        }
		@media print {
			main {
				margin: 0;
				padding: 0;
			}
		}
    </style>
</head>
<body>

<header>
	{{ if .Header.Link }}<a href="{{ .Header.Link }}">{{ end }}<picture>
        <source srcset="{{ .Header.Dark }}" media="(prefers-color-scheme: dark)">
        <img src="{{ .Header.Light }}" alt="{{ .Header.Alt }}">
    </picture>{{ if .Header.Link }}</a>{{ end }}
</header>

<main>
`))

// toHTML converts Markdown to HTML.
func toHTML(md []byte) string {
	var p markdown.Parser
	p.Table = true
	p.HeadingID = true
	p.Strikethrough = true
	p.SmartDot = true
	p.SmartDash = true
	return markdown.ToHTML(p.Parse(string(md)))
}

type FrontMatter struct {
	Title     string
	Canonical string
	Header    struct {
		Link  string
		Light string
		Dark  string
		Alt   string
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: go run ./md2html <input.md>\n")
	}
	inputFile := os.Args[1]
	if !strings.HasSuffix(strings.ToLower(inputFile), ".md") {
		log.Fatalf("Error: input file must have a .md extension: %s", inputFile)
	}

	md, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Error reading input file %s: %v", inputFile, err)
	}

	if bytes.Count(md, []byte("---")) < 2 {
		log.Fatalf("Error: input file must contain front matter (YAML) enclosed in '---' lines")
	}
	_, md, _ = bytes.Cut(md, []byte("---"))
	frontMatter, md, _ := bytes.Cut(md, []byte("---"))
	md = bytes.TrimSpace(md)

	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(frontMatter), &fm); err != nil {
		log.Fatalf("Error parsing YAML front matter in %s: %v", inputFile, err)
	}

	if fm.Title == "" || fm.Canonical == "" || fm.Header.Light == "" || fm.Header.Dark == "" || fm.Header.Alt == "" {
		log.Fatalf("Error: front matter must contain title and canonical fields")
	}

	var finalHTML bytes.Buffer
	if err := htmlPrefixTemplate.Execute(&finalHTML, fm); err != nil {
		log.Fatalf("Error executing HTML template: %v", err)
	}
	finalHTML.WriteString(toHTML(md))

	outputFile := strings.TrimSuffix(inputFile, ".md") + ".html"

	if err := os.WriteFile(outputFile, finalHTML.Bytes(), 0644); err != nil {
		log.Fatalf("Error writing output file %s: %v", outputFile, err)
	}
}
