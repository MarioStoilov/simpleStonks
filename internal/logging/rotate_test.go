package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// countArchives returns how many numbered archives exist for base.
func countArchives(t *testing.T, base string) int {
	t.Helper()
	matches, err := filepath.Glob(base + ".*")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	return len(matches)
}

func TestRotatingWriterRotatesAtThreshold(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	// 100-byte threshold, keep at most 2 archives.
	w, err := newRotatingWriter(path, 100, 2)
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	defer w.Close()

	line := strings.Repeat("x", 60) + "\n" // 61 bytes; two lines exceed 100
	for i := 0; i < 10; i++ {
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	// Retention must be enforced: never more than maxArchives archives.
	if n := countArchives(t, path); n > 2 {
		t.Fatalf("kept %d archives, want <= 2", n)
	}
	// Rotation must have happened at least once.
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected archive %s.1 to exist: %v", path, err)
	}
	// The live log must never exceed the threshold plus one record.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat live log: %v", err)
	}
	if info.Size() > 100+int64(len(line)) {
		t.Fatalf("live log size %d exceeds threshold budget", info.Size())
	}
}

func TestRotatingWriterNoArchives(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	w, err := newRotatingWriter(path, 50, 0) // keep zero archives
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	defer w.Close()

	line := strings.Repeat("y", 40) + "\n"
	for i := 0; i < 6; i++ {
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	if n := countArchives(t, path); n != 0 {
		t.Fatalf("kept %d archives with maxArchives=0, want 0", n)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("live log should still exist: %v", err)
	}
}
