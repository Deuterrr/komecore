package contract_test

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type postmanCollection struct {
	Item []postmanItem `json:"item"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Item    []postmanItem   `json:"item,omitempty"`
	Request json.RawMessage `json:"request,omitempty"`
}

func countPostmanRequests(items []postmanItem) int {
	total := 0
	for _, item := range items {
		if len(item.Item) > 0 {
			total += countPostmanRequests(item.Item)
		} else if len(item.Request) > 0 {
			total += 1
		}
	}
	return total
}

func TestCollectionParity_CoverageAudit(t *testing.T) {
	// 1. Audit OpenAPI 3.0 Operations Count
	specPath := filepath.Join("..", "..", "api", "openapi.yaml")
	data, err := os.ReadFile(specPath)
	require.NoError(t, err, "failed to read api/openapi.yaml")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromData(data)
	require.NoError(t, err, "failed to parse OpenAPI spec")

	openAPIOperationsCount := 0
	for _, pathItem := range doc.Paths.Map() {
		openAPIOperationsCount += len(pathItem.Operations())
	}

	// 2. Audit Postman Collection Requests Count
	postmanPath := filepath.Join("..", "..", "api", "postman_collection.json")
	postmanData, err := os.ReadFile(postmanPath)
	require.NoError(t, err, "failed to read api/postman_collection.json")

	var postmanCol postmanCollection
	err = json.Unmarshal(postmanData, &postmanCol)
	require.NoError(t, err, "failed to unmarshal postman collection")
	postmanRequestsCount := countPostmanRequests(postmanCol.Item)

	// 3. Audit Bruno Collection Request Files Count
	brunoPath := filepath.Join("..", "..", "api", "bruno")
	brunoFilesCount := 0
	err = filepath.WalkDir(brunoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".bru") && d.Name() != "collection.bru" {
			brunoFilesCount++
		}
		return nil
	})
	require.NoError(t, err, "failed to walk bruno collection directory")

	// 4. Audit VS Code REST Client (.http) Requests Count
	httpFilePath := filepath.Join("..", "..", "api", "komecore.http")
	httpFile, err := os.Open(httpFilePath)
	require.NoError(t, err, "failed to open api/komecore.http")
	defer httpFile.Close()

	httpRequestsCount := 0
	scanner := bufio.NewScanner(httpFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "GET ") ||
			strings.HasPrefix(line, "POST ") ||
			strings.HasPrefix(line, "PUT ") ||
			strings.HasPrefix(line, "DELETE ") ||
			strings.HasPrefix(line, "PATCH ") {
			httpRequestsCount++
		}
	}
	require.NoError(t, scanner.Err(), "failed to scan api/komecore.http")

	// 5. Assert 100% Exact Parity Across All 4 Formats (95 operations)
	const expectedBaseline = 95

	t.Logf("OpenAPI Operations:    %d", openAPIOperationsCount)
	t.Logf("Postman Requests:      %d", postmanRequestsCount)
	t.Logf("Bruno Files:           %d", brunoFilesCount)
	t.Logf("komecore.http Entries: %d", httpRequestsCount)

	assert.Equal(t, expectedBaseline, openAPIOperationsCount, "OpenAPI operation count mismatch")
	assert.Equal(t, expectedBaseline, postmanRequestsCount, "Postman request count mismatch")
	assert.Equal(t, expectedBaseline, brunoFilesCount, "Bruno .bru request count mismatch")
	assert.Equal(t, expectedBaseline, httpRequestsCount, "komecore.http request count mismatch")

	assert.Equal(t, openAPIOperationsCount, postmanRequestsCount, "OpenAPI and Postman must have identical request counts")
	assert.Equal(t, openAPIOperationsCount, brunoFilesCount, "OpenAPI and Bruno must have identical request counts")
	assert.Equal(t, openAPIOperationsCount, httpRequestsCount, "OpenAPI and .http must have identical request counts")
}
