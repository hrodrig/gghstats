package contrib

import _ "embed"

//go:embed gghstats.env.example
var sampleEnv []byte

// SampleEnv returns the annotated environment template (same as contrib/gghstats.env.example).
func SampleEnv() string {
	return string(sampleEnv)
}
