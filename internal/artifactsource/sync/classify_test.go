package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	const (
		x = "X"
		y = "Y"
		z = "Z"
	)

	cases := []struct {
		name         string
		base, lh, rh string
		want         Classification
		wantAction   Action
	}{
		{name: "unchanged", base: x, lh: x, rh: x, want: ClsUnchanged, wantAction: ActSkip},
		{name: "local modified", base: x, lh: y, rh: x, want: ClsLocalModified, wantAction: ActUploadModify},
		{name: "remote modified", base: x, lh: x, rh: y, want: ClsRemoteModified, wantAction: ActDownloadModify},
		{name: "converged", base: x, lh: y, rh: y, want: ClsConverged, wantAction: ActSkip},
		{name: "conflict", base: x, lh: y, rh: z, want: ClsConflict, wantAction: ActConflictCopy},
		{name: "local deleted", base: x, lh: "", rh: x, want: ClsLocalDeleted, wantAction: ActUploadDelete},
		{name: "remote deleted", base: x, lh: x, rh: "", want: ClsRemoteDeleted, wantAction: ActDownloadDelete},
		{name: "both deleted", base: x, lh: "", rh: "", want: ClsBothDeleted, wantAction: ActSkip},
		{name: "del-edit conflict", base: x, lh: y, rh: "", want: ClsDelEditConflict, wantAction: ActConflictCopy},
		{name: "edit-del conflict", base: x, lh: "", rh: y, want: ClsEditDelConflict, wantAction: ActDownloadOverDel},
		{name: "local added", base: "", lh: y, rh: "", want: ClsLocalAdded, wantAction: ActUploadAdd},
		{name: "remote added", base: "", lh: "", rh: y, want: ClsRemoteAdded, wantAction: ActDownloadAdd},
		{name: "add conflict", base: "", lh: y, rh: z, want: ClsAddConflict, wantAction: ActConflictCopy},
		{name: "both added same", base: "", lh: y, rh: y, want: ClsBothAddedSame, wantAction: ActSkip},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := Classify(tc.base, tc.lh, tc.rh)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantAction, ActionFor(got))
		})
	}
}

// emptyFileSHA256 is what hashFile returns for a zero-byte file. It matters to
// Classify only as proof that a file which exists never hashes to "", so "" on
// the local side always means absent rather than unknown.
const emptyFileSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// The empty hash carries the whole meaning of "absent on that side", so a
// present file whose hash is merely unknown must never reach Classify as "".
// Pinned here because the wrong half of it is silent: a blank remote checksum
// reads as a remote deletion and takes a local file with it.
func TestClassify_EmptyHashMeansAbsentNotUnknown(t *testing.T) {
	t.Parallel()

	// A zero-byte file still has a digest, so it is present on every side it
	// appears on -- never a deletion.
	assert.Equal(t, ClsUnchanged, Classify(emptyFileSHA256, emptyFileSHA256, emptyFileSHA256))
	assert.Equal(t, ClsLocalAdded, Classify("", emptyFileSHA256, ""))
	assert.Equal(t, ClsRemoteAdded, Classify("", "", emptyFileSHA256))

	// A file still on the remote, arriving with a blank checksum, is
	// indistinguishable from one deleted there -- and the action deletes it
	// locally. filesapi.AllFiles rejects such a row; every other remote
	// manifest source has to do the same before it classifies.
	got := Classify("X", "X", "")
	assert.Equal(t, ClsRemoteDeleted, got)
	assert.Equal(t, ActDownloadDelete, ActionFor(got))
}

func TestActionFor_EditDelConflictDownloadsOverDelete(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ActDownloadOverDel, ActionFor(ClsEditDelConflict))
	assert.NotEqual(t, ActConflictCopy, ActionFor(ClsEditDelConflict))
}

func TestIsConflict(t *testing.T) {
	t.Parallel()

	conflicts := []Classification{ClsConflict, ClsAddConflict, ClsDelEditConflict, ClsEditDelConflict}
	for _, c := range conflicts {
		assert.True(t, c.IsConflict(), "%s", c)
	}

	nonConflicts := []Classification{
		ClsUnchanged, ClsLocalModified, ClsRemoteModified, ClsConverged,
		ClsLocalAdded, ClsRemoteAdded, ClsLocalDeleted, ClsRemoteDeleted,
		ClsBothDeleted, ClsBothAddedSame,
	}
	for _, c := range nonConflicts {
		assert.False(t, c.IsConflict(), "%s", c)
	}
}

func TestClassificationString(t *testing.T) {
	t.Parallel()

	all := []Classification{
		ClsUnchanged, ClsLocalModified, ClsRemoteModified, ClsConverged, ClsConflict,
		ClsLocalAdded, ClsRemoteAdded, ClsAddConflict, ClsLocalDeleted, ClsRemoteDeleted,
		ClsBothDeleted, ClsDelEditConflict, ClsEditDelConflict, ClsBothAddedSame,
	}
	for _, c := range all {
		assert.NotEqual(t, "UNKNOWN", c.String())
	}
	assert.Equal(t, "UNKNOWN", Classification(-1).String())
}
