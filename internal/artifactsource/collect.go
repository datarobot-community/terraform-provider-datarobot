package artifactsource

import (
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// CollectLocalFiles walks dir, applies ignore, and hashes every included
// regular file. It is the one walk-and-hash primitive in this package:
// internal/artifactsource/sync's Engine builds the three-way sync LOCAL
// manifest from it, and FingerprintDirectory digests what it returns.
//
// An empty (or fully ignored) directory returns an empty slice rather than
// an error, so Plan succeeds on a source.dir with no files yet.
func CollectLocalFiles(dir string, ignore IgnoreFunc) ([]LocalFile, error) {
	entries, err := walkDirectory(dir, ignore)
	if err != nil {
		return nil, err
	}

	files := make([]LocalFile, 0, len(entries))

	for _, e := range entries {
		if err := filesapi.SafeRelPath(e.RelPath); err != nil {
			return nil, fmt.Errorf("invalid path %q: %w", e.RelPath, err)
		}

		hash, size, err := hashFile(e.AbsPath)
		if err != nil {
			return nil, err
		}

		files = append(files, LocalFile{
			RelPath: e.RelPath,
			AbsPath: e.AbsPath,
			Size:    size,
			Hash:    hash,
		})
	}

	return files, nil
}
