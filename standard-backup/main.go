package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: standard-backup hostname credentials.json")
	}
	creds, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	hc := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", "https://"+os.Args[1]+"/auth/sign_in", bytes.NewReader(creds))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatal(resp.Status)
	}
	var jsonResp struct {
		Token string
	}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()

	req, err = http.NewRequest("POST", "https://"+os.Args[1]+"/items/sync", nil)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatal(resp.Status)
	}
	req.Header.Set("Authorization", "Bearer "+jsonResp.Token)
	resp, err = hc.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	var jsonData struct {
		Items json.RawMessage `json:"retrieved_items"`
	}
	err = json.NewDecoder(resp.Body).Decode(&jsonData)
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()

	if len(jsonData.Items) < 100*1024 || jsonData.Items[0] != '[' {
		log.Fatal("data looks corrupted")
	}

	var jsonCreds struct {
		Email string
	}
	err = json.Unmarshal(creds, &jsonCreds)
	if err != nil {
		log.Fatal(err)
	}
	req, err = http.NewRequest("GET", "https://"+os.Args[1]+"/auth/params?email="+jsonCreds.Email, nil)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatal(resp.Status)
	}
	resp, err = hc.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	authParams, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()

	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Items      json.RawMessage `json:"items"`
		AuthParams json.RawMessage `json:"auth_params"`
	}{
		Items:      jsonData.Items,
		AuthParams: json.RawMessage(authParams),
	}); err != nil {
		log.Fatal(err)
	}
}
