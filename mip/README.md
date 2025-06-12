# Review Pending Parser

This Go program parses the NIST Cryptographic Module Validation Program (CMVP) "Modules In Process" HTML page to extract all entries that are in "Review Pending" status and counts how many entered the queue before a specified cutoff date (5/8/2025).

## Usage

```bash
# Download from NIST and show all Review Pending entries with details
go run main.go

# Download from NIST and show only the summary
go run main.go -summary

# Use local file instead of downloading
go run main.go -file testdata/modules-in-process-list

# Use local file and show only summary
go run main.go -file testdata/modules-in-process-list -summary
```

By default, the program downloads the latest data from the NIST website. Use the `-file` flag to specify a local file instead.

## Features

- **Live data**: Downloads the latest modules list from NIST website by default
- **Local file support**: Can work with saved HTML files using `-local` flag
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

## Testing

```bash
# Run all tests
go test -v
```

## Installation

```bash
go mod tidy
go build
```
