package artifactsource

import (
	"crypto/sha256"
	"encoding/hex"
)

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
