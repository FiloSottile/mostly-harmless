package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type ModuleEntry struct {
	ModuleName string
	VendorName string
	Standard   string
	Status     string
	Date       time.Time
}

const nistURL = "https://csrc.nist.gov/Projects/cryptographic-module-validation-program/modules-in-process/modules-in-process-list"

func main() {
	var reader io.Reader
	var err error

	if len(os.Args) > 1 && os.Args[1] == "-" {
		reader = os.Stdin
	} else if len(os.Args) > 1 {
		file, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		defer file.Close()
		reader = file
	} else {
		resp, err := http.Get(nistURL)
		if err != nil {
			log.Fatalf("Error downloading from NIST: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("HTTP error: %s", resp.Status)
		}
		reader = resp.Body
	}

	doc, err := html.Parse(reader)
	if err != nil {
		log.Fatalf("Error parsing HTML: %v", err)
	}

	var entries []ModuleEntry
	parseTable(doc, &entries)

	// Filter for Review Pending entries
	reviewPendingEntries := filterReviewPending(entries)

	// Count entries before 5/8/2025
	cutoffDate := time.Date(2025, 5, 8, 0, 0, 0, 0, time.UTC)
	countBefore := 0

	for _, entry := range reviewPendingEntries {
		if entry.Date.Before(cutoffDate) {
			countBefore++
		}
	}

	fmt.Printf("Total Review Pending entries: %d\n", len(reviewPendingEntries))
	fmt.Printf("Entries before 5/8/2025: %d\n", countBefore)
}

func parseTable(n *html.Node, entries *[]ModuleEntry) {
	if n.Type == html.ElementNode && n.Data == "table" {
		parseTableRows(n, entries)
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parseTable(c, entries)
	}
}

func parseTableRows(table *html.Node, entries *[]ModuleEntry) {
	for row := table.FirstChild; row != nil; row = row.NextSibling {
		if row.Type == html.ElementNode {
			if row.Data == "tbody" {
				for tr := row.FirstChild; tr != nil; tr = tr.NextSibling {
					if tr.Type == html.ElementNode && tr.Data == "tr" {
						entry := parseTableRow(tr)
						if entry != nil {
							*entries = append(*entries, *entry)
						}
					}
				}
			} else if row.Data == "tr" {
				entry := parseTableRow(row)
				if entry != nil {
					*entries = append(*entries, *entry)
				}
			}
		}
	}
}

func parseTableRow(tr *html.Node) *ModuleEntry {
	var cells []string

	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type == html.ElementNode && td.Data == "td" {
			cellText := extractText(td)
			// Clean up whitespace and remove unwanted text like "View Contacts"
			cellText = strings.ReplaceAll(cellText, "View Contacts", "")
			cellText = regexp.MustCompile(`\s+`).ReplaceAllString(cellText, " ")
			cells = append(cells, strings.TrimSpace(cellText))
		}
	}

	// We need at least 4 cells: Module Name, Vendor Name, Standard, Status
	if len(cells) < 4 {
		return nil
	}

	entry := &ModuleEntry{
		ModuleName: cells[0],
		VendorName: cells[1],
		Standard:   cells[2],
		Status:     cells[3],
	}

	// Parse date from status field if it contains "Review Pending (date)"
	if strings.Contains(entry.Status, "Review Pending") {
		date := parseDate(entry.Status)
		if !date.IsZero() {
			entry.Date = date
		}
	}

	return entry
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(extractText(c))
	}
	return strings.TrimSpace(text.String())
}

func parseDate(status string) time.Time {
	// Extract date from "Review Pending (MM/DD/YYYY)" format
	re := regexp.MustCompile(`Review Pending\s+\((\d{1,2})/(\d{1,2})/(\d{4})\)`)
	matches := re.FindStringSubmatch(status)
	if len(matches) != 4 {
		return time.Time{}
	}

	// Parse the date - format is MM/DD/YYYY
	dateStr := fmt.Sprintf("%s/%s/%s", matches[1], matches[2], matches[3])
	date, err := time.Parse("1/2/2006", dateStr)
	if err != nil {
		fmt.Printf("Error parsing date '%s': %v\n", dateStr, err)
		return time.Time{}
	}

	return date
}

func filterReviewPending(entries []ModuleEntry) []ModuleEntry {
	var filtered []ModuleEntry
	for _, entry := range entries {
		if strings.Contains(entry.Status, "Review Pending") {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
