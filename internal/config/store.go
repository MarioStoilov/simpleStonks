package config

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

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

	mutex sync.RWMutex
	cfg   Config
	subs  []func(Config)

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
	if err := os.MkdirAll(dir, constants.DirPerm); err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	// Watch the directory, not the file: atomic saves (write-tmp + rename)
	// replace the file, which a file-level watch would stop following.
	if err := watcher.Add(dir); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	store := &Store{path: path, watcher: watcher, cfg: cfg, done: make(chan struct{})}
	go store.watch()
	return store, nil
}

// Get returns a snapshot of the current configuration.
func (store *Store) Get() Config {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	return store.cfg
}

// Subscribe registers callback to be called whenever the configuration changes,
// whether from Update or an external file edit.
func (store *Store) Subscribe(callback func(Config)) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.subs = append(store.subs, callback)
}

// Update applies mutate to the current config, persists it, and notifies
// subscribers. It is the entry point for UI-initiated changes.
func (store *Store) Update(mutate func(*Config)) error {
	store.mutex.Lock()
	next := store.cfg.clone()
	mutate(&next)
	if reflect.DeepEqual(next, store.cfg) {
		store.mutex.Unlock()
		return nil
	}
	if err := Save(next); err != nil {
		store.mutex.Unlock()
		return err
	}
	store.cfg = next
	subs := append([]func(Config){}, store.subs...)
	store.mutex.Unlock()

	for _, callback := range subs {
		callback(next)
	}
	return nil
}

// Close stops watching and releases resources.
func (store *Store) Close() error {
	close(store.done)
	return store.watcher.Close()
}

// watch coalesces filesystem events for the config file and reloads on change.
func (store *Store) watch() {
	var timer *time.Timer
	var timerC <-chan time.Time
	for {
		select {
		case <-store.done:
			if timer != nil {
				timer.Stop()
			}
			return
		case event, ok := <-store.watcher.Events:
			if !ok {
				return
			}
			if filepath.Clean(event.Name) != store.path {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(constants.ConfigReloadDebounce)
			} else {
				timer.Reset(constants.ConfigReloadDebounce)
			}
			timerC = timer.C
		case err, ok := <-store.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config: watch error: %v", err)
		case <-timerC:
			store.reload()
		}
	}
}

// reload reads the file and, if the config changed, updates it and notifies
// subscribers. A malformed file is logged and ignored, keeping the last-good
// config in memory.
func (store *Store) reload() {
	next, err := Load()
	if err != nil {
		log.Printf("config: reload skipped: %v", err)
		return
	}
	store.mutex.Lock()
	if reflect.DeepEqual(next, store.cfg) {
		store.mutex.Unlock()
		return
	}
	store.cfg = next
	subs := append([]func(Config){}, store.subs...)
	store.mutex.Unlock()

	for _, callback := range subs {
		callback(next)
	}
}
