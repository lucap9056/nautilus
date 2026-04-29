package compiler_test

import (
	"nautilus/internal/compiler"
	"nautilus/internal/rtree"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	script := `
# This is a comment
* example.com/api/v1 svc-1
    mw-auth
    mw-log

GET example.com/api/v2 svc-2
    # Nested comment
    mw-cache

# Case with omitted Method (defaults to *)
example.com/api/v3 svc-3
`

	tree, err := compiler.ParseString(script)
	require.NoError(t, err)
	require.NotNil(t, tree)

	t.Run("Middleware Inheritance", func(t *testing.T) {
		url := rtree.ReverseHost("example.com/api/v1")
		node, exists := tree.Search(url)
		require.True(t, exists)
		assert.Equal(t, "svc-1", tree.ServicePool[node.ServiceID])

		// Verify middlewares are correctly compiled
		var mws []string
		for _, id := range node.Middlewares {
			mws = append(mws, tree.MiddlewarePool[id])
		}
		assert.ElementsMatch(t, []string{"mw-auth", "mw-log"}, mws)
	})

	t.Run("Method Filtering", func(t *testing.T) {
		url := rtree.ReverseHost("example.com/api/v2")
		node, exists := tree.Search(url)
		require.True(t, exists)
		assert.Equal(t, rtree.MethodGet, node.Methods&rtree.MethodGet)
	})

	t.Run("Default Method Star", func(t *testing.T) {
		url := rtree.ReverseHost("example.com/api/v3")
		node, exists := tree.Search(url)
		require.True(t, exists)
		assert.Equal(t, rtree.MethodAny, node.Methods)
	})
}

func TestParse_WithExpansion(t *testing.T) {
	script := `
GET [a|b].io/api svc-expanded
`
	tree, err := compiler.ParseString(script)
	require.NoError(t, err)

	// Verify both expanded paths point to the same service
	urls := []string{"a.io/api", "b.io/api"}
	for _, u := range urls {
		node, exists := tree.Search(rtree.ReverseHost(u))
		assert.True(t, exists, "Path %s should exist", u)
		assert.Equal(t, "svc-expanded", tree.ServicePool[node.ServiceID])
	}
}

func TestParse_InvalidBuiltin(t *testing.T) {
	script := `
GET example.com/api svc
    $NonExistentFunc()
`
	_, err := compiler.ParseString(script)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown builtin middleware")
}
