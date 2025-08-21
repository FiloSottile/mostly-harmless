package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func domainr(ctx context.Context, pool *sqlitex.Pool) error {
	read, err := pool.Take(ctx)
	if err != nil {
		return fmt.Errorf("failed to take database connection: %w", err)
	}
	defer pool.Put(read)
	write, err := pool.Take(ctx)
	if err != nil {
		return fmt.Errorf("failed to take database connection: %w", err)
	}
	defer pool.Put(write)

	ticker := time.NewTicker(24 * time.Hour)
	for {
		if err := sqlitex.Execute(read, `
		    SELECT DISTINCT etldp1 FROM hostnames
		    WHERE domainr_status IS NULL
			OR domainr_status = 'unknown' OR domainr_status = 'undelegated'
			OR domainr_updated < datetime('now', '-31 days')
		`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				domain := stmt.ColumnText(0)
				status, err := domainrStatus(domain)
				if err != nil {
					return fmt.Errorf("failed to get status for domain %q: %w", domain, err)
				}

				slog.Debug("fetched domainr status", "domain", domain, "status", status)

				return sqlitex.Execute(write, `
					UPDATE hostnames
					SET domainr_status = :status, domainr_updated = datetime('now')
					WHERE etldp1 = :domain
				`, &sqlitex.ExecOptions{
					Named: map[string]any{
						":domain": domain,
						":status": status,
					},
				})
			},
		}); err != nil {
			slog.Error("failed to fetch domainr statuses", "error", err)
		}

		size, err := dbSize(read)
		if err != nil {
			return fmt.Errorf("failed to get database size: %w", err)
		}
		if err := sqlitex.Execute(read, `
		    SELECT hostname, domainr_status FROM hostnames
		    WHERE domainr_status != 'active' AND domainr_status IS NOT NULL
			AND bad_since IS NULL
		`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				hostname := stmt.ColumnText(0)
				status := stmt.ColumnText(1)

				if status == "inactive active" {
					// There are a few of these and apparently they are active.
					return nil
				}

				for _, s := range strings.Split(status, " ") {
					switch s {
					case "unknown", "undelegated", "reserved", "premium", "claimed", "tld",
						"disallowed", "zone", "suffix", "active", "dpml", "pending", "invalid":
						// These are neutral or active statuses.
						//
						// Of these, "undelegated active" is suspect, but could
						// also be downtime. It would be nice to get access to a
						// historical database with information on deleted and
						// re-activated domains, or just compare the current
						// creation date with the first sumdb entry.
					case "inactive", "parked", "marketed", "expiring", "deleting", "priced", "transferable":
						// These are bad statuses.
						slog.Debug("marking hostname as bad", "hostname", hostname, "status", status)
						return sqlitex.Execute(write, `
							UPDATE hostnames
							SET bad_since = :size
							WHERE hostname = :hostname
						`, &sqlitex.ExecOptions{
							Named: map[string]any{
								":hostname": hostname,
								":size":     size,
							},
						})
					default:
						slog.Warn("unknown domainr status", "hostname", hostname, "status", status)
					}
				}
				return nil
			},
		}); err != nil {
			slog.Error("failed to update hostname statuses", "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

var domainrClient = &http.Client{
	Timeout: 35 * time.Second,
}

func domainrStatus(domain string) (string, error) {
	url := "https://domainr.p.rapidapi.com/v2/status?domain=" + domain
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-RapidAPI-Host", "domainr.p.rapidapi.com")
	req.Header.Set("X-RapidAPI-Key", os.Getenv("DOMAINR_API_KEY"))
	resp, err := domainrClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Warn("unexpected status code from Domainr API",
			"status", resp.StatusCode, "body", string(body), "domain", domain)
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var result struct {
		Status []struct {
			Domain string `json:"domain"`
			Status string `json:"status"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	if len(result.Status) == 0 {
		return "", fmt.Errorf("no status found")
	}
	if result.Status[0].Domain != domain {
		return "", fmt.Errorf("domain mismatch: expected %s, got %s", domain, result.Status[0].Domain)
	}
	return result.Status[0].Status, nil
}
