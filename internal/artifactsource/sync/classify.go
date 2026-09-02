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

// The four download actions -- ActDownloadModify, ActDownloadAdd,
// ActDownloadDelete and ActDownloadOverDel -- are ported so this stays the
// CLI's whole table, but nothing in datarobot_artifact reaches them as the
// resource is wired today, and the Engine PR cannot simply hand them real I/O.
//
// What stands in the way is source.dir_hash. It is a Computed attribute, so
// Terraform holds the provider to whatever the plan committed: if the plan
// carries a known dir_hash, the state saved after apply has to be that same
// value, or the apply ends in "Provider produced inconsistent result after
// apply". Leaving the attribute unknown during plan is what buys the freedom
// to return something else; committing a value spends it.
//
// dir_hash is a fingerprint of the source tree, and the provider takes that
// fingerprint twice. ModifyPlan walks the directory during plan and commits
// the digest it finds there (plan.Source.DirHash = dirHash in
// pkg/provider/artifact_resource.go). Create and Update walk the same
// directory again at the end of apply and save that digest instead
// (refreshArtifactSourceDirHash). The two agree only for as long as apply
// leaves the tree alone. Uploads do -- they read local files and push bytes at
// the API. Downloads do not: add and modify create or overwrite a file under
// source.dir, delete removes one. So the second fingerprint stops matching the
// first, and Terraform fails the apply on the attribute rather than on the
// download.
//
// The provider does already write one file during apply, and the reason that
// one is legal is the reason downloads are not. computeArtifactSourceDirHash
// folds a synthetic entry for the .drignore it is about to write -- the
// embedded template's own hash and size -- into the plan-time digest, so plan
// fingerprints the tree as it will look after apply rather than as it looks
// now, and the two digests match. That works because the file is entirely
// knowable during plan: fixed name, fixed bytes, no network. A download set is
// not. Predicting it means fetching the remote manifest and diffing it against
// base and local, and the remote can move between plan and apply even then.
//
// So before any of these four does real I/O, the Engine either stays push-only
// -- classify against the remote, then execute only the skip and upload cells,
// which leave the local tree untouched -- or gives up committing a known
// dir_hash at plan time whenever a download could run, and accepts that the
// attribute then plans as "known after apply".
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
//
// The download actions it returns are unreachable in datarobot_artifact today;
// the Action constants record what has to change before they are wired.
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
