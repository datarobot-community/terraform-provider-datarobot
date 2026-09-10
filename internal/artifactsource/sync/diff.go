package sync

// CLI source: cli/internal/workload/sync/diff.go
//
// Diff is a pure port. FromFilesAPI is deferred: this PR has no Files API.

// FileEntry is the per-path hash and size Diff needs.
type FileEntry struct {
	Hash string
	Size int64
}

type (
	LocalManifest  = map[string]FileEntry
	RemoteManifest = map[string]FileEntry
	BaseManifest   = map[string]FileEntry
)

// Diff produces a SyncPlan from the three input manifests. The result is
// unsorted; callers must call Sort before display or execution.
func Diff(base, local, remote BaseManifest) *SyncPlan {
	plan := &SyncPlan{}

	for path := range pathUnion(base, local, remote) {
		b := base[path]
		l := local[path]
		r := remote[path]

		cls := Classify(b.Hash, l.Hash, r.Hash)
		act := ActionFor(cls)

		if act == ActSkip {
			continue
		}

		plan.Append(FileAction{
			Path:           path,
			Classification: cls,
			Action:         act,
			LocalSize:      l.Size,
			RemoteSize:     r.Size,
			LocalHash:      l.Hash,
			RemoteHash:     r.Hash,
		})
	}

	return plan
}

func pathUnion(maps ...BaseManifest) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range maps {
		for k := range m {
			out[k] = struct{}{}
		}
	}

	return out
}
