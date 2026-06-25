// Package collector sends anonymous, non-identifying usage statistics to help
// improve gghstats and checks for new releases. No credentials, hostnames,
// repository names, or file paths are ever transmitted — only boolean feature
// flags and a one-way hash for deduplication.
package collector

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/version"
)

// repoLatestReleaseURL is the GitHub API endpoint used to check for updates.
const repoLatestReleaseURL = "https://api.github.com/repos/hrodrig/gghstats/releases/latest"

// collectorURL is the endpoint that receives anonymous usage reports.
const collectorURL = "https://collect.gghstats.com/c5f7a2e1d9b83460"

// data is the JSON payload sent to the collector.
type data struct {
	Version   string   `json:"version"`
	Commit    string   `json:"commit"`
	BuildDate string   `json:"build_date"`
	Hash      string   `json:"hash"`
	Features  features `json:"features"`
}

// features holds only on/off flags — no values, paths, or identifiers.
type features struct {
	BadgePublic      bool `json:"badge_public"`
	MetricsEnabled   bool `json:"metrics_enabled"`
	MetricsPerRepo   bool `json:"metrics_per_repo"`
	SyncOnStartup    bool `json:"sync_on_startup"`
	HasAPIToken      bool `json:"has_api_token"`
	HasPublicURL     bool `json:"has_public_url"`
	HasCustomCSS     bool `json:"has_custom_css"`
	RateLimitEnabled bool `json:"rate_limit_enabled"`
	PortCustom       bool `json:"port_custom"`
	HostCustom       bool `json:"host_custom"`
}

// ServeFeatures is the subset of serve configuration relevant for anonymous
// collection. All fields are booleans derived from the actual config values so
// that no real data leaves the machine.
type ServeFeatures struct {
	BadgePublic      bool
	MetricsEnabled   bool
	MetricsPerRepo   bool
	SyncOnStartup    bool
	HasAPIToken      bool
	HasPublicURL     bool
	HasCustomCSS     bool
	RateLimitEnabled bool
	Port             string
	Host             string
}

// Collect sends one anonymous usage report. Errors are logged but never
// returned — collection must never disrupt the running server.
func Collect(cfg ServeFeatures) {
	buf, err := createBody(cfg)
	if err != nil {
		slog.Debug("collector: failed to create body", "error", err)
		return
	}

	client := makeHTTPClient()
	client.Timeout = 15 * time.Second

	// Log the exact payload at debug level so users can verify what is sent.
	slog.Debug("collector: sending anonymous usage report", "payload", buf.String())

	resp, err := client.Post(collectorURL, "application/json; charset=utf-8", buf)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		slog.Debug("collector: send failed", "error", err)
	} else {
		slog.Debug("collector: anonymous stats sent")
	}
}

// CollectWithUpdate sends an anonymous usage report and checks for newer
// releases in a single call. Equivalent to calling Collect then CheckUpdate.
func CollectWithUpdate(cfg ServeFeatures) {
	Collect(cfg)
	CheckUpdate()
}

func createBody(cfg ServeFeatures) (*bytes.Buffer, error) {
	hash := hashConfig(cfg)

	f := features{
		BadgePublic:      cfg.BadgePublic,
		MetricsEnabled:   cfg.MetricsEnabled,
		MetricsPerRepo:   cfg.MetricsPerRepo,
		SyncOnStartup:    cfg.SyncOnStartup,
		HasAPIToken:      cfg.HasAPIToken,
		HasPublicURL:     cfg.HasPublicURL,
		HasCustomCSS:     cfg.HasCustomCSS,
		RateLimitEnabled: cfg.RateLimitEnabled,
		PortCustom:       cfg.Port != "" && cfg.Port != "8080",
		HostCustom:       cfg.Host != "" && cfg.Host != "127.0.0.1",
	}

	d := &data{
		Version:   version.Version,
		Commit:    version.Commit,
		BuildDate: version.BuildDate,
		Hash:      hash,
		Features:  f,
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(d); err != nil {
		return nil, err
	}
	return buf, nil
}

// hashConfig produces a short, one-way fingerprint of the full serve
// configuration for deduplication. It is not reversible.
func hashConfig(cfg ServeFeatures) string {
	h := sha256.New()
	fmt.Fprintf(h, "%v", cfg) // %v of struct → field:value pairs, order-stable
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func makeHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{Transport: transport}
}

// CheckUpdate queries the GitHub API for the latest release and logs a warning
// if the running version is older. It creates its own HTTP client so it can be
// called independently of Collect.
func CheckUpdate() {
	client := makeHTTPClient()
	client.Timeout = 15 * time.Second
	checkUpdate(client)
}

// checkUpdate queries the GitHub API for the latest release and logs a warning
// if the running version is older.
func checkUpdate(client *http.Client) {
	req, err := http.NewRequest("GET", repoLatestReleaseURL, nil)
	if err != nil {
		slog.Debug("collector: update check request failed", "error", err)
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gghstats/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("collector: update check request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	// Decode only the tag_name field from the release response.
	var latest struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		slog.Debug("collector: update check decode failed", "error", err)
		return
	}

	latestVer := strings.TrimPrefix(latest.TagName, "v")
	if latestVer == "" || latestVer == version.Version {
		return
	}

	if semverGT(latestVer, version.Version) {
		slog.Warn("A new release of gghstats has been found: " + latest.TagName +
			". Please consider upgrading (current: v" + version.Version + ").")
	}
}

// semverGT returns true if a > b for MAJOR.MINOR.PATCH semantic versions.
// Non-numeric or differently sized segments fall back to string comparison.
func semverGT(a, b string) bool {
	pa := parseSemver(a)
	pb := parseSemver(b)
	if len(pa) == 0 || len(pb) == 0 {
		return a > b
	}
	n := len(pa)
	if len(pb) < n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		if pa[i] > pb[i] {
			return true
		}
		if pa[i] < pb[i] {
			return false
		}
	}
	return len(pa) > len(pb)
}

func parseSemver(v string) []int {
	parts := strings.SplitN(v, ".", 3)
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		var n int
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return nil
			}
			n = n*10 + int(ch-'0')
		}
		out = append(out, n)
	}
	return out
}
