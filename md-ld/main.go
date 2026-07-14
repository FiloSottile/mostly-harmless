// Copyright 2018 Filippo Valsorda
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Command md-ld generates placeholders for all unmatched link references in a
// Markdown document.
//
// The idea is that you write your document without stopping to add links, and
// just assign meaningful references like
//
//	this is a document written in [Markdown][markdown daringfireball]
//
// and at the end run md-ld, which will extract for you the references you need
// to add links for, like
//
//	[markdown daringfireball]:
//
// With -llm, the document and the placeholders are handed to claude, which
// looks the links up and fills the placeholders in.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/russross/blackfriday"
)

func main() {
	llmFlag := flag.Bool("llm", false, "ask claude to look up and fill in the links")
	flag.Parse()

	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var references []string
	seen := make(map[string]struct{})

	f := func(reference string) (ref *blackfriday.Reference, overridden bool) {
		// Ignore footnotes.
		if strings.HasPrefix(reference, "^") {
			return nil, false
		}

		if _, ok := seen[reference]; ok || reference == "" {
			return nil, false
		}
		references = append(references, reference)
		seen[reference] = struct{}{}
		return nil, false
	}

	blackfriday.MarkdownOptions(in, blackfriday.HtmlRenderer(0, "", ""),
		blackfriday.Options{
			Extensions:        blackfriday.EXTENSION_FENCED_CODE,
			ReferenceOverride: blackfriday.ReferenceOverrideFunc(f),
		})

	if !*llmFlag {
		for _, reference := range references {
			fmt.Printf("[%s]: \n", reference)
		}
		return
	}

	if len(references) == 0 {
		return
	}
	if err := fillIn(string(in), references); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

const prompt = `You are given a Markdown document and the list of its link
references that have no definition yet. For each reference, find the URL the
author meant to link to: the reference name is the hint, and so is the text it
is attached to and the surrounding context. Use web search and fetch the pages
to make sure the URLs are correct, current, and canonical (no tracking
parameters, no redirectors, no AMP). Prefer the primary source over aggregators.

Output only the definition lines, one per reference, in the order they are
listed below, in the form

    [reference name]: https://example.com/page

If you can't confidently determine a URL, output the reference with an empty
definition, like

    [reference name]:

Do not output anything else: no explanations, no code fences, no other Markdown.

Here are the references to define:

%s

Here is the document:

%s
`

// fillIn runs claude on the document, and writes the definitions it produces to
// standard output. claude runs in an empty temporary directory, with no access
// to the file system, so that the directory md-ld was invoked in has no effect
// on the result and is left untouched.
func fillIn(document string, references []string) error {
	var list strings.Builder
	for _, reference := range references {
		fmt.Fprintf(&list, "[%s]: \n", reference)
	}

	dir, err := ioutil.TempDir("", "md-ld")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	cmd := exec.Command("claude", "--print",
		// Only the tools needed to look links up, so that no file in the
		// invocation directory (or anywhere else) can be read or written.
		"--tools", "WebSearch,WebFetch",
		// The tools above are harmless, and there is no one to ask anyway.
		"--permission-mode", "bypassPermissions",
		// Ignore any MCP server and settings file configured on this machine.
		"--strict-mcp-config", "--setting-sources", "",
		"--no-session-persistence")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(fmt.Sprintf(prompt, list.String(), document))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude: %v", err)
	}
	return nil
}
