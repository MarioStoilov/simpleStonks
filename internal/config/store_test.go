package config

import (
	"testing"
	"time"
)

// waitFor blocks until fn returns true or the timeout elapses.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

func TestStoreLiveReloadOnExternalEdit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	notified := make(chan Config, 4)
	store.Subscribe(func(cfg Config) { notified <- cfg })

	// Simulate an external edit: write a new config directly to disk.
	edited := Default()
	edited.Symbols = []string{"ZZZZ"}
	if err := Save(edited); err != nil {
		t.Fatalf("Save (external edit): %v", err)
	}

	select {
	case cfg := <-notified:
		if len(cfg.Symbols) != 1 || cfg.Symbols[0] != "ZZZZ" {
			t.Fatalf("subscriber got %v, want [ZZZZ]", cfg.Symbols)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reload notification")
	}

	if got := store.Get().Symbols; len(got) != 1 || got[0] != "ZZZZ" {
		t.Fatalf("Get() = %v, want [ZZZZ]", got)
	}
}

func TestStoreUpdatePersistsAndNotifies(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	notified := make(chan Config, 4)
	store.Subscribe(func(cfg Config) { notified <- cfg })

	if err := store.Update(func(conf *Config) { conf.Symbols = append(conf.Symbols, "TSLA") }); err != nil {
		t.Fatalf("Update: %v", err)
	}

	select {
	case <-notified:
	case <-time.After(time.Second):
		t.Fatal("Update did not notify subscribers")
	}

	// The change must be on disk.
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	found := false
	for _, symbol := range loaded.Symbols {
		if symbol == "TSLA" {
			found = true
		}
	}
	if !found {
		t.Fatalf("persisted config %v missing TSLA", loaded.Symbols)
	}
}

func TestStoreUpdateInPlaceReorderPersistsAndNotifies(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	if err := store.Update(func(conf *Config) { conf.Symbols = []string{"A", "B", "C"} }); err != nil {
		t.Fatalf("seed Update: %v", err)
	}

	notified := make(chan Config, 4)
	store.Subscribe(func(cfg Config) { notified <- cfg })

	// Swap the first two entries in place — this must be detected as a change
	// (regression: a shallow copy shared the backing array and defeated it).
	if err := store.Update(func(conf *Config) {
		conf.Symbols[0], conf.Symbols[1] = conf.Symbols[1], conf.Symbols[0]
	}); err != nil {
		t.Fatalf("reorder Update: %v", err)
	}

	select {
	case cfg := <-notified:
		if len(cfg.Symbols) != 3 || cfg.Symbols[0] != "B" || cfg.Symbols[1] != "A" || cfg.Symbols[2] != "C" {
			t.Fatalf("notified order = %v, want [B A C]", cfg.Symbols)
		}
	case <-time.After(time.Second):
		t.Fatal("in-place reorder did not notify subscribers")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Symbols) != 3 || loaded.Symbols[0] != "B" || loaded.Symbols[1] != "A" {
		t.Fatalf("persisted order = %v, want [B A C]", loaded.Symbols)
	}
}

func TestStoreNoNotifyWhenUnchanged(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	var count int
	store.Subscribe(func(Config) { count++ })

	// Updating to an identical config must not persist or notify.
	if err := store.Update(func(*Config) {}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if waitFor(t, 300*time.Millisecond, func() bool { return count > 0 }) {
		t.Fatalf("no-op Update notified subscribers %d time(s)", count)
	}
}
