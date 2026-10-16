package version_test

import (
	"testing"

	"komecore/pkg/version"

	"github.com/stretchr/testify/assert"
)

func TestVersion_DefaultValues(t *testing.T) {
	info := version.Get()
	assert.Equal(t, "v1.0.0", info.Version)
	assert.NotEmpty(t, version.String())
}
