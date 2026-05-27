package main

import (
	"os"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func TestParseModulesInProcessList(t *testing.T) {
	// Open the test data file
	file, err := os.Open("testdata/modules-in-process-list")
	if err != nil {
		t.Fatalf("Error opening test file: %v", err)
	}
	defer file.Close()

	// Parse the HTML
	doc, err := html.Parse(file)
	if err != nil {
		t.Fatalf("Error parsing HTML: %v", err)
	}

	// Extract all entries
	var entries []ModuleEntry
	parseTable(doc, &entries)

	// Filter for Pending Review entries
	pendingReviewEntries := filterPendingReview(entries)

	// Test 1: Check total number of Pending Review entries
	expectedTotal := 105
	if len(pendingReviewEntries) != expectedTotal {
		t.Errorf("Expected %d Pending Review entries, got %d", expectedTotal, len(pendingReviewEntries))
	}

	// Test 2: Count entries before 4/25/2026
	cutoffDate := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	countBefore := 0
	for _, entry := range pendingReviewEntries {
		if entry.Date.Before(cutoffDate) {
			countBefore++
		}
	}

	expectedBeforeCutoff := 77
	if countBefore != expectedBeforeCutoff {
		t.Errorf("Expected %d entries before cutoff, got %d", expectedBeforeCutoff, countBefore)
	}

	// Test 3: Sanity check - find Geomys LLC entry with Pending Review (4/25/2026)
	foundGeomys := false
	for _, entry := range pendingReviewEntries {
		if entry.VendorName == "Geomys LLC" {
			if entry.ModuleName != "Go Cryptographic Module" {
				t.Errorf("Expected Geomys LLC module name 'Go Cryptographic Module', got '%s'", entry.ModuleName)
			}
			expectedDate := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
			if !entry.Date.Equal(expectedDate) {
				t.Errorf("Expected Geomys LLC date to be 4/25/2026, got %s", entry.Date.Format("1/2/2006"))
			}
			if entry.Standard != "FIPS 140-3" {
				t.Errorf("Expected Geomys LLC standard 'FIPS 140-3', got '%s'", entry.Standard)
			}
			foundGeomys = true
			break
		}
	}

	if !foundGeomys {
		t.Error("Expected to find Geomys LLC entry with Pending Review status")
	}
}

func TestParseDateFunction(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected time.Time
		wantZero bool
	}{
		{
			name:     "valid date format",
			status:   "Pending Review (4/25/2026)",
			expected: time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
			wantZero: false,
		},
		{
			name:     "valid date with extra spaces",
			status:   "Pending Review  (10/1/2024)",
			expected: time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC),
			wantZero: false,
		},
		{
			name:     "invalid format - no parentheses",
			status:   "Pending Review 4/25/2026",
			expected: time.Time{},
			wantZero: true,
		},
		{
			name:     "invalid format - no date",
			status:   "Pending Review",
			expected: time.Time{},
			wantZero: true,
		},
		{
			name:     "different status",
			status:   "In Review (4/25/2026)",
			expected: time.Time{},
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDate(tt.status)
			if tt.wantZero {
				if !result.IsZero() {
					t.Errorf("Expected zero time, got %v", result)
				}
			} else {
				if !result.Equal(tt.expected) {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestFilterPendingReview(t *testing.T) {
	entries := []ModuleEntry{
		{
			ModuleName: "Test Module 1",
			VendorName: "Test Vendor 1",
			Standard:   "FIPS 140-3",
			Status:     "Pending Review (4/25/2026)",
			Date:       time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			ModuleName: "Test Module 2",
			VendorName: "Test Vendor 2",
			Standard:   "FIPS 140-3",
			Status:     "In Review (4/1/2026)",
			Date:       time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ModuleName: "Test Module 3",
			VendorName: "Test Vendor 3",
			Standard:   "FIPS 140-3",
			Status:     "Pending Review (3/1/2026)",
			Date:       time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	filtered := filterPendingReview(entries)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 Pending Review entries, got %d", len(filtered))
	}

	// Verify the correct entries were filtered
	if filtered[0].ModuleName != "Test Module 1" || filtered[1].ModuleName != "Test Module 3" {
		t.Error("Filtered entries don't match expected Pending Review entries")
	}
}
