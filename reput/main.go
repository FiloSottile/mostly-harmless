// Copyright 2019 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalln("Failed to open file:", err)
	}

	if err := uploadFile(f); err != nil {
		log.Fatalln("Failed to upload file:", err)
	}
}

func uploadFile(f *os.File) error {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	name := filepath.Base(f.Name())
	if fw, err := w.CreateFormFile("file", name); err != nil {
		return err
	} else if _, err := io.Copy(fw, f); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	client := &http.Client{Timeout: 1 * time.Minute}
	url := "http://10.11.99.1/upload"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("got HTTP status %d: %s", res.StatusCode, res.Status)
	}
	return nil
}
