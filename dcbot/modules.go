package main

import (
	"context"
	"encoding/json"
	"time"

	"crawshaw.io/iox"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (dcb *DocumentCloudBot) Latest(ctx context.Context) error {
	log := logrus.WithField("module", "latest")

	for ctx.Err() == nil {
		log.Debug("Fetching latest entries...")
		page := 1
	SearchLoop:
		for ctx.Err() == nil {
			sr, err := dcb.Search(ctx, page)
			if err != nil {
				log.WithError(err).Error("Search error")
				continue
			}
			for _, doc := range sr.Documents {
				id := IDForDocument(doc)
				new, err := dcb.insertDocument(ctx, id, doc)
				if err != nil {
					return err
				}
				log.WithField("id", id).WithField("new", new).Debug("Got document")
				if !new {
					break SearchLoop
				}
			}
			page++
		}

		log.Debug("Sleeping 5 minutes...")
		select {
		case <-time.After(5 * time.Minute):
		case <-ctx.Done():
		}
	}

	log.WithError(ctx.Err()).Debug("Shutting down")
	return ctx.Err()
}

func (dcb *DocumentCloudBot) Backfill(ctx context.Context, fromPage int) error {
	log := logrus.WithField("module", "backfill")

	page := fromPage
	for ctx.Err() == nil {
		sr, err := dcb.Search(ctx, page)
		if err != nil {
			log.WithError(err).Error("Search error")
			continue
		}
		if len(sr.Documents) == 0 {
			log.Info("Reached the end!")
			return nil
		}
		for _, doc := range sr.Documents {
			id := IDForDocument(doc)
			new, err := dcb.insertDocument(ctx, id, doc)
			if err != nil {
				return err
			}
			log.WithField("id", id).WithField("new", new).Debug("Got document")
		}
		log.WithFields(logrus.Fields{
			"page":     sr.Page,
			"total":    sr.Total,
			"per_page": sr.PerPage}).Debug("Downloaded page")
		page++
	}

	log.WithError(ctx.Err()).Debug("Shutting down")
	return ctx.Err()
}

func (dcb *DocumentCloudBot) Download(ctx context.Context) error {
	log := logrus.WithField("module", "download")

	for ctx.Err() == nil {
		jsonDoc, err := dcb.getPendingDocument(ctx)
		if err != nil {
			return err
		}
		if jsonDoc == nil {
			log.Debug("Sleeping 5 minutes...")
			select {
			case <-time.After(5 * time.Minute):
			case <-ctx.Done():
			}
			continue
		}
		var doc struct {
			ID        string
			Resources map[string]interface{}
		}
		if err := json.Unmarshal(jsonDoc, &doc); err != nil {
			return errors.Wrap(err, "failed to unmarshal document")
		}
		log.WithField("doc", doc.ID).Debug("Downloading files")
		files := make(map[string]*iox.BufferFile)
		cleanup := func() {
			for _, f := range files {
				f.Close()
			}
		}
		sizes := make(map[string]int64)
		for _, res := range []string{"pdf", "text"} {
			url, ok := doc.Resources[res].(string)
			if !ok {
				cleanup()
				return errors.Errorf("document %s is missing resource %s", doc.ID, res)
			}
			var (
				n   int64
				f   *iox.BufferFile
				err error
			)
			for retry := 0; retry < 5; retry++ {
				n, f, err = dcb.DownloadFile(ctx, url)
				if err != nil {
					log.WithError(err).WithField("retry", retry).Error("error downloading file")
					continue
				}
				break
			}
			if err != nil {
				cleanup()
				return err
			}
			files[res] = f
			sizes[res] = n
		}
		err = dcb.insertFiles(ctx, doc.ID, files, sizes)
		cleanup()
		if err != nil {
			return err
		}
	}

	log.WithError(ctx.Err()).Debug("Shutting down")
	return ctx.Err()
}
