package config

import (
	"testing"
	"time"
)

// waitFor blocks until fn returns true or the timeout elapses.
func waitFor(t *testing.T, timeout time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fn()
}

func TestStoreLiveReloadOnExternalEdit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	notified := make(chan Config, 4)
	store.Subscribe(func(c Config) { notified <- c })

	// Simulate an external edit: write a new config directly to disk.
	edited := Default()
	edited.Symbols = []string{"ZZZZ"}
	if err := Save(edited); err != nil {
		t.Fatalf("Save (external edit): %v", err)
	}

	select {
	case c := <-notified:
		if len(c.Symbols) != 1 || c.Symbols[0] != "ZZZZ" {
			t.Fatalf("subscriber got %v, want [ZZZZ]", c.Symbols)
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
	store.Subscribe(func(c Config) { notified <- c })

	if err := store.Update(func(c *Config) { c.Symbols = append(c.Symbols, "TSLA") }); err != nil {
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
	for _, s := range loaded.Symbols {
		if s == "TSLA" {
			found = true
		}
	}
	if !found {
		t.Fatalf("persisted config %v missing TSLA", loaded.Symbols)
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
