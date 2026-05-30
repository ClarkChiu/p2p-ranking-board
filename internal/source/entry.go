// Package source fetches and normalizes ranking listings from a torrent index.
// The first (and only) source is apibay, The Pirate Bay's JSON backend.
package source

import (
	"fmt"
	"time"
)

// Categories tracked in the first version. apibay category codes:
// 207 HD movies, 208 HD TV, 301 Windows, 303 Linux/UNIX, 401 PC games,
// 601 e-books. (300 "Applications" dropped: it returns the same Windows rows as
// 301, so tracking both was pure redundancy.)
var Categories = []int{207, 208, 301, 303, 401, 601}

// Entry is one normalized ranking row. Rank is the 1-based position within its
// category's top-100 list.
type Entry struct {
	InfoHash  string    `json:"info_hash"`
	Title     string    `json:"title"`
	Category  int       `json:"category"`
	Rank      int       `json:"rank"`
	Seeders   int       `json:"seeders"`
	Leechers  int       `json:"leechers"`
	SizeBytes int64     `json:"size_bytes"`
	Added     time.Time `json:"added"`
}

// Key is the snapshot map key: category + full infohash. A torrent that charts
// in two categories is two entries, so per-category rank moves compare correctly.
func (e Entry) Key() string { return fmt.Sprintf("%d:%s", e.Category, e.InfoHash) }

// ID is the short, human-facing handle used in notifications and `get <id>`:
// "<category>:<first 12 hex of infohash>".
func (e Entry) ID() string {
	h := e.InfoHash
	if len(h) > 12 {
		h = h[:12]
	}
	return fmt.Sprintf("%d:%s", e.Category, h)
}
