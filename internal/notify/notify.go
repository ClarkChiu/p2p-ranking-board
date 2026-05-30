// Package notify turns a set of changes into something the user (or, in this
// project's intended setup, the external Hermes agent's LLM step) can act on.
// The interface is pluggable; the first version ships only a stdout notifier,
// which doubles as the structured feed an LLM reads.
package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/freebooters/p2p-ranking-board/internal/diff"
)

// Notifier delivers a batch of changes somewhere. Implementations: stdout now,
// Telegram/etc. later — without touching the diff logic.
type Notifier interface {
	Notify(ctx context.Context, changes []diff.Change) error
}

// Filter keeps only changes whose kind is in `only`. An empty `only` passes
// everything. Recognised keys: new, dropped, rank_move, seed_shift.
func Filter(changes []diff.Change, only []string) []diff.Change {
	if len(only) == 0 {
		return changes
	}
	keep := map[string]bool{}
	for _, k := range only {
		keep[k] = true
	}
	out := changes[:0:0]
	for _, c := range changes {
		if keep[string(c.Kind)] {
			out = append(out, c)
		}
	}
	return out
}

// Stdout writes changes to w: human-readable lines, or one JSON array when
// JSON is true (the form the external agent parses).
type Stdout struct {
	W    io.Writer
	JSON bool
}

func (s Stdout) Notify(_ context.Context, changes []diff.Change) error {
	if len(changes) == 0 {
		return nil // nothing to say; do not nag
	}
	if s.JSON {
		enc := json.NewEncoder(s.W)
		enc.SetIndent("", "  ")
		return enc.Encode(changes)
	}
	for _, c := range changes {
		switch c.Kind {
		case diff.New:
			fmt.Fprintf(s.W, "[NEW]   %s  s=%d  %s\n", c.Entry.ID(), c.Entry.Seeders, c.Entry.Title)
		case diff.Dropped:
			fmt.Fprintf(s.W, "[DROP]  %s  was rank %d  %s\n", c.Entry.ID(), c.PrevRank, c.Entry.Title)
		case diff.RankMove:
			fmt.Fprintf(s.W, "[RANK]  %s  %d→%d  %s\n", c.Entry.ID(), c.PrevRank, c.Entry.Rank, c.Entry.Title)
		case diff.SeedShift:
			fmt.Fprintf(s.W, "[SEED]  %s  %d→%d  %s\n", c.Entry.ID(), c.PrevSeeders, c.Entry.Seeders, c.Entry.Title)
		}
	}
	return nil
}
