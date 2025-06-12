package main

import (
	"flag"
	"fmt"
	"log"
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

func main() {
	summaryOnly := flag.Bool("summary", false, "Show only the summary")
	flag.Parse()

	filename := "modules-in-process-list"
	if flag.NArg() > 0 {
		filename = flag.Arg(0)
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	doc, err := html.Parse(file)
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

	if !*summaryOnly {
		fmt.Printf("Found %d Review Pending entries:\n\n", len(reviewPendingEntries))
	}

	for _, entry := range reviewPendingEntries {
		if entry.Date.Before(cutoffDate) {
			countBefore++
		}
		if !*summaryOnly {
			fmt.Printf("Module: %s\n", entry.ModuleName)
			fmt.Printf("Vendor: %s\n", entry.VendorName)
			fmt.Printf("Standard: %s\n", entry.Standard)
			fmt.Printf("Status: %s\n", entry.Status)
			fmt.Printf("Date: %s\n", entry.Date.Format("1/2/2006"))
			if entry.Date.Before(cutoffDate) {
				fmt.Printf("*** BEFORE CUTOFF ***")
			}
			fmt.Println("---")
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
