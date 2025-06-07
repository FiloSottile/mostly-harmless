package main

import (
	"bytes"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"rsc.io/markdown"
)

const htmlPrefixTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="canonical" href="$CANONICAL">
    <title>$TITLE</title>
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
`

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

	if fm.Title == "" || fm.Canonical == "" {
		log.Fatalf("Error: front matter must contain title and canonical fields")
	}

	replacer := strings.NewReplacer(
		"$TITLE", fm.Title,
		"$CANONICAL", fm.Canonical,
	)
	htmlPrefix := replacer.Replace(htmlPrefixTemplate)

	var finalHTML bytes.Buffer
	finalHTML.WriteString(htmlPrefix)
	finalHTML.WriteString(toHTML(md))

	outputFile := strings.TrimSuffix(inputFile, ".md") + ".html"

	if err := os.WriteFile(outputFile, finalHTML.Bytes(), 0644); err != nil {
		log.Fatalf("Error writing output file %s: %v", outputFile, err)
	}
}
