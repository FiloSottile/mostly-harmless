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
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func main() {
	f, err := os.Open(os.Args[3])
	if err != nil {
		log.Fatalln("Failed to open file:", err)
	}
	defer f.Close()

	log.Println("Connecting via SSH...")
	sshc, err := sshConnect(os.Args[1], os.Args[2])
	if err != nil {
		log.Fatalln("Failed to connect via SSH:", err)
	}
	defer sshc.Close()

	log.Println("Uploading file to Web UI...")
	if err := uploadFile(sshc.Dial, f); err != nil {
		log.Fatalln("Failed to upload file:", err)
	}

	log.Println("Success!")
}

func sshConnect(endpoint, fingerprint string) (*ssh.Client, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH_AUTH_SOCK: %v", err)
	}

	agentClient := agent.NewClient(conn)
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agentClient.Signers),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if ssh.FingerprintSHA256(key) != fingerprint {
				return fmt.Errorf("incorrect host key: %v", key)
			}
			return nil
		},
	}

	return ssh.Dial("tcp", endpoint, config)
}

func uploadFile(dial func(network, addr string) (net.Conn, error), f *os.File) error {
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

	client := &http.Client{
		Transport: &http.Transport{
			Dial:              dial,
			DisableKeepAlives: true,
		},
		Timeout: 1 * time.Minute,
	}
	url := "http://127.0.0.1/upload"
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
