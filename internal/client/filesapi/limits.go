// CLI source:
//   - cli/internal/drapi/filesapi/limits.go (HTTP timeouts)
//   - cli/internal/workload/sync/limits.go (ZipPollInterval, ZipPollTimeout)
//
// Provider differences from CLI:
//   - Timeout constants are exported (UploadHTTPTimeout, etc.); CLI uses unexported names
//     (uploadHTTPTimeout, downloadHTTPTimeout, statusPollHTTPTimeout).
//   - ZipPollInterval/ZipPollTimeout are co-located here for upcoming sync code; CLI keeps
//     them in internal/workload/sync/limits.go.
package filesapi

import "time"

// HTTP-level tunables for the upload and download paths.
const (
	UploadHTTPTimeout   = 600 * time.Second
	DownloadHTTPTimeout = 600 * time.Second

	// StatusPollHTTPTimeout caps a single async-status poll request.
	StatusPollHTTPTimeout = 30 * time.Second
)

// Async zip upload polling defaults.
const (
	ZipPollInterval = 500 * time.Millisecond
	ZipPollTimeout  = 600 * time.Second
)
