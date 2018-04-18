package main

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

func (dcb *DocumentCloudBot) Latest(ctx context.Context) error {
	log := logrus.WithField("module", "latest")

	for ctx.Err() == nil {
		log.Info("Fetching latest entries...")
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

	return ctx.Err()
}
