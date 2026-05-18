package metrics

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func splitRepo(fullName string) (owner, repo string, ok bool) {
	i := strings.Index(fullName, "/")
	if i <= 0 || i >= len(fullName)-1 {
		return "", "", false
	}
	return fullName[:i], fullName[i+1:], true
}

func repoLabelKey(owner, repo string) string {
	return owner + "\x00" + repo
}

// refreshPerRepoGauges updates per-repository gauges from the store (Phase 3).
func (d *Domain) refreshPerRepoGauges() {
	if d == nil || !d.perRepoEnabled || d.listRepos == nil {
		return
	}
	if d.repoStars == nil {
		return
	}

	repos, err := d.listRepos()
	if err != nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	seen := make(map[string]struct{}, len(repos))
	for _, r := range repos {
		owner, name, ok := splitRepo(r.Name)
		if !ok {
			continue
		}
		seen[repoLabelKey(owner, name)] = struct{}{}
		labels := prometheus.Labels{"owner": owner, "repo": name}
		d.repoStars.With(labels).Set(float64(r.Stars))
		d.repoForks.With(labels).Set(float64(r.Forks))
		d.repoClones.With(labels).Set(float64(r.TotalClones))
		d.repoViews.With(labels).Set(float64(r.TotalViews))
		d.repoClones1d.With(labels).Set(float64(r.Clones1d))
		d.repoClones7d.With(labels).Set(float64(r.Clones7d))
		d.repoClones30d.With(labels).Set(float64(r.Clones30d))
	}

	for key := range d.lastRepoLabels {
		if _, ok := seen[key]; ok {
			continue
		}
		owner, name, ok := splitRepoKey(key)
		if !ok {
			continue
		}
		d.repoStars.DeleteLabelValues(owner, name)
		d.repoForks.DeleteLabelValues(owner, name)
		d.repoClones.DeleteLabelValues(owner, name)
		d.repoViews.DeleteLabelValues(owner, name)
		d.repoClones1d.DeleteLabelValues(owner, name)
		d.repoClones7d.DeleteLabelValues(owner, name)
		d.repoClones30d.DeleteLabelValues(owner, name)
	}
	d.lastRepoLabels = seen
}

func splitRepoKey(key string) (owner, repo string, ok bool) {
	owner, repo, ok = strings.Cut(key, "\x00")
	return owner, repo, ok && owner != "" && repo != ""
}
