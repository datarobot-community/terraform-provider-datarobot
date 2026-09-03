package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func entry(hash string, size int64) FileEntry { return FileEntry{Hash: hash, Size: size} }

func TestDiff_EmptyWhenUnchanged(t *testing.T) {
	t.Parallel()

	base := BaseManifest{"a.py": entry("aaa", 1), "b.py": entry("bbb", 2)}
	plan := Diff(base, base, base)

	assert.True(t, plan.IsEmpty())
	assert.False(t, plan.HasConflicts())
}

func TestDiff_LocalAddModDel(t *testing.T) {
	t.Parallel()

	base := BaseManifest{
		"keep.py": entry("k", 1),
		"mod.py":  entry("old", 2),
		"del.py":  entry("d", 3),
	}
	local := BaseManifest{
		"keep.py": entry("k", 1),
		"mod.py":  entry("new", 4),
		"add.py":  entry("a", 5),
	}
	plan := Diff(base, local, base)
	plan.Sort()

	require.Len(t, plan.Uploads, 2)
	assert.Equal(t, "add.py", plan.Uploads[0].Path)
	assert.Equal(t, ClsLocalAdded, plan.Uploads[0].Classification)
	assert.Equal(t, "mod.py", plan.Uploads[1].Path)
	assert.Equal(t, ClsLocalModified, plan.Uploads[1].Classification)

	require.Len(t, plan.Deletes, 1)
	assert.Equal(t, "del.py", plan.Deletes[0].Path)
	assert.Equal(t, ClsLocalDeleted, plan.Deletes[0].Classification)
	assert.Empty(t, plan.Downloads)
	assert.Empty(t, plan.Conflicts)
}

func TestDiff_RemoteAddModDel(t *testing.T) {
	t.Parallel()

	base := BaseManifest{
		"keep.py": entry("k", 1),
		"mod.py":  entry("old", 2),
		"del.py":  entry("d", 3),
	}
	remote := BaseManifest{
		"keep.py": entry("k", 1),
		"mod.py":  entry("new", 4),
		"add.py":  entry("a", 5),
	}
	plan := Diff(base, base, remote)
	plan.Sort()

	require.Len(t, plan.Downloads, 2)
	assert.Equal(t, "add.py", plan.Downloads[0].Path)
	assert.Equal(t, ClsRemoteAdded, plan.Downloads[0].Classification)
	assert.Equal(t, "mod.py", plan.Downloads[1].Path)
	assert.Equal(t, ClsRemoteModified, plan.Downloads[1].Classification)

	require.Len(t, plan.Deletes, 1)
	assert.Equal(t, "del.py", plan.Deletes[0].Path)
	assert.Equal(t, ClsRemoteDeleted, plan.Deletes[0].Classification)
	assert.Empty(t, plan.Uploads)
	assert.Empty(t, plan.Conflicts)
}

func TestDiff_BothAddedSameIsSkip(t *testing.T) {
	t.Parallel()

	local := BaseManifest{"same.py": entry("h", 10)}
	remote := BaseManifest{"same.py": entry("h", 10)}
	plan := Diff(nil, local, remote)

	assert.True(t, plan.IsEmpty())
}

func TestDiff_ConflictPaths(t *testing.T) {
	t.Parallel()

	base := BaseManifest{"shared.py": entry("X", 10)}
	local := BaseManifest{
		"shared.py": entry("Y", 11),
		"new.py":    entry("L", 3),
	}
	remote := BaseManifest{
		"shared.py": entry("Z", 12),
		"new.py":    entry("R", 4),
	}
	plan := Diff(base, local, remote)
	plan.Sort()

	assert.True(t, plan.HasConflicts())
	assert.Equal(t, []string{"new.py", "shared.py"}, plan.ConflictPaths())
	assert.Equal(t, ClsAddConflict, plan.Conflicts[0].Classification)
	assert.Equal(t, ClsConflict, plan.Conflicts[1].Classification)
	assert.Empty(t, plan.Uploads)
	assert.Empty(t, plan.Downloads)
	assert.Equal(t, int64(12+4), plan.TotalDownloadBytes())
}

func TestDiff_EditDelConflictIsDownload(t *testing.T) {
	t.Parallel()

	base := BaseManifest{"gone.py": entry("X", 10)}
	remote := BaseManifest{"gone.py": entry("Y", 20)}
	plan := Diff(base, nil, remote)

	require.Len(t, plan.Downloads, 1)
	assert.Equal(t, ClsEditDelConflict, plan.Downloads[0].Classification)
	assert.Equal(t, ActDownloadOverDel, plan.Downloads[0].Action)
	assert.Empty(t, plan.Conflicts)
}

func TestDiff_DelEditConflictIsConflict(t *testing.T) {
	t.Parallel()

	base := BaseManifest{"kept.py": entry("X", 10)}
	local := BaseManifest{"kept.py": entry("Y", 11)}
	plan := Diff(base, local, nil)

	require.Len(t, plan.Conflicts, 1)
	assert.Equal(t, ClsDelEditConflict, plan.Conflicts[0].Classification)
	assert.Equal(t, ActConflictCopy, plan.Conflicts[0].Action)
	assert.Empty(t, plan.Downloads)
	assert.Empty(t, plan.Uploads)
	assert.Empty(t, plan.Deletes)
}

func TestDiff_ConvergedIsSkip(t *testing.T) {
	t.Parallel()

	base := BaseManifest{"a.py": entry("X", 10)}
	local := BaseManifest{"a.py": entry("Y", 11)}
	remote := BaseManifest{"a.py": entry("Y", 11)}
	plan := Diff(base, local, remote)

	assert.True(t, plan.IsEmpty())
}

func TestDiff_BothDeletedIsSkip(t *testing.T) {
	t.Parallel()

	base := BaseManifest{"a.py": entry("X", 10)}
	plan := Diff(base, nil, nil)

	assert.True(t, plan.IsEmpty())
}

func TestDiff_Totals(t *testing.T) {
	t.Parallel()

	plan := Diff(nil, BaseManifest{"up.bin": entry("L", 1024)}, BaseManifest{"down.bin": entry("R", 2048)})

	assert.Equal(t, int64(1024), plan.TotalUploadBytes())
	assert.Equal(t, int64(2048), plan.TotalDownloadBytes())
}
