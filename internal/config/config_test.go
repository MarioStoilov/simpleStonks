package config

import "testing"

func TestDefaultLogFileResolvedAtRuntime(t *testing.T) {
	// The default config must NOT bake in an absolute log path: empty means
	// "resolve DefaultLogPath() at runtime". A stored absolute path goes stale
	// across snap refreshes (it contains the revision directory) and broke
	// startup on the first refresh (0.1.0 -> 0.2.0).
	if f := Default().Logging.File; f != "" {
		t.Errorf("Default().Logging.File = %q, want empty (runtime-resolved)", f)
	}
}
