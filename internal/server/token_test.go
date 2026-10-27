package server

import "testing"

func TestTokenMatches(t *testing.T) {
	tests := []struct {
		name       string
		presented  string
		configured string
		want       bool
	}{
		{name: "exact match", presented: "s3cret", configured: "s3cret", want: true},
		{name: "wrong token", presented: "s3crey", configured: "s3cret", want: false},
		{name: "prefix of the token", presented: "s3cr", configured: "s3cret", want: false},
		{name: "configured token is a prefix", presented: "s3cretx", configured: "s3cret", want: false},
		{name: "empty presented token", presented: "", configured: "s3cret", want: false},
		{name: "empty configured token never matches", presented: "", configured: "", want: false},
		{name: "empty configured token with a presented token", presented: "s3cret", configured: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tokenMatches(tt.presented, tt.configured); got != tt.want {
				t.Errorf("tokenMatches(%q, %q) = %v, want %v", tt.presented, tt.configured, got, tt.want)
			}
		})
	}
}
