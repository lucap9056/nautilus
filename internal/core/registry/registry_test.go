package registry

import (
	"os"
	"path/filepath"
	"testing"
)

type mockStore struct {
	services     map[string]bool
	nodes        map[string][]string
	upsertCalled int
}

func newMockStore() *mockStore {
	return &mockStore{
		services: make(map[string]bool),
		nodes:    make(map[string][]string),
	}
}

func (m *mockStore) UpsertService(serviceID string) error {
	m.services[serviceID] = true
	m.upsertCalled++
	return nil
}

func (m *mockStore) RegisterNode(nodeID, serviceID string) error {
	m.nodes[serviceID] = append(m.nodes[serviceID], nodeID)
	return nil
}

func (m *mockStore) UnregisterNode(nodeID string) error {
	return nil
}

func (m *mockStore) UnregisterAllNodes(serviceID string) error {
	delete(m.nodes, serviceID)
	return nil
}

func (m *mockStore) ReplaceServiceNodes(serviceID string, nodeIDs []string) error {
	m.nodes[serviceID] = nodeIDs
	return nil
}

func (m *mockStore) GetNodesByService(serviceID string) ([]string, error) {
	return m.nodes[serviceID], nil
}

func TestRegistry_Scan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nibble-registry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	reg := NewRegistry(tmpDir)

	// Test case 1: Empty directory
	changed, err := reg.Scan()
	if err != nil {
		t.Errorf("Scan failed: %v", err)
	}
	if changed {
		t.Errorf("expected no changes in empty directory")
	}

	// Test case 2: Add services and nodes
	// Structure:
	// tmpDir/
	//   api/
	//     v1.sock
	//     v2.sock
	//   web/
	//     app.sock

	apiDir := filepath.Join(tmpDir, "api")
	webDir := filepath.Join(tmpDir, "web")
	os.MkdirAll(apiDir, 0755)
	os.MkdirAll(webDir, 0755)

	v1Path := filepath.Join(apiDir, "v1.sock")
	v2Path := filepath.Join(apiDir, "v2.sock")
	appPath := filepath.Join(webDir, "app.sock")

	os.WriteFile(v1Path, []byte(""), 0644)
	os.WriteFile(v2Path, []byte(""), 0644)
	os.WriteFile(appPath, []byte(""), 0644)

	changed, err = reg.Scan()
	if err != nil {
		t.Errorf("Scan failed: %v", err)
	}
	if !changed {
		t.Errorf("expected changes after adding sockets")
	}

	// Test case 4: Remove a node
	os.Remove(v1Path)
	changed, err = reg.Scan()
	if err != nil {
		t.Errorf("Scan failed: %v", err)
	}
	if !changed {
		t.Errorf("expected changes after removing a socket")
	}

}
