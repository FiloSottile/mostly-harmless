package main

import "sync"

// Table frequent implements the Space Saving algorithm for counting
// appearances of the most frequent items in a stream of data (sometimes
// referred to as the heavy hitters).
type Table struct {
	mu    sync.Mutex
	items []Item
	index map[string]int // value => index in items
}

type Item struct {
	Value    string
	Latest   string
	Count    int
	MaxError int
}

// NewTable creates a new Table with the specified size.
func NewTable(size int) *Table {
	return &Table{
		items: make([]Item, 0, size),
		index: make(map[string]int, size),
	}
}

// Count updates the count of the specified value in the table, and associates
// the attr value as its [Item.Latest] attribute.
func (t *Table) Count(value, attr string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	i, ok := t.index[value]
	switch {
	case ok:
		t.items[i].Count++
	case len(t.items) < cap(t.items):
		t.items = append(t.items, Item{Value: value, Count: 1})
		i = len(t.items) - 1
	default:
		i = len(t.items) - 1
		delete(t.index, t.items[i].Value)
		t.items[i].Value = value
		t.items[i].MaxError = t.items[i].Count
		t.items[i].Count++
	}
	for k := i - 1; k >= 0; k-- {
		if t.items[k].Count >= t.items[i].Count {
			break
		}
		t.items[k], t.items[i] = t.items[i], t.items[k]
		t.index[t.items[i].Value] = i
		i = k
	}
	t.items[i].Latest = attr
	t.index[t.items[i].Value] = i
}

// Top returns up to n items with the highest counts.
func (t *Table) Top(n int) []Item {
	t.mu.Lock()
	defer t.mu.Unlock()
	n = min(n, len(t.items))
	top := make([]Item, n)
	copy(top, t.items[:n])
	return top
}
