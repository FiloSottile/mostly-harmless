package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (dcb *DocumentCloudBot) Latest(ctx context.Context) error {
	log := logrus.WithField("module", "latest")

	for ctx.Err() == nil {
		log.Debug("Fetching latest entries...")
	SearchLoop:
		for page := 1; ctx.Err() == nil; page++ {
			sr, err := dcb.Search(ctx, page)
			if err != nil {
				return err
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

	for page := fromPage; ctx.Err() == nil; page++ {
		sr, err := dcb.Search(ctx, page)
		if err != nil {
			return err
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
		files := make(map[string][]byte)
		for _, res := range []string{"pdf", "text"} {
			url, ok := doc.Resources[res].(string)
			if !ok {
				return errors.Errorf("document %s is missing resource %s", doc.ID, res)
			}
			body, err := dcb.DownloadFile(ctx, url)
			if err != nil {
				return err
			}
			files[res] = body
		}
		if err := dcb.insertFiles(ctx, doc.ID, files); err != nil {
			return err
		}
	}

	log.WithError(ctx.Err()).Debug("Shutting down")
	return ctx.Err()
}
