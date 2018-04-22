package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	var sr *SearchResult
	if err := json.NewDecoder(res.Body).Decode(&sr); err != nil {
		return nil, errors.Wrap(err, "failed reading search result")
	}
	return sr, nil
}

func (dcb *DocumentCloudBot) DownloadFile(ctx context.Context, url string) (*os.File, error) {
	select {
	case <-dcb.assetRate.C:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	res, err := dcb.httpClient.Do(newRequest(ctx, url))
	if err != nil {
		return nil, errors.Wrap(err, "failed asset request")
	}
	defer res.Body.Close()
	f, err := ioutil.TempFile(dcb.tmpDir, "tmp-dc-asset-")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp file")
	}
	if _, err := io.Copy(f, res.Body); err != nil {
		return nil, errors.Wrap(err, "failed reading asset")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Wrap(err, "failed seeking")
	}
	return f, nil
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
