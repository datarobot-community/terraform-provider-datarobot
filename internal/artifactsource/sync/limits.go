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
//   - RemoteDeleteBatchSize is provider-only: the CLI's applyDeletes sends
//     every path in one request.
const (
	// DownloadConcurrency bounds parallel remote-to-local downloads.
	DownloadConcurrency = 6

	// RemoteDeleteBatchSize bounds the paths sent in one DeleteFiles
	// request. The local half caps what it rewrites (RollbackMaxFiles);
	// the remote half had no bound at all, so a subtree removed locally
	// went out in a single request whose body grew with the deletion and
	// whose failure was all-or-nothing. Each batch is a catalog version of
	// its own, and a failure part-way leaves the next Plan a smaller set.
	RemoteDeleteBatchSize = 500
)
