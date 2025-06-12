# Review Pending Parser

This Go program parses the NIST Cryptographic Module Validation Program (CMVP) "Modules In Process" HTML file to extract all entries that are in "Review Pending" status and counts how many entered the queue before a specified cutoff date (5/8/2025).

## Usage

```bash
# Show all Review Pending entries with details
go run main.go [filename]

# Show only the summary
go run main.go -summary [filename]
```

If no filename is provided, it defaults to "modules-in-process-list" in the current directory.

## Features

- Parses HTML table structure to extract module information
- Filters for entries with "Review Pending" status
- Extracts dates from status field (format: "Review Pending (MM/DD/YYYY)")
- Counts entries that entered the queue before 5/8/2025
- Provides both detailed output and summary-only mode

## Example Output

```
Total Review Pending entries: 180
Entries before 5/8/2025: 151
```

## Dependencies

- golang.org/x/net/html for HTML parsing

## Installation

```bash
go mod tidy
go build
```
