package artifactsource

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

type walkEntry struct {
	AbsPath string
	RelPath string
	Size    int64
}

// walkDirectory enumerates regular files under root. Symlinks are skipped.
// Ignored directories are pruned. Returned entries are sorted by RelPath.
func walkDirectory(root string, ignore IgnoreFunc) ([]walkEntry, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("abs %s: %w", root, err)
	}

	var entries []walkEntry

	visit := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %s: %w", path, walkErr)
		}

		if path == absRoot {
			return nil
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return fmt.Errorf("relativize %s under %s: %w", path, absRoot, err)
		}

		normRel := filesapi.NormalizePath(rel)

		switch {
		case d.Type()&os.ModeSymlink != 0:
			return nil
		case d.IsDir():
			if ignore != nil && ignore(normRel, true) {
				return filepath.SkipDir
			}
			return nil
		case !d.Type().IsRegular():
			return nil
		}

		if ignore != nil && ignore(normRel, false) {
			return nil
		}

		fi, err := d.Info()
		if err != nil {
			return fmt.Errorf("info %s: %w", path, err)
		}

		entries = append(entries, walkEntry{
			AbsPath: path,
			RelPath: normRel,
			Size:    fi.Size(),
		})

		return nil
	}

	if err := filepath.WalkDir(absRoot, visit); err != nil {
		return nil, err
	}

	sortWalkEntries(entries)

	return entries, nil
}

func sortWalkEntries(entries []walkEntry) {
	// insertion sort is fine for typical tree sizes; keeps deps minimal
	for i := 1; i < len(entries); i++ {
		j := i
		for j > 0 && entries[j-1].RelPath > entries[j].RelPath {
			entries[j-1], entries[j] = entries[j], entries[j-1]
			j--
		}
	}
}
