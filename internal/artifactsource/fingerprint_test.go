package artifactsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirectoryFingerprint_OrderStable(t *testing.T) {
	t.Parallel()

	files := []LocalFile{
		{RelPath: "a.txt", Hash: "aaa"},
		{RelPath: "b.txt", Hash: "bbb"},
	}

	h1 := directoryFingerprint(files)
	h2 := directoryFingerprint(files)
	assert.Equal(t, h1, h2)
	assert.NotEmpty(t, h1)
}

func TestDirectoryFingerprint_DifferentPaths(t *testing.T) {
	t.Parallel()

	filesA := []LocalFile{{RelPath: "a.txt", Hash: "same"}}
	filesB := []LocalFile{{RelPath: "b.txt", Hash: "same"}}

	assert.NotEqual(t, directoryFingerprint(filesA), directoryFingerprint(filesB))
}

func TestDirectoryFingerprint_DifferentHashes(t *testing.T) {
	t.Parallel()

	filesA := []LocalFile{{RelPath: "a.txt", Hash: "one"}}
	filesB := []LocalFile{{RelPath: "a.txt", Hash: "two"}}

	assert.NotEqual(t, directoryFingerprint(filesA), directoryFingerprint(filesB))
}

func TestDirectoryFingerprint_NULSeparatorPreventsAmbiguousConcatenation(t *testing.T) {
	t.Parallel()

	// Each entry is hashed as relPath + NUL + fileHash. Without the NUL byte,
	// ("ab", "cdef") and ("a", "bcdef") would serialize to the same byte
	// sequence "abcdef" and collide. The separator keeps distinct trees distinct
	// even when path/hash boundaries could otherwise align.
	filesA := []LocalFile{{RelPath: "ab", Hash: "cdef"}}
	filesB := []LocalFile{{RelPath: "a", Hash: "bcdef"}}

	assert.NotEqual(t, directoryFingerprint(filesA), directoryFingerprint(filesB))
}
