package artifactsource

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// FingerprintDirectory returns a deterministic SHA-256 hex digest of uploadable
// files under dir after applying ignore. extra is merged in (e.g. a pending
// .drignore that will be written on apply) and the combined list is sorted.
func FingerprintDirectory(dir string, ignore IgnoreFunc, extra []LocalFile) (string, error) {
	files, _, err := collectLocalFiles(dir, ignore, true)
	if err != nil {
		return "", err
	}
	if len(extra) > 0 {
		files = append(files, extra...)
		sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })
	}

	return directoryFingerprint(files), nil
}

// directoryFingerprint returns a deterministic SHA-256 hex digest of the tree.
// Files must already be sorted by RelPath. Each entry contributes relPath + NUL + fileHash.
func directoryFingerprint(files []LocalFile) string {
	h := sha256.New()

	for _, f := range files {
		_, _ = h.Write([]byte(f.RelPath))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(f.Hash))
	}

	return hex.EncodeToString(h.Sum(nil))
}
