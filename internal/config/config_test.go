package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultLogFileResolvedAtRuntime(t *testing.T) {
	// The default config must NOT bake in an absolute log path: empty means
	// "resolve DefaultLogPath() at runtime". A stored absolute path goes stale
	// across snap refreshes (it contains the revision directory) and broke
	// startup on the first refresh (0.1.0 -> 0.2.0).
	if file := Default().Logging.File; file != "" {
		t.Errorf("Default().Logging.File = %q, want empty (runtime-resolved)", file)
	}
}

func TestExtendedHoursDefaultsOn(t *testing.T) {
	if !Default().ExtendedHours {
		t.Error("Default().ExtendedHours = false, want true")
	}

	// Load unmarshals into Default(), so a pre-existing config file without
	// the key must keep the on-by-default behavior...
	cfg := Default()
	if err := json.Unmarshal([]byte(`{"symbols":["AAPL"]}`), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !cfg.ExtendedHours {
		t.Error("config without extendedHours key lost the default true")
	}

	// ...while an explicit false is honored.
	cfg = Default()
	if err := json.Unmarshal([]byte(`{"extendedHours":false}`), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.ExtendedHours {
		t.Error("explicit extendedHours=false was not honored")
	}

	// The field round-trips through the persisted JSON shape.
	data, err := json.Marshal(Config{ExtendedHours: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundTrip Config
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !roundTrip.ExtendedHours {
		t.Error("extendedHours did not round-trip")
	}
}
