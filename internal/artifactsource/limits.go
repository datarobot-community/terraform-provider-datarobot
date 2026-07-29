package artifactsource

// Upload orchestration tunables ported from cli/internal/workload/sync/limits.go.
const (
	UploadConcurrency = 4

	// Stage path is used when file count and total bytes are both at or below these thresholds.
	StageVsZipFileThreshold  = 20
	StageVsZipBytesThreshold = 50 * 1024 * 1024
)
