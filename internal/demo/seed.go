// Package demo seeds sample SQLite data for local UI evaluation without a GitHub token.
package demo

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

// SeedIfEmpty inserts sample repositories and traffic when the DB has no repos.
// Safe to call on every demo startup; no-op when data already exists.
func SeedIfEmpty(db *store.Store) error {
	has, err := db.HasRepos()
	if err != nil {
		return fmt.Errorf("demo seed: %w", err)
	}
	if has {
		slog.Info("demo mode: using existing database (skip seed)")
		return nil
	}
	if err := Seed(db); err != nil {
		return err
	}
	slog.Info("demo mode: seeded sample repositories")
	return nil
}

// Seed writes a fixed sample dataset suitable for index, repo charts, trends, and H2H.
func Seed(db *store.Store) error {
	type repoSpec struct {
		name, desc             string
		stars, forks, watchers int
		cloneBase, viewBase    int
		cloneStep              int
	}
	repos := []repoSpec{
		{name: "demo/alpha", desc: "Sample rising traffic (demo)", stars: 42, forks: 3, watchers: 10, cloneBase: 8, viewBase: 20, cloneStep: 1},
		{name: "demo/beta", desc: "Sample steady traffic (demo)", stars: 18, forks: 1, watchers: 5, cloneBase: 12, viewBase: 15, cloneStep: 0},
		{name: "demo/gamma", desc: "Sample declining traffic (demo)", stars: 7, forks: 0, watchers: 2, cloneBase: 25, viewBase: 30, cloneStep: -1},
	}

	today := time.Now().UTC()
	for _, r := range repos {
		if err := db.UpsertRepo(r.name, r.desc, r.stars, r.forks, r.watchers, 0, 0, false, false, ""); err != nil {
			return fmt.Errorf("demo seed repo %s: %w", r.name, err)
		}
		// 60 days so 30d momentum has a previous window.
		for i := 0; i < 60; i++ {
			d := today.AddDate(0, 0, -i).Format("2006-01-02")
			age := 59 - i // older days first in growth curve
			clones := r.cloneBase + r.cloneStep*age/2
			if clones < 1 {
				clones = 1
			}
			views := r.viewBase + r.cloneStep*age/3
			if views < 1 {
				views = 1
			}
			uniqC := clones/2 + 1
			uniqV := views/2 + 1
			if err := db.UpsertClone(r.name, d, clones, uniqC); err != nil {
				return fmt.Errorf("demo seed clones %s: %w", r.name, err)
			}
			if err := db.UpsertView(r.name, d, views, uniqV); err != nil {
				return fmt.Errorf("demo seed views %s: %w", r.name, err)
			}
		}
		starDay := today.Format("2006-01-02")
		if err := db.UpsertStar(r.name, starDay, r.stars); err != nil {
			return fmt.Errorf("demo seed stars %s: %w", r.name, err)
		}
	}
	if err := db.UpdateDeltas(); err != nil {
		return fmt.Errorf("demo seed deltas: %w", err)
	}
	return nil
}
