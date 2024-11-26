package main

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/progrium/darwinkit/macos/appkit"
)

func main() {
	pboard := appkit.Pasteboard_GeneralPasteboard()
	html := pboard.StringForType(appkit.PasteboardTypeHTML)

	p := bluemonday.StrictPolicy()

	p.AllowElements("br", "ol", "ul", "li", "a", "b", "i", "strong",
		"em", "u", "s", "sub", "sup", "code", "pre", "blockquote",
		"h1", "h2", "h3", "h4", "h5", "h6")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("style").OnElements("span")
	p.AllowStyles("font-weight", "font-style", "text-decoration").Globally()

	html = p.Sanitize(html)

	html = strings.Replace(html, "<br/>", "<br/><br/>", -1)

	pboard.ClearContents()
	pboard.SetStringForType(html, appkit.PasteboardTypeHTML)
}
