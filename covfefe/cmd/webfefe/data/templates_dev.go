// +build dev generate

package data

import "net/http"

//go:generate go run -tags=generate templates_generate.go

var Templates http.FileSystem = http.Dir("templates")
