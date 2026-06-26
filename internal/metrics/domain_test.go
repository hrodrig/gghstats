package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDomainNilSafe(t *testing.T) {
	var d *Domain
	d.ObserveGitHubRequest("repos", "success")
	d.SetGitHubRateLimitRemaining(1)
	d.SetGitHubRateLimitReset(1700000000)
	d.ObserveSync(time.Second, true)
	d.ObserveSyncRepo("success")
	d.RefreshStoreGauges()
}

func TestRegisterDomainDefaultFilter(t *testing.T) {
	reg := prometheus.NewRegistry()
	d := RegisterDomain(reg, DomainConfig{})
	if d.filter != "*" {
		t.Fatalf("filter = %q, want *", d.filter)
	}
}

func TestDomainObserveGitHubAndSync(t *testing.T) {
	reg := prometheus.NewRegistry()
	d := RegisterDomain(reg, DomainConfig{
		Filter: "*",
		StoreRepoCount: func() (int, error) {
			return 2, nil
		},
	})

	d.ObserveGitHubRequest("repos", "success")
	d.SetGitHubRateLimitRemaining(99)
	d.SetGitHubRateLimitReset(1700000000)
	d.ObserveSync(3*time.Second, true)
	d.ObserveSync(time.Second, false)
	d.ObserveSyncRepo("success")
	d.ObserveSyncRepo("error")
	d.ObserveSyncRepo("error")

	cnt, err := d.githubRequests.GetMetricWithLabelValues("repos", "success")
	if err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(cnt); v != 1 {
		t.Fatalf("github requests = %v, want 1", v)
	}
	rem, err := d.githubRateRem.GetMetricWithLabelValues("core")
	if err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(rem); v != 99 {
		t.Fatalf("rate limit remaining = %v, want 99", v)
	}
	rst, err := d.githubRateRst.GetMetricWithLabelValues("core")
	if err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(rst); v != 1700000000 {
		t.Fatalf("rate limit reset = %v, want 1700000000", v)
	}
	if !metricFamilyHasSample(reg, "gghstats_sync_duration_seconds", "status", "success") {
		t.Fatal("expected success sync duration observation")
	}
	if !metricFamilyHasSample(reg, "gghstats_sync_duration_seconds", "status", "error") {
		t.Fatal("expected error sync duration observation")
	}
	if v := testutil.ToFloat64(d.lastSyncTS); v <= 0 {
		t.Fatalf("last sync ts = %v, want > 0", v)
	}

	// sync_repos_processed_total
	successCnt, err := d.syncReposProc.GetMetricWithLabelValues("success")
	if err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(successCnt); v != 1 {
		t.Fatalf("sync_repos_processed_total{success} = %v, want 1", v)
	}
	errorCnt, err := d.syncReposProc.GetMetricWithLabelValues("error")
	if err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(errorCnt); v != 2 {
		t.Fatalf("sync_repos_processed_total{error} = %v, want 2", v)
	}
}

func TestDomainRefreshStoreGauges(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "metrics.db")
	if err := os.WriteFile(dbPath, []byte("sqlite-placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}

	reg := prometheus.NewRegistry()
	d := RegisterDomain(reg, DomainConfig{
		Filter: "org/*",
		DBPath: dbPath,
		StoreRepoCount: func() (int, error) {
			return 5, nil
		},
	})
	d.RefreshStoreGauges()

	total, err := d.reposTotal.GetMetricWithLabelValues("org/*")
	if err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(total); v != 5 {
		t.Fatalf("repos_total = %v, want 5", v)
	}
	if v := testutil.ToFloat64(d.dbSizeBytes); v != float64(len("sqlite-placeholder")) {
		t.Fatalf("db_size_bytes = %v, want %d", v, len("sqlite-placeholder"))
	}
}

func metricFamilyHasSample(reg *prometheus.Registry, name, labelName, labelValue string) bool {
	mfs, err := reg.Gather()
	if err != nil {
		return false
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == labelName && lp.GetValue() == labelValue && m.GetHistogram().GetSampleCount() > 0 {
					return true
				}
			}
		}
	}
	return false
}

func TestPerRepoDisabledNoOp(t *testing.T) {
	reg := prometheus.NewRegistry()
	d := RegisterDomain(reg, DomainConfig{PerRepoEnabled: false})
	d.refreshPerRepoGauges() // should return immediately
	if d.repoStars != nil {
		t.Fatal("per-repo gauges should not be registered")
	}
}

func TestDomainObserveSyncError(t *testing.T) {
	reg := prometheus.NewRegistry()
	d := RegisterDomain(reg, DomainConfig{})

	// nil-safe
	var nilD *Domain
	nilD.ObserveSyncError("ignored")
	d.ObserveSyncError("")
	d.ObserveSyncError("views")
	d.ObserveSyncError("views")
	d.ObserveSyncError("clones")

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"views": 2, "clones": 1}
	got := map[string]float64{}
	for _, mf := range mfs {
		if mf.GetName() != "gghstats_sync_errors_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "kind" {
					got[lp.GetValue()] = m.GetCounter().GetValue()
				}
			}
		}
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("kind=%s got %v want %v", k, got[k], v)
		}
	}
}
