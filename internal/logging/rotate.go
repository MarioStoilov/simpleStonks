package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// rotatingWriter is an io.WriteCloser that appends to a log file and rotates it
// once it exceeds a size threshold, retaining a bounded number of numbered
// archives (path.1, path.2, ...). It is safe for concurrent use.
type rotatingWriter struct {
	path        string
	maxSize     int64 // bytes; <= 0 disables rotation
	maxArchives int   // number of archive files to retain

	mu   sync.Mutex
	file *os.File
	size int64
}

// newRotatingWriter opens (creating as needed) the log file and its directory.
func newRotatingWriter(path string, maxSizeBytes int64, maxArchives int) (*rotatingWriter, error) {
	w := &rotatingWriter{path: path, maxSize: maxSizeBytes, maxArchives: maxArchives}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

// open opens the log file for appending and records its current size.
func (w *rotatingWriter) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	w.file = f
	w.size = info.Size()
	return nil
}

// Write appends p, rotating first if it would push the file past the threshold.
func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.maxSize > 0 && w.size > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// rotate closes the current file, shifts the archives, and reopens a fresh log.
// Caller must hold w.mu.
func (w *rotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	if w.maxArchives <= 0 {
		// Retain no archives: discard the current file and start fresh.
		if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return w.open()
	}
	// Drop the archive that would fall off the end of the retention window.
	if err := os.Remove(w.archiveName(w.maxArchives)); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Shift existing archives up: .(n-1) -> .n, ..., .1 -> .2
	for i := w.maxArchives - 1; i >= 1; i-- {
		src := w.archiveName(i)
		if _, err := os.Stat(src); err != nil {
			continue // gap in the sequence; nothing to move
		}
		if err := os.Rename(src, w.archiveName(i+1)); err != nil {
			return err
		}
	}
	// Current log becomes the newest archive.
	if err := os.Rename(w.path, w.archiveName(1)); err != nil {
		return err
	}
	return w.open()
}

// archiveName returns the path of the i-th archive (1 = newest).
func (w *rotatingWriter) archiveName(i int) string {
	return fmt.Sprintf("%s.%d", w.path, i)
}

// Close closes the underlying file.
func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
