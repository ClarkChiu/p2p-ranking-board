package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const topListURL = "https://apibay.org/precompiled/data_top100_%d.json"

var httpClient = &http.Client{Timeout: 20 * time.Second}

// rawRow tolerates apibay's inconsistent typing: the top-100 precompiled files
// encode numbers as JSON ints, while q.php encodes them as strings. json.Number
// accepts both.
type rawRow struct {
	Name     string      `json:"name"`
	InfoHash string      `json:"info_hash"`
	Category json.Number `json:"category"`
	Seeders  json.Number `json:"seeders"`
	Leechers json.Number `json:"leechers"`
	Size     json.Number `json:"size"`
	Added    json.Number `json:"added"`
}

const zeroHash = "0000000000000000000000000000000000000000"

// FetchCategory pulls one category's top-100 list and normalizes it. Rank is the
// listing order (1-based).
func FetchCategory(ctx context.Context, cat int) ([]Entry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(topListURL, cat), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("category %d: %w", cat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("category %d: http %d", cat, resp.StatusCode)
	}

	var rows []rawRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("category %d: decode: %w", cat, err)
	}

	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		hash := strings.ToLower(strings.TrimSpace(r.InfoHash))
		if hash == "" || hash == zeroHash {
			continue // sentinel / invalid row
		}
		out = append(out, Entry{
			InfoHash:  hash,
			Title:     r.Name,
			Category:  cat,
			Rank:      len(out) + 1,
			Seeders:   num(r.Seeders),
			Leechers:  num(r.Leechers),
			SizeBytes: num64(r.Size),
			Added:     epoch(r.Added),
		})
	}
	return out, nil
}

// FetchAll pulls every tracked category concurrently and returns the combined
// map keyed by Entry.Key. If ANY category fails, it returns an error and no map —
// a partial snapshot would corrupt the diff against the previous full snapshot.
func FetchAll(ctx context.Context, cats []int) (map[string]Entry, error) {
	type res struct {
		entries []Entry
		err     error
	}
	results := make([]res, len(cats))
	var wg sync.WaitGroup
	for i, cat := range cats {
		wg.Add(1)
		go func(i, cat int) {
			defer wg.Done()
			e, err := FetchCategory(ctx, cat)
			results[i] = res{e, err}
		}(i, cat)
	}
	wg.Wait()

	merged := map[string]Entry{}
	var failed []string
	for i, r := range results {
		if r.err != nil {
			failed = append(failed, fmt.Sprintf("%d (%v)", cats[i], r.err))
			continue
		}
		for _, e := range r.entries {
			merged[e.Key()] = e
		}
	}
	if len(failed) > 0 {
		return nil, fmt.Errorf("category fetch failed: %s — refusing to overwrite snapshot", strings.Join(failed, ", "))
	}
	return merged, nil
}

func num(n json.Number) int {
	v, _ := strconv.Atoi(strings.TrimSpace(n.String()))
	return v
}

func num64(n json.Number) int64 {
	v, _ := strconv.ParseInt(strings.TrimSpace(n.String()), 10, 64)
	return v
}

func epoch(n json.Number) time.Time {
	sec, err := strconv.ParseInt(strings.TrimSpace(n.String()), 10, 64)
	if err != nil || sec <= 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}
