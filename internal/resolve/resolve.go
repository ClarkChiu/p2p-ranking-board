// Package resolve turns a user-approved ranking ID back into a magnet link.
// That is the full extent of ranking-board's involvement with downloading: it
// emits a magnet on stdout and stops. Verifying swarm health and fetching bytes
// are out of scope: the caller pipes the magnet wherever it likes, and
// ranking-board has no dependency on, or knowledge of, any downloader.
package resolve

import (
	"fmt"
	"net/url"

	"github.com/freebooters/p2p-ranking-board/internal/snapshot"
	"github.com/freebooters/p2p-ranking-board/internal/source"
)

// ByID finds the snapshot entry whose short ID matches id.
func ByID(snap *snapshot.Snapshot, id string) (source.Entry, error) {
	for _, e := range snap.Entries {
		if e.ID() == id {
			return e, nil
		}
	}
	return source.Entry{}, fmt.Errorf("id %q not in latest snapshot", id)
}

// Magnet builds a minimal magnet (infohash + display name) for an entry.
func Magnet(e source.Entry) string {
	m := "magnet:?xt=urn:btih:" + e.InfoHash
	if e.Title != "" {
		m += "&dn=" + url.QueryEscape(e.Title)
	}
	return m
}
