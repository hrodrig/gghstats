package report

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/hrodrig/gghstats/internal/store"
)

// Terminal writes a formatted traffic report to w.
func Terminal(w io.Writer, repo string, views, clones []store.DayRow, refs []store.ReferrerRow, paths []store.PathRow) {
	fmt.Fprintf(w, "Traffic report: %s\n", repo)
	fmt.Fprintln(w, strings.Repeat("=", 60))

	printDayTable(w, "Views", views)
	fmt.Fprintln(w)
	printDayTable(w, "Clones", clones)

	if len(refs) > 0 {
		fmt.Fprintln(w)
		printReferrers(w, refs)
	}

	if len(paths) > 0 {
		fmt.Fprintln(w)
		printPaths(w, paths)
	}
}

func printDayTable(w io.Writer, title string, rows []store.DayRow) {
	fmt.Fprintf(w, "\n%s\n", title)
	fmt.Fprintln(w, strings.Repeat("-", 40))

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Date\tCount\tUniques")
	fmt.Fprintln(tw, "----\t-----\t-------")

	totalCount, totalUniques := 0, 0
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%d\t%d\n", r.Date, r.Count, r.Uniques)
		totalCount += r.Count
		totalUniques += r.Uniques
	}

	fmt.Fprintln(tw, "\t\t")
	fmt.Fprintf(tw, "Total\t%d\t%d\n", totalCount, totalUniques)
	tw.Flush()

	if len(rows) > 0 {
		avg := float64(totalCount) / float64(len(rows))
		fmt.Fprintf(w, "Average: %.1f/day over %d days\n", avg, len(rows))
	}
}

func printReferrers(w io.Writer, refs []store.ReferrerRow) {
	fmt.Fprintln(w, "Top Referrers")
	fmt.Fprintln(w, strings.Repeat("-", 40))

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Referrer\tCount\tUniques")
	fmt.Fprintln(tw, "--------\t-----\t-------")

	seen := make(map[string]bool)
	for _, r := range refs {
		if seen[r.Referrer] {
			continue
		}
		seen[r.Referrer] = true
		fmt.Fprintf(tw, "%s\t%d\t%d\n", r.Referrer, r.Count, r.Uniques)
	}
	tw.Flush()
}

func printPaths(w io.Writer, paths []store.PathRow) {
	fmt.Fprintln(w, "Popular Content")
	fmt.Fprintln(w, strings.Repeat("-", 40))

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Path\tCount\tUniques")
	fmt.Fprintln(tw, "----\t-----\t-------")

	seen := make(map[string]bool)
	for _, p := range paths {
		if seen[p.Path] {
			continue
		}
		seen[p.Path] = true
		fmt.Fprintf(tw, "%s\t%d\t%d\n", p.Path, p.Count, p.Uniques)
	}
	tw.Flush()
}
