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

	mutex sync.Mutex
	file  *os.File
	size  int64
}

// newRotatingWriter opens (creating as needed) the log file and its directory.
func newRotatingWriter(path string, maxSizeBytes int64, maxArchives int) (*rotatingWriter, error) {
	writer := &rotatingWriter{path: path, maxSize: maxSizeBytes, maxArchives: maxArchives}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := writer.open(); err != nil {
		return nil, err
	}
	return writer, nil
}

// open opens the log file for appending and records its current size.
func (writer *rotatingWriter) open() error {
	file, err := os.OpenFile(writer.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	writer.file = file
	writer.size = info.Size()
	return nil
}

// Write appends data, rotating first if it would push the file past the
// threshold.
func (writer *rotatingWriter) Write(data []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	if writer.maxSize > 0 && writer.size > 0 && writer.size+int64(len(data)) > writer.maxSize {
		if err := writer.rotate(); err != nil {
			return 0, err
		}
	}
	written, err := writer.file.Write(data)
	writer.size += int64(written)
	return written, err
}

// rotate closes the current file, shifts the archives, and reopens a fresh log.
// Caller must hold writer.mutex.
func (writer *rotatingWriter) rotate() error {
	if err := writer.file.Close(); err != nil {
		return err
	}
	if writer.maxArchives <= 0 {
		// Retain no archives: discard the current file and start fresh.
		if err := os.Remove(writer.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return writer.open()
	}
	// Drop the archive that would fall off the end of the retention window.
	if err := os.Remove(writer.archiveName(writer.maxArchives)); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Shift existing archives up: .(n-1) -> .n, ..., .1 -> .2
	for idx := writer.maxArchives - 1; idx >= 1; idx-- {
		src := writer.archiveName(idx)
		if _, err := os.Stat(src); err != nil {
			continue // gap in the sequence; nothing to move
		}
		if err := os.Rename(src, writer.archiveName(idx+1)); err != nil {
			return err
		}
	}
	// Current log becomes the newest archive.
	if err := os.Rename(writer.path, writer.archiveName(1)); err != nil {
		return err
	}
	return writer.open()
}

// archiveName returns the path of the idx-th archive (1 = newest).
func (writer *rotatingWriter) archiveName(idx int) string {
	return fmt.Sprintf("%s.%d", writer.path, idx)
}

// Close closes the underlying file.
func (writer *rotatingWriter) Close() error {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	if writer.file == nil {
		return nil
	}
	err := writer.file.Close()
	writer.file = nil
	return err
}
