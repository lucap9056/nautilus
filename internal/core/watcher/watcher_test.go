package watcher

import (
	"os"
	"path/filepath"
	"testing"

	"nautilus/internal/core/registry"
)

func TestWatcher_Basic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nibble-watcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	reg := registry.NewRegistry(tmpDir)

	w, err := NewWatcher(reg)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Close()

	// Initial start should perform a scan
	err = w.Start()
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	// Add a socket and trigger manual scan via event signal or just wait for ticker
	// For simplicity in unit test, we verify that Start() didn't crash and initialized the registry.
	if reg.BaseDir() != tmpDir {
		t.Errorf("expected base dir %s, got %s", tmpDir, reg.BaseDir())
	}
}

func TestWatcher_ScanOnStart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nibble-watcher-start-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Pre-create some structure
	svcDir := filepath.Join(tmpDir, "test-svc")
	os.MkdirAll(svcDir, 0755)
	os.WriteFile(filepath.Join(svcDir, "1.sock"), []byte(""), 0644)

	reg := registry.NewRegistry(tmpDir)

	w, err := NewWatcher(reg)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

}
