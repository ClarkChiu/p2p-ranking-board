package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/freebooters/p2p-ranking-board/internal/diff"
	"github.com/freebooters/p2p-ranking-board/internal/notify"
	"github.com/freebooters/p2p-ranking-board/internal/resolve"
	"github.com/freebooters/p2p-ranking-board/internal/snapshot"
	"github.com/freebooters/p2p-ranking-board/internal/source"
)

func main() {
	root := &cobra.Command{
		Use:   "p2p-ranking-board",
		Short: "Track The Pirate Bay rankings and report what changed.",
		Long: "A stateless one-shot tool meant to be driven by an external scheduler (e.g. a\n" +
			"Hermes cron). `poll` fetches the tracked top-100 lists, diffs against the last\n" +
			"snapshot, and prints what changed; `get` resolves an approved entry to its magnet.",
	}
	root.AddCommand(pollCmd(), listCmd(), getCmd())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "p2p-ranking-board:", err)
		os.Exit(1)
	}
}

func pollCmd() *cobra.Command {
	var (
		state     string
		jsonOut   bool
		only      []string
		threshold float64
		timeout   time.Duration
	)
	cmd := &cobra.Command{
		Use:   "poll",
		Short: "Fetch rankings, diff against the last snapshot, report changes.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if state == "" {
				state = snapshot.DefaultPath()
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			current, err := source.FetchAll(ctx, source.Categories)
			if err != nil {
				return err // snapshot intentionally left untouched
			}
			fmt.Fprintf(os.Stderr, "→ fetched %d entries across %d categories\n", len(current), len(source.Categories))

			prev, existed, err := snapshot.Load(state)
			if err != nil {
				return err
			}

			if !existed {
				fmt.Fprintln(os.Stderr, "  no prior snapshot — establishing baseline, no changes reported")
			} else {
				changes := notify.Filter(diff.Compare(prev.Entries, current, threshold), only)
				fmt.Fprintf(os.Stderr, "  %d change(s) after filter\n", len(changes))
				n := notify.Stdout{W: os.Stdout, JSON: jsonOut}
				if err := n.Notify(ctx, changes); err != nil {
					return err
				}
			}

			return snapshot.Save(state, &snapshot.Snapshot{TakenAt: time.Now(), Entries: current})
		},
	}
	cmd.Flags().StringVar(&state, "state", "", "snapshot file path (default XDG state dir)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit changes as JSON (for an LLM/agent to read)")
	cmd.Flags().StringSliceVar(&only, "only", nil, "report only these kinds: new,dropped,rank_move,seed_shift")
	cmd.Flags().Float64Var(&threshold, "seed-threshold", 0.20, "fractional seeder change to report a seed_shift")
	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "overall fetch budget")
	return cmd
}

var categoryNames = map[int]string{
	207: "HD 電影", 208: "HD 影集", 300: "應用程式", 301: "Windows",
	303: "Linux/UNIX", 401: "PC 遊戲", 601: "電子書",
}

func listCmd() *cobra.Command {
	var (
		top     int
		timeout time.Duration
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Print the current top entries per tracked category (no diff).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			for _, cat := range source.Categories {
				entries, err := source.FetchCategory(ctx, cat)
				if err != nil {
					return err
				}
				fmt.Printf("\n# %d %s — top %d / %d\n", cat, categoryNames[cat], min(top, len(entries)), len(entries))
				for _, e := range entries {
					if e.Rank > top {
						break
					}
					fmt.Printf("  %2d. s=%-5d %s  [%s]\n", e.Rank, e.Seeders, e.Title, e.ID())
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&top, "top", "n", 10, "entries to show per category")
	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "overall fetch budget")
	return cmd
}

func getCmd() *cobra.Command {
	var state string
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Print the magnet for an approved ranking id (pipe it to a downloader).",
		Long: "Resolves a ranking id (from poll/list output) to its magnet and prints it to\n" +
			"stdout — nothing else. Verifying or downloading is up to whatever you pipe it to.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if state == "" {
				state = snapshot.DefaultPath()
			}
			snap, existed, err := snapshot.Load(state)
			if err != nil {
				return err
			}
			if !existed {
				return fmt.Errorf("no snapshot at %s — run `poll` first", state)
			}
			entry, err := resolve.ByID(snap, args[0])
			if err != nil {
				return err
			}
			fmt.Println(resolve.Magnet(entry))
			return nil
		},
	}
	cmd.Flags().StringVar(&state, "state", "", "snapshot file path (default XDG state dir)")
	return cmd
}
