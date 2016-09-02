package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	barChar = "∎"
)

type histogram struct {
	points  []time.Duration
	slowest time.Duration
	fastest time.Duration
}

// newHistogram produces an empty histogram.
func newHistogram() *histogram {
	return &histogram{
		slowest: 0,
		fastest: time.Duration(math.MaxInt64),
	}
}

// Observe updates the histogram with a new measurement.
func (h *histogram) Observe(d time.Duration) {
	h.points = append(h.points, d)
	if d > h.slowest {
		h.slowest = d
	}
	if d < h.fastest {
		h.fastest = d
	}
}

type ByDuration []time.Duration

func (a ByDuration) Len() int           { return len(a) }
func (a ByDuration) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDuration) Less(i, j int) bool { return a[i] < a[j] }

// Print formats the histogram data and prints it to stdout.
func (h *histogram) Print(log bool) {
	sort.Sort(ByDuration(h.points))

	// buckets is the upper threshold for each bucket in the hist
	buckets := make([]time.Duration, 8)
	if log {
		bs := h.slowest - h.fastest
		for i := range buckets {
			buckets[len(buckets)-1-i] = h.fastest + bs
			bs = bs / 2
		}
	} else {
		bs := int64(h.slowest-h.fastest)/int64(len(buckets)-1) + 1
		for i := range buckets {
			buckets[i] = h.fastest + time.Duration(bs*int64(i))
		}
	}

	// counts is the number of latencies that fell in each bucket.
	counts := make([]int, len(buckets))
	var bi, max int
	for i := 0; i < len(h.points); {
		if h.points[i] <= buckets[bi] {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		} else {
			panic(fmt.Sprintf("%d is higher than %d", h.points[i], buckets[bi]))
		}
	}

	// Print histogram to stdout.
	lowerBound := time.Duration(0)
	for i, upperBound := range buckets {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = counts[i] * 40 / max
		}
		fmt.Printf("  %3dms - %3dms [%v]\t|%v\n", lowerBound/time.Millisecond,
			upperBound/time.Millisecond, counts[i], strings.Repeat(barChar, barLen))
		lowerBound = upperBound
	}
}
