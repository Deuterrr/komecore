package contract_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateSnapshots = flag.Bool("update", false, "update golden snapshot files")

var requestIDRegex = regexp.MustCompile(`"request_id":\s*"[^"]+"`)

func normalizeSnapshotBody(data []byte) []byte {
	// Format to indented JSON for consistent diffs
	var prettyBuf bytes.Buffer
	if err := json.Indent(&prettyBuf, data, "", "  "); err == nil {
		data = prettyBuf.Bytes()
	}

	// Normalize dynamic request IDs
	data = requestIDRegex.ReplaceAll(data, []byte(`"request_id": "00000000-0000-0000-0000-000000000000"`))

	return data
}

func TestGoldenSnapshots(t *testing.T) {
	server := newTestContractServer(t)
	defer server.Close()

	client := server.Client()
	snapshotsDir := filepath.Join("testdata", "snapshots")
	err := os.MkdirAll(snapshotsDir, 0o755)
	require.NoError(t, err)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
		goldenFile     string
	}{
		{
			name:           "Product List Snapshot",
			method:         http.MethodGet,
			path:           "/api/v1/products?page=1&limit=10",
			expectedStatus: http.StatusOK,
			goldenFile:     "product_list.golden.json",
		},
		{
			name:           "Cart Calculate Snapshot",
			method:         http.MethodPost,
			path:           "/api/v1/carts/checkout/calculate",
			body:           `{"address_id":"e0000000-0000-0000-0000-000000000001","payment_method_id":"d0000000-0000-0000-0000-000000000001","shops":[{"shop_id":"c0000000-0000-0000-0000-000000000001","courier":{"code":"jne","service":"REG"},"items":[{"product_id":"b0000000-0000-0000-0000-000000000001","quantity":2}]}]}`,
			expectedStatus: http.StatusOK,
			goldenFile:     "cart_calculate.golden.json",
		},
		{
			name:           "Error Envelope Snapshot",
			method:         http.MethodPost,
			path:           "/api/v1/coupons/validate",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
			goldenFile:     "error_envelope.golden.json",
		},
		{
			name:           "Analytics Revenue Snapshot",
			method:         http.MethodGet,
			path:           "/api/v1/staff/analytics/revenue?range=30d&interval=day",
			expectedStatus: http.StatusOK,
			goldenFile:     "analytics_revenue.golden.json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = bytes.NewBufferString(tc.body)
			}

			reqURL := server.URL + tc.path
			req, err := http.NewRequest(tc.method, reqURL, bodyReader)
			require.NoError(t, err)

			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.expectedStatus, resp.StatusCode, "unexpected status for %s %s: %s", tc.method, tc.path, string(bodyBytes))

			normalized := normalizeSnapshotBody(bodyBytes)
			snapshotPath := filepath.Join(snapshotsDir, tc.goldenFile)

			if *updateSnapshots {
				err = os.WriteFile(snapshotPath, append(normalized, '\n'), 0o644)
				require.NoError(t, err, "failed to update snapshot: %s", snapshotPath)
				return
			}

			expectedData, err := os.ReadFile(snapshotPath)
			require.NoError(t, err, "golden snapshot missing, run with -update flag: %s", snapshotPath)

			expectedTrimmed := string(bytes.TrimSpace(expectedData))
			actualTrimmed := string(bytes.TrimSpace(normalized))

			assert.Equal(t, expectedTrimmed, actualTrimmed, "golden snapshot mismatch for %s", tc.goldenFile)
		})
	}
}
