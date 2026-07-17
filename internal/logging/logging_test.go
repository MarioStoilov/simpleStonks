package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MarioStoilov/simplestonks/internal/config"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestLevelFiltering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	lg, err := New(config.Logging{Level: config.LogInfo, File: path, MaxSizeMB: 10, MaxArchives: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer lg.Close()

	lg.Slog().Debug("debug-should-be-filtered")
	lg.Slog().Info("info-should-appear")
	lg.Slog().Error("error-should-appear")

	out := readFile(t, path)
	if strings.Contains(out, "debug-should-be-filtered") {
		t.Errorf("debug record leaked at info level:\n%s", out)
	}
	if !strings.Contains(out, "info-should-appear") || !strings.Contains(out, "error-should-appear") {
		t.Errorf("expected info and error records, got:\n%s", out)
	}
}

func TestSilentWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	lg, err := New(config.Logging{Level: config.LogSilent, File: path})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer lg.Close()

	lg.Slog().Error("must-not-be-written")

	if out := readFile(t, path); out != "" {
		t.Errorf("silent logger wrote output:\n%s", out)
	}
}

func TestReconfigureSwitchesDestination(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.log")
	pathB := filepath.Join(dir, "b.log")

	lg, err := New(config.Logging{Level: config.LogInfo, File: pathA, MaxSizeMB: 10, MaxArchives: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer lg.Close()

	lg.Slog().Info("first-into-a")

	if err := lg.Reconfigure(config.Logging{Level: config.LogInfo, File: pathB, MaxSizeMB: 10, MaxArchives: 1}); err != nil {
		t.Fatalf("Reconfigure: %v", err)
	}
	lg.Slog().Info("second-into-b")

	a, b := readFile(t, pathA), readFile(t, pathB)
	if !strings.Contains(a, "first-into-a") || strings.Contains(a, "second-into-b") {
		t.Errorf("file A has wrong content:\n%s", a)
	}
	if !strings.Contains(b, "second-into-b") || strings.Contains(b, "first-into-a") {
		t.Errorf("file B has wrong content:\n%s", b)
	}
}
