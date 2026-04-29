package watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nautilus/internal/core/registry"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	registry *registry.Registry

	mu          sync.Mutex
	activeScan  bool
	lastChange  time.Time
	eventSignal chan struct{}
	cancel      context.CancelFunc
	fsWatcher   *fsnotify.Watcher
}

func NewWatcher(r *registry.Registry) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		registry:    r,
		eventSignal: make(chan struct{}, 1),
		fsWatcher:   fw,
	}

	return watcher, nil
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			log.Printf("Watching directory: %s", path)
			return w.fsWatcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) Start() error {
	// Initial Scan
	if _, err := w.registry.Scan(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	// Start Event Listener
	go w.listenEvents(ctx)

	// Start Hybrid Controller
	go w.runHybridLoop(ctx)

	// Add base directory to fsnotify
	root := w.registry.BaseDir()
	if err := w.addRecursive(root); err != nil {
		return err
	}

	if _, err := w.registry.Scan(); err != nil {
		log.Printf("Initial scan error: %v", err)
	}

	log.Println("Hybrid Watcher started (Events + Dynamic Ticker)")
	return nil
}

func (w *Watcher) listenEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			log.Println("fsnotify event:", event)

			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					w.addRecursive(event.Name)
				}
			}

			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) ||
				event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				select {
				case w.eventSignal <- struct{}{}:
				default:
				}
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Println("fsnotify error:", err)
		}
	}
}

func (w *Watcher) runHybridLoop(ctx context.Context) {
	var ticker *time.Ticker
	var tickerChan <-chan time.Time

	idleTimeout := 30 * time.Second
	scanInterval := 1 * time.Second

	for {
		select {
		case <-ctx.Done():
			if ticker != nil {
				ticker.Stop()
			}
			return

		case <-w.eventSignal:
			w.mu.Lock()
			w.lastChange = time.Now()
			if !w.activeScan {
				log.Println("Activity detected, entering high-frequency scan mode...")
				w.activeScan = true
				ticker = time.NewTicker(scanInterval)
				tickerChan = ticker.C
			}
			w.mu.Unlock()

		case <-tickerChan:
			changed, err := w.registry.Scan()
			if err != nil {
				log.Println("scan error:", err)
			}

			w.mu.Lock()
			if changed {
				w.lastChange = time.Now()
			}

			// Check for idleness
			if time.Since(w.lastChange) > idleTimeout {
				log.Println("No changes for 30s, entering idle mode (Events only)...")
				w.activeScan = false
				ticker.Stop()
				ticker = nil
				tickerChan = nil
			}
			w.mu.Unlock()
		}
	}
}

func (w *Watcher) Close() error {
	if w.cancel != nil {
		w.cancel()
	}
	if w.fsWatcher != nil {
		w.fsWatcher.Close()
	}
	return nil
}
