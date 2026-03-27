package report

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/hrodrig/gghstats/internal/store"
)

// CSV writes traffic data as CSV to w.
func CSV(w io.Writer, repo string, views, clones []store.DayRow, refs []store.ReferrerRow, paths []store.PathRow) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := writeDaySection(cw, "# Views", []string{"date", "views", "unique_visitors"}, views); err != nil {
		return err
	}
	if err := writeDaySection(cw, "# Clones", []string{"date", "clones", "unique_cloners"}, clones); err != nil {
		return err
	}
	if err := writeReferrerSection(cw, refs); err != nil {
		return err
	}
	if err := writePathSection(cw, paths); err != nil {
		return err
	}

	return cw.Error()
}

func writeDaySection(cw *csv.Writer, title string, header []string, rows []store.DayRow) error {
	if err := writeSectionTitleAndHeader(cw, title, header); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write([]string{r.Date, itoa(r.Count), itoa(r.Uniques)}); err != nil {
			return err
		}
	}
	return cw.Write(nil)
}

func writeReferrerSection(cw *csv.Writer, refs []store.ReferrerRow) error {
	if len(refs) == 0 {
		return nil
	}
	if err := writeSectionTitleAndHeader(cw, "# Referrers", []string{"date", "referrer", "count", "uniques"}); err != nil {
		return err
	}
	for _, r := range refs {
		if err := cw.Write([]string{r.Date, r.Referrer, itoa(r.Count), itoa(r.Uniques)}); err != nil {
			return err
		}
	}
	return cw.Write(nil)
}

func writePathSection(cw *csv.Writer, paths []store.PathRow) error {
	if len(paths) == 0 {
		return nil
	}
	if err := writeSectionTitleAndHeader(cw, "# Popular Paths", []string{"date", "path", "title", "count", "uniques"}); err != nil {
		return err
	}
	for _, p := range paths {
		if err := cw.Write([]string{p.Date, p.Path, p.Title, itoa(p.Count), itoa(p.Uniques)}); err != nil {
			return err
		}
	}
	return nil
}

func writeSectionTitleAndHeader(cw *csv.Writer, title string, header []string) error {
	if err := cw.Write([]string{title}); err != nil {
		return err
	}
	return cw.Write(header)
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
