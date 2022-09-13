# cliplet

This is a small bookmarklet that copies a `go.dev` shortened version of the
current URL, inspired by an analogous Google-internal tool.

```
javascript:(()=>{const o=[[/github\.com\/golang\/go\/issues\/(\d+)/i,"https://go.dev/issue/"],[/go-review\.googlesource\.com\/c\/.*\/\+\/([\d\/]+)/i,"https://go.dev/cl/"],[/github\.com\/golang\/go\/wiki\/(\w+)/i,"https://go.dev/wiki/"],[/go\.googlesource\.com\/proposal\/\+\/master\/design\/([\w\-]+)\.md/i,"https://go.dev/design/"]];for(const i of o){const o=window.location.href.match(i[0]);if(o)return void navigator.clipboard.writeText(i[1]+o[1])}navigator.clipboard.writeText(window.location.href)})();
```

To use it, create a bookmark with the URL above, and place it in the bookmarks
bar. On Firefox, it's possible to place the bookmarks bar next to the URL bar,
not to sacrifice vertical space. The `✂` character is a good name for it.
