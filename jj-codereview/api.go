package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
}

type GerritChange struct {
	ID        string `json:"triplet_id"`
	Branch    string `json:"branch"`
	Project   string `json:"project"`
	Number    int    `json:"_number"`
	Revisions map[string]struct {
		Number int `json:"_number"`
		Fetch  struct {
			HTTP struct {
				URL string `json:"url"`
				Ref string `json:"ref"`
			} `json:"http"`
		} `json:"fetch"`
	} `json:"revisions"`
	CurrentRevision string `json:"current_revision"`
}

func GetChange(q string) (*GerritChange, error) {
	url := fmt.Sprintf("https://go-review.googlesource.com/changes/?q=%s&n=1&o=ALL_REVISIONS", q)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch change: %v", err)
	}
	defer resp.Body.Close()

	// Gerrit prepends a magic string to prevent XSSI
	reader := bufio.NewReader(resp.Body)
	_, err = reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read anti-XSSI line: %v", err)
	}

	var changes []*GerritChange
	if err := json.NewDecoder(reader).Decode(&changes); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("change not found")
	}
	return changes[0], nil
}
