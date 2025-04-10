package main

import (
	"os"
)

var jjConfigTOML = `
[revset-aliases]
"jjcrmailpending()" = "remote_bookmarks(remote=origin)..jjcrmail() ~ remote_bookmarks(remote=gerrit)"
"jjcrbranchpoint(x)" = "heads(::x & ::remote_bookmarks(remote=origin))"
"jjcrbranchhead(x)" = "jjcrbranchpoint(x):: & remote_bookmarks(remote=origin)"

[templates]
bookmarks = "separate('\n', remote_bookmarks.map(|b| if(b.remote() == 'origin', b.name()))) ++ '\n'"

[ui]
color = "never"
paginate = "never"
`

func jjConfig() string {
	tmp, err := os.CreateTemp("", "jj-config-")
	if err != nil {
		dief("failed to create temporary file: %v", err)
	}
	if _, err := tmp.WriteString(jjConfigTOML); err != nil {
		dief("failed to write temporary file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		dief("failed to close temporary file: %v", err)
	}
	return tmp.Name()
}

func jjLog(config string, extra ...string) func(args ...string) []string {
	return func(args ...string) []string {
		a := append(extra, "--quiet", "--config-file", config, "log", "--no-graph")
		a = append(a, args...)
		return lines(cmdOutput("jj", a...))
	}
}
