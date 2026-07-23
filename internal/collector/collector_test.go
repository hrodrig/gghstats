package collector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hrodrig/gghstats/internal/version"
)

func TestCreateBody(t *testing.T) {
	t.Parallel()

	cfg := ServeFeatures{
		BadgePublic:      true,
		MetricsEnabled:   false,
		MetricsPerRepo:   true,
		SyncOnStartup:    false,
		HasAPIToken:      true,
		HasPublicURL:     false,
		HasCustomCSS:     true,
		RateLimitEnabled: false,
		Port:             "9090",
		Host:             "0.0.0.0",
	}

	buf, err := createBody(cfg)
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("createBody returned empty buffer")
	}

	var d data
	if err := json.NewDecoder(buf).Decode(&d); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if d.Features.BadgePublic != true {
		t.Errorf("BadgePublic = %v, want true", d.Features.BadgePublic)
	}
	if d.Features.MetricsEnabled != false {
		t.Errorf("MetricsEnabled = %v, want false", d.Features.MetricsEnabled)
	}
	if d.Features.PortCustom != true {
		t.Errorf("PortCustom = %v, want true (9090 ≠ 8080)", d.Features.PortCustom)
	}
	if d.Features.HostCustom != true {
		t.Errorf("HostCustom = %v, want true (0.0.0.0 ≠ 127.0.0.1)", d.Features.HostCustom)
	}
	if d.Hash == "" {
		t.Error("Hash is empty")
	}
}

func TestCreateBodyDefaults(t *testing.T) {
	t.Parallel()

	cfg := ServeFeatures{
		Port: "8080",
		Host: "127.0.0.1",
	}

	buf, err := createBody(cfg)
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}

	var d data
	if err := json.NewDecoder(buf).Decode(&d); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if d.Features.PortCustom {
		t.Error("PortCustom should be false for default port 8080")
	}
	if d.Features.HostCustom {
		t.Error("HostCustom should be false for default host 127.0.0.1")
	}
}

func TestHashConfigDeterministic(t *testing.T) {
	t.Parallel()

	cfg := ServeFeatures{
		BadgePublic:   true,
		SyncOnStartup: false,
		Port:          "3000",
		Host:          "127.0.0.1",
	}

	h1 := hashConfig(cfg)
	h2 := hashConfig(cfg)
	if h1 != h2 {
		t.Errorf("hash should be deterministic: %q ≠ %q", h1, h2)
	}
}

func TestHashConfigDifferent(t *testing.T) {
	t.Parallel()

	a := ServeFeatures{BadgePublic: true}
	b := ServeFeatures{BadgePublic: false}

	if hashConfig(a) == hashConfig(b) {
		t.Error("different configs should produce different hashes")
	}
}

func TestCollectSendsPayload(t *testing.T) {
	t.Parallel()

	var received bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	// Override the URL for the test by calling createBody + POST directly
	cfg := ServeFeatures{BadgePublic: true, Port: "8080", Host: "127.0.0.1"}
	buf, err := createBody(cfg)
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}

	resp, err := http.Post(srv.URL, "application/json; charset=utf-8", buf)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	resp.Body.Close()

	if !received {
		t.Error("server did not receive payload")
	}
}

func TestCollectAndCheckUpdate(t *testing.T) {
	var gotCollect, gotUpdate bool
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			gotCollect = true
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			gotUpdate = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v99.0.0"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	prevCollect, prevUpdate := collectEndpoint, updateCheckURL
	collectEndpoint, updateCheckURL = srv.URL, srv.URL
	t.Cleanup(func() {
		collectEndpoint, updateCheckURL = prevCollect, prevUpdate
	})

	CollectWithUpdate(ServeFeatures{Port: "8080", Host: "127.0.0.1"})
	if !gotCollect {
		t.Fatal("Collect did not POST to collect endpoint")
	}
	if !gotUpdate {
		t.Fatal("CheckUpdate did not GET release endpoint")
	}
}

func TestCheckUpdateSameVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v` + version.Version + `"}`))
	}))
	t.Cleanup(srv.Close)

	prev := updateCheckURL
	updateCheckURL = srv.URL
	t.Cleanup(func() { updateCheckURL = prev })

	CheckUpdate() // same version — no warn path beyond decode
}

func TestSemverGT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "0.9.9", true},
		{"0.9.9", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"1.0.1", "1.0.0", true},
		{"1.1.0", "1.0.9", true},
		{"2.0.0", "1.9.9", true},
		{"0.7.10", "0.7.9", true},
		{"0.7.9", "0.7.10", false},
		{"0.7.10", "0.7.10", false},
		{"0.8.0", "0.7.10", true},
		{"1.0.0", "0.99.99", true},
		// Non-numeric suffix (pre-release) falls back to string compare
		{"1.0.0", "1.0.0-alpha", false},
		// Same version strings
		{"dev", "dev", false},
	}

	for _, tc := range tests {
		got := semverGT(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("semverGT(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
