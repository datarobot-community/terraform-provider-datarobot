package sync

import "sort"

// CLI source: cli/internal/workload/sync/plan.go
//
// SyncPlan is the blueprint later PRs execute. Skip actions are never stored.

// FileAction is one row of the SyncPlan.
type FileAction struct {
	Path           string
	Classification Classification
	Action         Action
	LocalSize      int64
	RemoteSize     int64
	LocalHash      string
	RemoteHash     string
}

// SyncPlan groups FileActions by what execute will do.
type SyncPlan struct {
	Uploads   []FileAction // LOCAL_MODIFIED + LOCAL_ADDED
	Downloads []FileAction // REMOTE_MODIFIED + REMOTE_ADDED + EDIT_DEL_CONFLICT
	Deletes   []FileAction // LOCAL_DELETED + REMOTE_DELETED
	Conflicts []FileAction // CONFLICT + ADD_CONFLICT + DEL_EDIT_CONFLICT

	// OldVersionShort is the 8-char prefix of the BASE catalog version;
	// empty before the first successful sync. Set by the Engine later.
	OldVersionShort string
}

// Append routes a FileAction into the right group based on its Action.
// Skip actions are dropped.
func (p *SyncPlan) Append(fa FileAction) {
	switch fa.Action {
	case ActSkip:
	case ActUploadModify, ActUploadAdd:
		p.Uploads = append(p.Uploads, fa)
	case ActDownloadModify, ActDownloadAdd, ActDownloadOverDel:
		p.Downloads = append(p.Downloads, fa)
	case ActUploadDelete, ActDownloadDelete:
		p.Deletes = append(p.Deletes, fa)
	case ActConflictCopy:
		// Conflicts also need the remote download (remote wins); the
		// executor issues that download alongside the conflict copy.
		p.Conflicts = append(p.Conflicts, fa)
	}
}

// Sort orders every group by path. Call once after Diff.
func (p *SyncPlan) Sort() {
	sort.Slice(p.Uploads, func(i, j int) bool { return p.Uploads[i].Path < p.Uploads[j].Path })
	sort.Slice(p.Downloads, func(i, j int) bool { return p.Downloads[i].Path < p.Downloads[j].Path })
	sort.Slice(p.Deletes, func(i, j int) bool { return p.Deletes[i].Path < p.Deletes[j].Path })
	sort.Slice(p.Conflicts, func(i, j int) bool { return p.Conflicts[i].Path < p.Conflicts[j].Path })
}

// HasConflicts reports whether any conflict-class rows exist.
func (p *SyncPlan) HasConflicts() bool {
	return len(p.Conflicts) > 0
}

// IsEmpty reports whether the plan has nothing to do.
func (p *SyncPlan) IsEmpty() bool {
	return len(p.Uploads) == 0 && len(p.Downloads) == 0 && len(p.Deletes) == 0 && len(p.Conflicts) == 0
}

// TotalUploadBytes sums the bytes the upload step will push.
func (p *SyncPlan) TotalUploadBytes() int64 {
	var n int64
	for _, fa := range p.Uploads {
		n += fa.LocalSize
	}

	return n
}

// TotalDownloadBytes sums download bytes plus remote bytes for conflict
// copies (those also pull the remote).
func (p *SyncPlan) TotalDownloadBytes() int64 {
	var n int64
	for _, fa := range p.Downloads {
		n += fa.RemoteSize
	}
	for _, fa := range p.Conflicts {
		n += fa.RemoteSize
	}

	return n
}

// ConflictPaths returns the conflict paths sorted alphabetically.
func (p *SyncPlan) ConflictPaths() []string {
	out := make([]string, 0, len(p.Conflicts))
	for _, fa := range p.Conflicts {
		out = append(out, fa.Path)
	}
	sort.Strings(out)

	return out
}
