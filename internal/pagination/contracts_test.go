package pagination_test

import (
	"encoding/json"
	"testing"

	"komecore/internal/pagination"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateTotalPages(t *testing.T) {
	tests := []struct {
		name       string
		totalItems int64
		limit      int
		expected   int
	}{
		{"zero total items", 0, 10, 0},
		{"negative total items", -5, 10, 0},
		{"zero limit", 100, 0, 0},
		{"negative limit", 100, -10, 0},
		{"exact multiple", 20, 10, 2},
		{"exact single page", 10, 10, 1},
		{"less than one page", 7, 10, 1},
		{"one more than page ceiling", 21, 10, 3},
		{"large dataset ceiling", 142, 20, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pagination.CalculateTotalPages(tt.totalItems, tt.limit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewPaginationMeta(t *testing.T) {
	meta := pagination.NewPaginationMeta(1, 20, 142)
	assert.Equal(t, 1, meta.Page)
	assert.Equal(t, 20, meta.Limit)
	assert.Equal(t, int64(142), meta.TotalItems)
	assert.Equal(t, 8, meta.TotalPages)
	assert.Nil(t, meta.NextCursor)

	// JSON serialization test: next_cursor MUST be serialized as null (not omitted)
	bytes, err := json.Marshal(meta)
	require.NoError(t, err)

	expectedJSON := `{"page":1,"limit":20,"total_items":142,"total_pages":8,"next_cursor":null}`
	assert.JSONEq(t, expectedJSON, string(bytes))
}
