package artifactsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffPushOnly_NoChanges(t *testing.T) {
	t.Parallel()

	base := Manifest{
		"a.txt": {Hash: "aaa", Size: 3},
		"b.txt": {Hash: "bbb", Size: 3},
	}
	local := Manifest{
		"a.txt": {Hash: "aaa", Size: 3},
		"b.txt": {Hash: "bbb", Size: 3},
	}

	plan := DiffPushOnly(base, local)
	assert.True(t, plan.IsEmpty())
}

func TestDiffPushOnly_AddedModifiedDeleted(t *testing.T) {
	t.Parallel()

	base := Manifest{
		"keep.txt":    {Hash: "same", Size: 4},
		"changed.txt": {Hash: "old", Size: 3},
		"removed.txt": {Hash: "gone", Size: 4},
	}
	local := Manifest{
		"keep.txt":    {Hash: "same", Size: 4},
		"changed.txt": {Hash: "new", Size: 3},
		"added.txt":   {Hash: "add", Size: 3},
	}

	plan := DiffPushOnly(base, local)
	assert.Equal(t, []string{"added.txt", "changed.txt"}, plan.Uploads)
	assert.Equal(t, []string{"removed.txt"}, plan.Deletes)
}

func TestDiffPushOnly_BaseOnlyFileSchedulesDeleteNotUpload(t *testing.T) {
	t.Parallel()

	// Push-only / delete-only diff: local is the source of truth. When a path
	// exists in base (last successful push, i.e. Terraform state) but is absent
	// locally, we do not download it back — PushPlan has no Downloads bucket at
	// all (unlike the CLI three-way sync). The only action is a remote delete.
	//
	// For that path the upload list stays empty: there is nothing local to push,
	// and we never pull remote-only files into the plan.
	base := Manifest{
		"keep.txt":        {Hash: "same", Size: 4},
		"remote-only.txt": {Hash: "remote", Size: 6},
	}
	local := Manifest{
		"keep.txt": {Hash: "same", Size: 4},
	}

	plan := DiffPushOnly(base, local)

	assert.Empty(t, plan.Uploads)
	assert.Equal(t, []string{"remote-only.txt"}, plan.Deletes)
}
