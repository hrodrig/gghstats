package github

import (
	"net/http"
	"testing"
)

type spyMetrics struct {
	endpoint  string
	status    string
	remaining int
	calls     int
}

func (s *spyMetrics) ObserveGitHubRequest(endpoint, status string) {
	s.calls++
	s.endpoint = endpoint
	s.status = status
}

func (s *spyMetrics) SetGitHubRateLimitRemaining(remaining int) {
	s.remaining = remaining
}

func TestClientRecordResponse(t *testing.T) {
	spy := &spyMetrics{}
	c := NewClient("token")
	c.SetMetrics(spy)

	okResp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header)}
	okResp.Header.Set("X-RateLimit-Remaining", "12")
	c.recordResponse("/repos/o/r/traffic/views", okResp, nil)
	if spy.calls != 1 || spy.endpoint != "traffic_views" || spy.status != "success" || spy.remaining != 12 {
		t.Fatalf("spy = %+v, want traffic_views success remaining 12", spy)
	}

	spy.calls = 0
	c.recordResponse("/repos/o/r", nil, http.ErrServerClosed)
	if spy.endpoint != "repos" || spy.status != "error" {
		t.Fatalf("error path spy = %+v", spy)
	}

	c.SetMetrics(nil)
	c.recordResponse("/repos/o/r", nil, nil)
	if spy.calls != 1 {
		t.Fatal("nil metrics should not record")
	}
}
