// CLI source: cli/internal/drapi/filesapi/types_test.go
//
// Provider differences from CLI:
// - Tests live in external package filesapi_test (CLI tests are in package filesapi).
// - Logic and cases are identical to CLI.
package filesapi_test

import (
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
)

func TestIsTerminalStatus(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{filesapi.StatusInitialized, false},
		{filesapi.StatusRunningToWorkers, false},
		{filesapi.StatusStartedOnWorker, false},
		{filesapi.StatusCompleted, true},
		{filesapi.StatusError, true},
		{filesapi.StatusAborted, true},
		{filesapi.StatusExpired, true},
		{"", false},
		{"UNKNOWN", false},
	}

	for _, tc := range cases {
		t.Run(tc.s, func(t *testing.T) {
			assert.Equal(t, tc.want, filesapi.IsTerminalStatus(tc.s))
		})
	}
}

func TestIsErrorStatus(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{filesapi.StatusInitialized, false},
		{filesapi.StatusRunningToWorkers, false},
		{filesapi.StatusCompleted, false},
		{filesapi.StatusError, true},
		{filesapi.StatusAborted, true},
		{filesapi.StatusExpired, true},
	}

	for _, tc := range cases {
		t.Run(tc.s, func(t *testing.T) {
			assert.Equal(t, tc.want, filesapi.IsErrorStatus(tc.s))
		})
	}
}
