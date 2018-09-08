package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

type SearchResult struct {
	Total     int               `json:"total"`
	Page      int               `json:"page"`
	PerPage   int               `json:"per_page"`
	Documents []json.RawMessage `json:"documents"`
}

func newRequest(ctx context.Context, url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", "DCBot/1.0 <dcbot@bip.filippo.io> (https://github.com/FiloSottile/mostly-harmless/tree/master/dcbot)")
	return req.WithContext(ctx)
}

const perPage = 900

func (dcb *DocumentCloudBot) Search(ctx context.Context, page int) (*SearchResult, error) {
	select {
	case <-dcb.searchRate.C:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	url := fmt.Sprintf("https://www.documentcloud.org/api/search.json?per_page=%d&page=%d", perPage, page)
	res, err := dcb.httpClient.Do(newRequest(ctx, url))
	if err != nil {
		return nil, errors.Wrap(err, "failed search request")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("search result returned %d: %s", res.StatusCode, res.Status)
	}
	var sr *SearchResult
	if err := json.NewDecoder(res.Body).Decode(&sr); err != nil {
		return nil, errors.Wrap(err, "failed reading search result")
	}
	return sr, nil
}

func (dcb *DocumentCloudBot) DownloadFile(ctx context.Context, url string, f *os.File) error {
	select {
	case <-dcb.assetRate.C:
	case <-ctx.Done():
		return ctx.Err()
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}

	res, err := dcb.httpClient.Do(newRequest(ctx, url))
	if err != nil {
		return errors.Wrap(err, "failed asset request")
	}
	defer res.Body.Close()
	if _, err := io.Copy(f, res.Body); err != nil {
		return errors.Wrap(err, "failed reading asset")
	}
	return nil
}

func IDForDocument(doc json.RawMessage) string {
	var d struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(doc, &d); err != nil {
		panic(err)
	}
	return d.ID
}
