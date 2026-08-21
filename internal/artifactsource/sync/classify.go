package sync

// CLI source: cli/internal/workload/sync/classify.go
//
// Classify / ActionFor are a pure port. Empty hash means the path is absent
// on that side.

// Classification names a cell of the three-way (BASE × LOCAL × REMOTE) truth
// table. Each Classification maps to exactly one Action.
type Classification int

const (
	ClsUnchanged Classification = iota
	ClsLocalModified
	ClsRemoteModified
	ClsConverged
	ClsConflict
	ClsLocalAdded
	ClsRemoteAdded
	ClsAddConflict
	ClsLocalDeleted
	ClsRemoteDeleted
	ClsBothDeleted
	ClsDelEditConflict // local edited, remote deleted
	ClsEditDelConflict // local deleted, remote edited
	ClsBothAddedSame
)

var classificationNames = map[Classification]string{
	ClsUnchanged:       "UNCHANGED",
	ClsLocalModified:   "LOCAL_MODIFIED",
	ClsRemoteModified:  "REMOTE_MODIFIED",
	ClsConverged:       "CONVERGED",
	ClsConflict:        "CONFLICT",
	ClsLocalAdded:      "LOCAL_ADDED",
	ClsRemoteAdded:     "REMOTE_ADDED",
	ClsAddConflict:     "ADD_CONFLICT",
	ClsLocalDeleted:    "LOCAL_DELETED",
	ClsRemoteDeleted:   "REMOTE_DELETED",
	ClsBothDeleted:     "BOTH_DELETED",
	ClsDelEditConflict: "DEL_EDIT_CONFLICT",
	ClsEditDelConflict: "EDIT_DEL_CONFLICT",
	ClsBothAddedSame:   "BOTH_ADDED_SAME",
}

func (c Classification) String() string {
	if name, ok := classificationNames[c]; ok {
		return name
	}

	return "UNKNOWN"
}

// IsConflict reports whether the classification represents a conflict.
func (c Classification) IsConflict() bool {
	switch c {
	case ClsConflict, ClsAddConflict, ClsDelEditConflict, ClsEditDelConflict:
		return true
	case ClsUnchanged, ClsLocalModified, ClsRemoteModified, ClsConverged,
		ClsLocalAdded, ClsRemoteAdded, ClsLocalDeleted, ClsRemoteDeleted,
		ClsBothDeleted, ClsBothAddedSame:
		return false
	}

	return false
}

// Action is the operation later PRs apply to a path during execute.
type Action int

const (
	ActSkip Action = iota
	ActUploadModify
	ActUploadAdd
	ActUploadDelete
	ActDownloadModify
	ActDownloadAdd
	ActDownloadDelete
	ActConflictCopy
	ActDownloadOverDel
)

// ActionFor returns the action for a Classification. EDIT_DEL_CONFLICT maps
// to ActDownloadOverDel rather than ActConflictCopy because the user already
// deleted that file, so no .LOCAL copy is kept.
func ActionFor(c Classification) Action {
	switch c {
	case ClsUnchanged, ClsConverged, ClsBothDeleted, ClsBothAddedSame:
		return ActSkip
	case ClsLocalModified:
		return ActUploadModify
	case ClsLocalAdded:
		return ActUploadAdd
	case ClsLocalDeleted:
		return ActUploadDelete
	case ClsRemoteModified:
		return ActDownloadModify
	case ClsRemoteAdded:
		return ActDownloadAdd
	case ClsRemoteDeleted:
		return ActDownloadDelete
	case ClsConflict, ClsAddConflict, ClsDelEditConflict:
		return ActConflictCopy
	case ClsEditDelConflict:
		return ActDownloadOverDel
	}

	return ActSkip
}

// Classify maps the (base, local, remote) triple to a Classification. An
// empty hash means absent on that side.
func Classify(baseHash, localHash, remoteHash string) Classification {
	bExists := baseHash != ""
	lExists := localHash != ""
	rExists := remoteHash != ""

	if !bExists {
		return classifyAbsentBase(localHash, remoteHash, lExists, rExists)
	}

	return classifyPresentBase(baseHash, localHash, remoteHash, lExists, rExists)
}

func classifyAbsentBase(localHash, remoteHash string, lExists, rExists bool) Classification {
	switch {
	case !lExists && !rExists:
		return ClsUnchanged
	case lExists && !rExists:
		return ClsLocalAdded
	case !lExists && rExists:
		return ClsRemoteAdded
	case localHash == remoteHash:
		return ClsBothAddedSame
	}

	return ClsAddConflict
}

func classifyPresentBase(base, local, remote string, lExists, rExists bool) Classification {
	if !lExists || !rExists {
		return classifyDeletionInvolvedWithBase(base, local, remote, lExists, rExists)
	}

	return classifyBothPresentWithBase(base, local, remote)
}

func classifyDeletionInvolvedWithBase(base, local, remote string, lExists, rExists bool) Classification {
	switch {
	case !lExists && !rExists:
		return ClsBothDeleted
	case !lExists && remote == base:
		return ClsLocalDeleted
	case !lExists:
		return ClsEditDelConflict
	case !rExists && local == base:
		return ClsRemoteDeleted
	}

	return ClsDelEditConflict
}

func classifyBothPresentWithBase(base, local, remote string) Classification {
	localChanged := local != base
	remoteChanged := remote != base

	switch {
	case !localChanged && !remoteChanged:
		return ClsUnchanged
	case localChanged && !remoteChanged:
		return ClsLocalModified
	case !localChanged && remoteChanged:
		return ClsRemoteModified
	case local == remote:
		return ClsConverged
	}

	return ClsConflict
}
