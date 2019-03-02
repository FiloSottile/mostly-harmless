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
//     this is a document written in [Markdown][markdown daringfireball]
//
// and at the end run md-ld, which will extract for you the references you need
// to add links for, like
//
//     [markdown daringfireball]:
package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/russross/blackfriday"
)

func main() {
	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	seen := make(map[string]struct{})

	f := func(reference string) (ref *blackfriday.Reference, overridden bool) {
		if _, ok := seen[reference]; ok || reference == "" {
			return nil, false
		}
		fmt.Printf("[%s]: \n", reference)
		seen[reference] = struct{}{}
		return nil, false
	}

	blackfriday.MarkdownOptions(in, blackfriday.HtmlRenderer(0, "", ""),
		blackfriday.Options{
			Extensions:        blackfriday.EXTENSION_FENCED_CODE,
			ReferenceOverride: blackfriday.ReferenceOverrideFunc(f),
		})
}
