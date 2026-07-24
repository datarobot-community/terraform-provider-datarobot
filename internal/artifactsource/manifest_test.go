package artifactsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestFromFiles(t *testing.T) {
	t.Parallel()

	files := []LocalFile{
		{RelPath: "a.txt", Hash: "aaa", Size: 3},
		{RelPath: "b.txt", Hash: "bbb", Size: 3},
	}

	m := manifestFromFiles(files)
	assert.Equal(t, FileMeta{Hash: "aaa", Size: 3}, m["a.txt"])
	assert.Equal(t, FileMeta{Hash: "bbb", Size: 3}, m["b.txt"])
}
