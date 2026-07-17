package config

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounce is how long the watcher coalesces filesystem events before reloading.
// Editors and atomic saves emit several events per write; this collapses them.
const debounce = 150 * time.Millisecond

// Store holds the live configuration and keeps it in sync with the config file
// in both directions: UI edits go through Update (which persists), and external
// edits to the file are detected and reloaded automatically.
//
// Store is safe for concurrent use. Subscriber callbacks are invoked on the
// goroutine that observed the change (the watcher goroutine for external edits,
// the caller's goroutine for Update), so UI subscribers must marshal their work
// onto the UI thread themselves.
type Store struct {
	path    string
	watcher *fsnotify.Watcher

	mu   sync.RWMutex
	cfg  Config
	subs []func(Config)

	done chan struct{}
}

// Open loads the config (falling back to defaults when the file is absent) and
// starts watching its directory for external changes.
func Open() (*Store, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	cfg, err := Load()
	if err != nil {
		// Corrupt file on startup: start from defaults rather than failing to
		// launch; a later valid edit will be picked up by the watcher.
		log.Printf("config: %v; starting from defaults", err)
		cfg = Default()
	}

	// The config directory may not exist yet on first run; create it so it can
	// be watched before the first save.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	// Watch the directory, not the file: atomic saves (write-tmp + rename)
	// replace the file, which a file-level watch would stop following.
	if err := w.Add(dir); err != nil {
		_ = w.Close()
		return nil, err
	}

	s := &Store{path: path, watcher: w, cfg: cfg, done: make(chan struct{})}
	go s.watch()
	return s, nil
}

// Get returns a snapshot of the current configuration.
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Subscribe registers fn to be called whenever the configuration changes,
// whether from Update or an external file edit.
func (s *Store) Subscribe(fn func(Config)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs = append(s.subs, fn)
}

// Update applies mutate to the current config, persists it, and notifies
// subscribers. It is the entry point for UI-initiated changes.
func (s *Store) Update(mutate func(*Config)) error {
	s.mu.Lock()
	next := s.cfg
	mutate(&next)
	if reflect.DeepEqual(next, s.cfg) {
		s.mu.Unlock()
		return nil
	}
	if err := Save(next); err != nil {
		s.mu.Unlock()
		return err
	}
	s.cfg = next
	subs := append([]func(Config){}, s.subs...)
	s.mu.Unlock()

	for _, fn := range subs {
		fn(next)
	}
	return nil
}

// Close stops watching and releases resources.
func (s *Store) Close() error {
	close(s.done)
	return s.watcher.Close()
}

// watch coalesces filesystem events for the config file and reloads on change.
func (s *Store) watch() {
	var timer *time.Timer
	var timerC <-chan time.Time
	for {
		select {
		case <-s.done:
			if timer != nil {
				timer.Stop()
			}
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if filepath.Clean(event.Name) != s.path {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(debounce)
			} else {
				timer.Reset(debounce)
			}
			timerC = timer.C
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config: watch error: %v", err)
		case <-timerC:
			s.reload()
		}
	}
}

// reload reads the file and, if the config changed, updates it and notifies
// subscribers. A malformed file is logged and ignored, keeping the last-good
// config in memory.
func (s *Store) reload() {
	next, err := Load()
	if err != nil {
		log.Printf("config: reload skipped: %v", err)
		return
	}
	s.mu.Lock()
	if reflect.DeepEqual(next, s.cfg) {
		s.mu.Unlock()
		return
	}
	s.cfg = next
	subs := append([]func(Config){}, s.subs...)
	s.mu.Unlock()

	for _, fn := range subs {
		fn(next)
	}
}
