package contrib

import (
	"strings"
	"testing"
)

func TestSampleEnv(t *testing.T) {
	s := SampleEnv()
	if !strings.Contains(s, "GGHSTATS_GITHUB_TOKEN=") {
		t.Fatal("missing GGHSTATS_GITHUB_TOKEN")
	}
	if !strings.Contains(s, "/etc/gghstats/gghstats.env") {
		t.Fatal("missing production path hint")
	}
}
