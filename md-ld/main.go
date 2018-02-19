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
