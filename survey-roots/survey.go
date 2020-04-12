// Copyright 2019 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

const Dockerfile = `
FROM golang:alpine3.11

ENV CGO_ENABLED=0
ADD . /usr/local/src/survey-roots
RUN cd /usr/local/src/survey-roots && go install

FROM {IMAGE}

{SETUP}

COPY --from=0 /go/bin/survey-roots /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/survey-roots"]
`

type Target struct {
	Image, Extra, Setup string
}

const aptGetDesc = "with ca-certificates"
const aptGetInstall = "RUN apt-get update && apt-get install -y ca-certificates"

var Targets = []Target{
	{Image: "alpine:3.11"},
	{Image: "alpine:3.8"},
	{Image: "debian:buster", Extra: aptGetDesc, Setup: aptGetInstall},
	{Image: "debian:stretch", Extra: aptGetDesc, Setup: aptGetInstall},
	{Image: "gcr.io/distroless/static-debian10"},
	{Image: "ubuntu:20.04", Extra: aptGetDesc, Setup: aptGetInstall},
	{Image: "ubuntu:18.04", Extra: aptGetDesc, Setup: aptGetInstall},
	{Image: "ubuntu:16.04", Extra: aptGetDesc, Setup: aptGetInstall},
	{Image: "centos:8"},
	{Image: "centos:7"},
	{Image: "centos:6"},
	{Image: "amazonlinux:2"},
	{Image: "amazonlinux:1"},
	{Image: "fedora:31"},
	{Image: "archlinux"},
}

func runSurvey(t Target) {
	tmpfile, err := ioutil.TempFile("", "surve-roots")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	dockerfile := strings.Replace(
		strings.Replace(Dockerfile, "{IMAGE}", t.Image, -1),
		"{SETUP}", t.Setup, -1,
	)

	if _, err := tmpfile.WriteString(dockerfile); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("docker", "build", "-f", tmpfile.Name(), "-q", ".")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	image := strings.TrimSpace(string(out))
	cmd = exec.Command("docker", "run", "--", image)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	for _, target := range Targets {
		fmt.Printf("### %s", target.Image)
		if target.Extra != "" {
			fmt.Printf(" (%s)", target.Extra)
		}
		fmt.Printf("\n\n")
		runSurvey(target)
		fmt.Printf("\n")
	}
}
