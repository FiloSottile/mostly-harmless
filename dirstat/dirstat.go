// Copyright 2019 Filippo Valsorda
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Command dirstat prints cheap statistics for a directory. It does not recurse
// and it ignores sub-directories, but it counts hidden files. It is meant for
// very large directories, and prints progress every 1024 entries.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: dirstat DIR\n")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening directory: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Fprintf(os.Stderr, "Reading directory...")

	var (
		count, bytes int64
		latestChange time.Time
		latestFile   string
		emptyFiles   int64
		largestFile  string
		largestSize  int64
		hiddenFiles  int64
	)
	for {
		list, err := f.Readdir(1024)
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "\nError reading directory: %v\n", err)
			os.Exit(1)
		}

		for _, fi := range list {
			if fi.IsDir() {
				continue
			}
			count++
			size := fi.Size()
			bytes += size
			if size == 0 {
				emptyFiles++
			}
			if size > largestSize {
				largestSize = size
				largestFile = fi.Name()
			}
			if strings.HasPrefix(fi.Name(), ".") {
				hiddenFiles++
			}
			if fi.ModTime().After(latestChange) {
				latestChange = fi.ModTime()
				latestFile = fi.Name()
			}
		}

		if err == io.EOF {
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Printf("File count:\t%d\n", count)
			fmt.Printf("\tof which %d empty\n", emptyFiles)
			fmt.Printf("\tand %d hidden\n", hiddenFiles)
			fmt.Printf("Total size:\t%d bytes\n", bytes)
			fmt.Printf("Largest file:\t%q\n", largestFile)
			fmt.Printf("\tof %d bytes\n", largestSize)
			fmt.Printf("Latest change:\t%v\n", latestChange)
			fmt.Printf("\ton %q\n", latestFile)
			os.Exit(0)
		}

		fmt.Fprintf(os.Stderr, ".")
	}
}
