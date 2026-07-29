// CLI source: cli/internal/workload/fileops/paths_test.go (partial — SafeRelPath/NormalizePath only)
//
// Provider differences from CLI:
// - Tests paths.go functions in package filesapi; CLI tests fileops.SafeRelPath/NormalizePath.
// - DetectCaseCollisions tests remain CLI-only (not ported to provider).
package filesapi_test

import (
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeRelPath(t *testing.T) {
	require.NoError(t, filesapi.SafeRelPath("src/main.py"))

	err := filesapi.SafeRelPath("../escape")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes project root")

	err = filesapi.SafeRelPath(`dir\file.py`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backslash")
}

func TestNormalizePath(t *testing.T) {
	assert.Equal(t, "a/b.py", filesapi.NormalizePath("./a/b.py"))
	assert.Equal(t, "café.py", filesapi.NormalizePath("café.py"))
}
