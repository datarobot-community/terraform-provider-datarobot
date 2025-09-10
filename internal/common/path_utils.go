package common

import (
	"os"
	"path/filepath"
)

// WalkSymlinkSafe has the same interface as filepath.Walk.
// It walks the file tree rooted at root, calling walkFn for each file or directory.
// To avoid complexity with platform specific builds, it does not detect cycles.
func WalkSymlinkSafe(root string, walkFn filepath.WalkFunc) error {

	var walk func(path string, info os.FileInfo, err error) error
	walk = func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return walkFn(path, info, err)
		}

		// If the path is a symlink, resolve it
		// and call walkFn with the resolved path and info.
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return walkFn(path, info, err)
			}
			targetInfo, err := os.Stat(target)
			if err != nil {
				return walkFn(path, targetInfo, err)
			}
			// Call walkFn with resolved path/info
			err = walkFn(path, targetInfo, nil)
			if err != nil {
				if err == filepath.SkipDir && targetInfo.IsDir() {
					return nil
				}
				return err
			}
			return walk(path, targetInfo, nil)
		}

		err = walkFn(path, info, nil)
		if err != nil {
			if err == filepath.SkipDir && info.IsDir() {
				return nil
			}
			return err
		}
		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return walkFn(path, info, err)
			}
			for _, entry := range entries {
				entryPath := filepath.Join(path, entry.Name())
				entryInfo, err := os.Lstat(entryPath)
				if err != nil {
					if err := walkFn(entryPath, entryInfo, err); err != nil {
						if err == filepath.SkipDir {
							continue
						}
						return err
					}
					continue
				}
				if err := walk(entryPath, entryInfo, nil); err != nil {
					if err == filepath.SkipDir {
						continue
					}
					return err
				}
			}
		}

		return nil
	}

	info, err := os.Lstat(root)
	if err != nil {
		return walkFn(root, nil, err)
	}
	return walk(root, info, nil)
}
