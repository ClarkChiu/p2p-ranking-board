// Package diff is the pure comparison core: given the previous and current
// ranking maps it reports what changed. No I/O, no side effects — trivially
// testable, and the deterministic half of the pipeline (judgement of which
// changes matter is left to the external agent).
package diff

import "github.com/freebooters/p2p-ranking-board/internal/source"

type Kind string

const (
	New       Kind = "new"        // charted this run, absent last run
	Dropped   Kind = "dropped"    // present last run, gone this run
	RankMove  Kind = "rank_move"  // present both runs, different rank
	SeedShift Kind = "seed_shift" // present both runs, seeders moved beyond threshold
)

// Change is one detected difference. PrevRank/PrevSeeders are zero for New.
type Change struct {
	Kind        Kind         `json:"kind"`
	Entry       source.Entry `json:"entry"`
	PrevRank    int          `json:"prev_rank,omitempty"`
	PrevSeeders int          `json:"prev_seeders,omitempty"`
}

// Compare reports the differences between prev and cur (both keyed by
// Entry.Key). seedThreshold is the fractional change in seeders required to
// report a SeedShift (e.g. 0.20 = ±20%); it suppresses swarm jitter noise.
// A New entry never also yields RankMove/SeedShift; the kinds are exclusive.
func Compare(prev, cur map[string]source.Entry, seedThreshold float64) []Change {
	var changes []Change

	for key, c := range cur {
		p, ok := prev[key]
		if !ok {
			changes = append(changes, Change{Kind: New, Entry: c})
			continue
		}
		if c.Rank != p.Rank {
			changes = append(changes, Change{Kind: RankMove, Entry: c, PrevRank: p.Rank, PrevSeeders: p.Seeders})
		}
		if seedShifted(p.Seeders, c.Seeders, seedThreshold) {
			changes = append(changes, Change{Kind: SeedShift, Entry: c, PrevRank: p.Rank, PrevSeeders: p.Seeders})
		}
	}

	for key, p := range prev {
		if _, ok := cur[key]; !ok {
			changes = append(changes, Change{Kind: Dropped, Entry: p, PrevRank: p.Rank, PrevSeeders: p.Seeders})
		}
	}
	return changes
}

func seedShifted(prev, cur int, threshold float64) bool {
	if prev == 0 {
		return cur > 0
	}
	delta := float64(cur-prev) / float64(prev)
	if delta < 0 {
		delta = -delta
	}
	return delta >= threshold
}
