// +build dev generate

package data

import (
	"go/build"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/httpfs/filter"
)

//go:generate go run -tags=generate templates_generate.go

var Templates http.FileSystem = filter.Skip(http.Dir(importPathToDir(
	"github.com/FiloSottile/mostly-harmless/covfefe/cmd/webfefe/data/_templates")),
	filter.FilesWithExtensions(".go"))

func importPathToDir(importPath string) string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	p, err := build.Import(importPath, wd, build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}
