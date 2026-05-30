package diff

import (
	"testing"

	"github.com/freebooters/p2p-ranking-board/internal/source"
)

// entry builds a minimal Entry; only the fields Compare keys on (Category,
// InfoHash, Rank, Seeders) matter here.
func entry(hash string, rank, seeders int) source.Entry {
	return source.Entry{InfoHash: hash, Category: 207, Rank: rank, Seeders: seeders}
}

func snap(es ...source.Entry) map[string]source.Entry {
	m := map[string]source.Entry{}
	for _, e := range es {
		m[e.Key()] = e
	}
	return m
}

// kinds counts how many changes of each kind were produced — lets tests assert
// on intent ("a new arrival is reported as New") rather than slice ordering.
func kinds(cs []Change) map[Kind]int {
	out := map[Kind]int{}
	for _, c := range cs {
		out[c.Kind]++
	}
	return out
}

func TestCompare_baselineFromEmptyIsAllNew(t *testing.T) {
	// WHY: the first poll has no prior snapshot. Everything currently charting
	// must read as New, never as a spurious move/drop, so the baseline run can
	// choose to suppress notifications without losing real arrivals later.
	cur := snap(entry("a", 1, 10), entry("b", 2, 5))
	got := kinds(Compare(nil, cur, 0.20))
	if got[New] != 2 || len(got) != 1 {
		t.Fatalf("empty→cur should be 2 New only, got %v", got)
	}
}

func TestCompare_newAndDropped(t *testing.T) {
	// WHY: a torrent entering the chart is actionable (maybe download it); one
	// leaving is informational. They must be distinguished, not lumped together.
	prev := snap(entry("a", 1, 10))
	cur := snap(entry("b", 1, 10))
	got := kinds(Compare(prev, cur, 0.20))
	if got[New] != 1 || got[Dropped] != 1 {
		t.Fatalf("want 1 New + 1 Dropped, got %v", got)
	}
}

func TestCompare_rankMoveCarriesPrevRank(t *testing.T) {
	// WHY: "moved 5→1" is the interesting signal; without PrevRank the change is
	// useless to a human or an LLM deciding whether it's worth a ping.
	prev := snap(entry("a", 5, 10))
	cur := snap(entry("a", 1, 10))
	cs := Compare(prev, cur, 0.20)
	if len(cs) != 1 || cs[0].Kind != RankMove || cs[0].PrevRank != 5 || cs[0].Entry.Rank != 1 {
		t.Fatalf("want RankMove 5→1, got %+v", cs)
	}
}

func TestCompare_seedShiftRespectsThreshold(t *testing.T) {
	// WHY: seeder counts jitter constantly. Only a move past the threshold is a
	// real signal; sub-threshold noise must stay silent or notifications are spam.
	prev := snap(entry("a", 1, 100))

	below := kinds(Compare(prev, snap(entry("a", 1, 110)), 0.20)) // +10% < 20%
	if below[SeedShift] != 0 {
		t.Fatalf("+10%% is below 20%% threshold, want no SeedShift, got %v", below)
	}
	above := kinds(Compare(prev, snap(entry("a", 1, 130)), 0.20)) // +30% > 20%
	if above[SeedShift] != 1 {
		t.Fatalf("+30%% exceeds threshold, want 1 SeedShift, got %v", above)
	}
}

func TestCompare_identicalIsSilent(t *testing.T) {
	// WHY: an unchanged chart must yield zero changes, so a scheduled poll that
	// finds nothing new never nags the user.
	s := snap(entry("a", 1, 10), entry("b", 2, 5))
	if cs := Compare(s, s, 0.20); len(cs) != 0 {
		t.Fatalf("identical snapshots should produce no changes, got %v", kinds(cs))
	}
}

func TestCompare_sameHashDifferentCategoryAreDistinct(t *testing.T) {
	// WHY: a torrent can chart in two categories at different ranks. Keying by
	// category+hash keeps their rank moves independent; keying by hash alone
	// would cross-contaminate them.
	prev := map[string]source.Entry{}
	a207 := source.Entry{InfoHash: "x", Category: 207, Rank: 3, Seeders: 10}
	a301 := source.Entry{InfoHash: "x", Category: 301, Rank: 9, Seeders: 10}
	prev[a207.Key()] = a207
	prev[a301.Key()] = a301
	// 207 moves, 301 unchanged.
	cur := map[string]source.Entry{}
	a207b := source.Entry{InfoHash: "x", Category: 207, Rank: 1, Seeders: 10}
	cur[a207b.Key()] = a207b
	cur[a301.Key()] = a301
	got := kinds(Compare(prev, cur, 0.20))
	if got[RankMove] != 1 || len(got) != 1 {
		t.Fatalf("only the 207 listing moved; want 1 RankMove, got %v", got)
	}
}
