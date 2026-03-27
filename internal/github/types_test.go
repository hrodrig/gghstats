package github

import (
	"encoding/json"
	"testing"
)

func TestRepoJSONParent(t *testing.T) {
	const payload = `{
		"full_name": "you/book",
		"fork": true,
		"parent": { "full_name": "rust-lang/book" }
	}`
	var r Repo
	if err := json.Unmarshal([]byte(payload), &r); err != nil {
		t.Fatal(err)
	}
	if r.FullName != "you/book" || !r.Fork {
		t.Fatalf("basic fields: %+v", r)
	}
	if got := r.ParentFullName(); got != "rust-lang/book" {
		t.Errorf("ParentFullName() = %q", got)
	}
}

func TestRepoJSONNoParent(t *testing.T) {
	const payload = `{"full_name": "o/r", "fork": false}`
	var r Repo
	if err := json.Unmarshal([]byte(payload), &r); err != nil {
		t.Fatal(err)
	}
	if r.ParentFullName() != "" {
		t.Errorf("want empty parent, got %q", r.ParentFullName())
	}
}
