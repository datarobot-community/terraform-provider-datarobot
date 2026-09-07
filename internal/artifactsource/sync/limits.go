package sync

// CLI source: cli/internal/workload/sync/limits.go
//
// Provider differences from CLI:
//   - Upload tunables (UploadConcurrency, StageVsZip* thresholds) already
//     live in internal/artifactsource/limits.go, which the upload half
//     reuses as its backend; they are not duplicated here.
//   - RollbackMaxFiles is defined next to the rollback tree it bounds
//     (rollback.go).
//   - The CLI's disk-space preflight (DiskSpaceMarginMB, diskspace*.go)
//     is not ported: it needs per-OS statfs syscalls, and terraform apply
//     has no "abort and free some space, then rerun" interaction.
const (
	// DownloadConcurrency bounds parallel remote-to-local downloads.
	DownloadConcurrency = 6
)
