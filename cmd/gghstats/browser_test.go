package main

import "testing"

func TestServeDashboardURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		host, port, want string
	}{
		{"127.0.0.1", "8080", "http://127.0.0.1:8080"},
		{"0.0.0.0", "9090", "http://127.0.0.1:9090"},
		{"::", "8080", "http://127.0.0.1:8080"},
		{"[::]", "3000", "http://127.0.0.1:3000"},
	}
	for _, tc := range tests {
		if got := serveDashboardURL(tc.host, tc.port); got != tc.want {
			t.Errorf("serveDashboardURL(%q, %q) = %q, want %q", tc.host, tc.port, got, tc.want)
		}
	}
}

func TestServeOpenURL(t *testing.T) {
	t.Parallel()
	if got := serveOpenURL("127.0.0.1", "8080", false); got != "http://127.0.0.1:8080" {
		t.Errorf("dashboard open = %q", got)
	}
	if got := serveOpenURL("0.0.0.0", "8080", true); got != "http://127.0.0.1:8080/api/v1/healthz" {
		t.Errorf("api-only open = %q", got)
	}
}
