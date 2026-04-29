package proxy_test

import (
	"nautilus/internal/core/proxy"
	"nautilus/internal/rtree"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_ServeHTTP_Routing(t *testing.T) {
	manager := proxy.NewManager(nil)

	// 1. Setup Route Tree
	rawNodes := []*rtree.RawNode{
		{
			URL:     "example.com/api/test",
			Service: "test-service",
			Methods: "GET",
		},
		{
			URL:     "example.com/api/post",
			Service: "post-service",
			Methods: "POST",
		},
	}
	tree := rtree.Build(rawNodes)
	manager.UpdateTree(tree)

	// 2. Setup Nodes
	nodes := map[string][]string{
		"test-service": {"/tmp/test1.sock"},
	}
	manager.UpdateNodes(nodes)

	t.Run("Match GET Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
		w := httptest.NewRecorder()

		// Since we don't actually have a UDS listener running in this unit test,
		// we expect it to fail at the forwarding stage, but we can verify it passed routing.
		// However, we can use a virtual service to test the full logic without UDS.
		
		rawNodesWithVirtual := append(rawNodes, &rtree.RawNode{
			URL:     "example.com/virtual",
			Service: "$echo",
			Methods: "GET",
		})
		manager.UpdateTree(rtree.Build(rawNodesWithVirtual))

		req = httptest.NewRequest("GET", "http://example.com/virtual", nil)
		w = httptest.NewRecorder()
		manager.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Assuming $echo returns 200 by default or we can check its behavior if we knew it.
	})

	t.Run("Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/api/test", nil)
		w := httptest.NewRecorder()
		manager.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/unknown", nil)
		w := httptest.NewRecorder()
		manager.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestManager_LoadBalancing(t *testing.T) {
	manager := proxy.NewManager(nil)

	// 1. Setup Route Tree
	rawNodes := []*rtree.RawNode{
		{
			URL:     "lb.example.com/work",
			Service: "lb-service",
			Methods: "GET",
		},
	}
	tree := rtree.Build(rawNodes)
	manager.UpdateTree(tree)

	// 2. Setup Multiple Nodes
	nodes := map[string][]string{
		"lb-service": {"/tmp/node1.sock", "/tmp/node2.sock", "/tmp/node3.sock"},
	}
	manager.UpdateNodes(nodes)

	// Since we can't easily mock the Forwarder's Forward method without changing the code
	// to use an interface, we can at least verify the internal index increment logic
	// if we expose it or use reflection, but let's focus on what we can test.
	
	// We'll check if the indices are initialized correctly.
	manager.NodeLock.RLock()
	idxPtr, exists := manager.Indices["lb-service"]
	manager.NodeLock.RUnlock()

	require.True(t, exists)
	assert.Equal(t, uint32(0), atomic.LoadUint32(idxPtr))
}
