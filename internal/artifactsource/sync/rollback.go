package sync

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
)

// CLI source: cli/internal/workload/sync/rollback.go

const (
	// RollbackMaxFiles caps the total number of files backed up/restored
	// during a single sync execution to prevent excessive disk/backup usage.
	RollbackMaxFiles = 1000

	rollbackDirName  = ".rollback"
	manifestFileName = "manifest.json"
)

// RollbackManifest tracks file actions for precise restoration.
type RollbackManifest struct {
	BackedUpFiles []string `json:"backedUpFiles"`
	CreatedFiles  []string `json:"createdFiles,omitempty"`
}

// RollbackTree manages backup and restoration of mutated files under
// the sync state directory's .rollback subdirectory.
type RollbackTree struct {
	projectDir  string
	rollbackDir string
	manifest    RollbackManifest
}

// NewRollbackTree constructs a RollbackTree for projectDir.
func NewRollbackTree(projectDir string) *RollbackTree {
	stateDir := wapi.Dir(projectDir)
	return &RollbackTree{
		projectDir:  projectDir,
		rollbackDir: filepath.Join(stateDir, rollbackDirName),
		manifest: RollbackManifest{
			BackedUpFiles: make([]string, 0),
			CreatedFiles:  make([]string, 0),
		},
	}
}

// HasRollback reports whether a .rollback directory exists in projectDir's
// sync state directory (resolved through wapi.Dir, see AcquireLock).
func HasRollback(projectDir string) bool {
	rollbackPath := filepath.Join(wapi.Dir(projectDir), rollbackDirName)
	info, err := os.Stat(rollbackPath)
	return err == nil && info.IsDir()
}

// Init creates the .rollback directory in the sync state directory if missing.
func (r *RollbackTree) Init() error {
	stateDir := wapi.Dir(r.projectDir)
	if _, err := os.Stat(stateDir); err != nil {
		return fmt.Errorf("sync state directory %s does not exist: %w", stateDir, err)
	}

	if err := os.MkdirAll(r.rollbackDir, 0755); err != nil {
		return fmt.Errorf("create rollback directory: %w", err)
	}
	return nil
}

// Backup backs up projectDir/relPath into the rollback tree.
// If the file does not exist locally (will be created by download), it tracks it for removal on rollback.
func (r *RollbackTree) Backup(relPath string) error {
	cleanRel, err := validateAndCleanRelPath(relPath)
	if err != nil {
		return fmt.Errorf("invalid path for backup: %w", err)
	}

	srcPath := filepath.Join(r.projectDir, cleanRel)
	dstPath := filepath.Join(r.rollbackDir, cleanRel)

	info, err := os.Stat(srcPath)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("cannot backup directory %s: only files are supported", cleanRel)
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("backup file %s: %w", cleanRel, err)
		}
		if !containsString(r.manifest.BackedUpFiles, cleanRel) {
			r.manifest.BackedUpFiles = append(r.manifest.BackedUpFiles, cleanRel)
		}
	} else if os.IsNotExist(err) {
		if !containsString(r.manifest.CreatedFiles, cleanRel) {
			r.manifest.CreatedFiles = append(r.manifest.CreatedFiles, cleanRel)
		}
	} else {
		return fmt.Errorf("stat %s: %w", cleanRel, err)
	}

	return r.saveManifest()
}

// Restore restores all backed up files to projectDir and cleans up the rollback tree.
func (r *RollbackTree) Restore() error {
	return RestoreRollback(r.projectDir)
}

// RestoreRollback restores backed-up files from the rollback tree into
// projectDir and deletes the tree.
func RestoreRollback(projectDir string) error {
	if !HasRollback(projectDir) {
		return nil // No rollback tree to restore
	}

	rollbackPath := filepath.Join(wapi.Dir(projectDir), rollbackDirName)
	manifestPath := filepath.Join(rollbackPath, manifestFileName)
	if manifestData, err := os.ReadFile(manifestPath); err == nil {
		var manifest RollbackManifest
		if err := json.Unmarshal(manifestData, &manifest); err == nil {
			// Remove newly created files
			for _, created := range manifest.CreatedFiles {
				cleanCreated, err := validateAndCleanRelPath(created)
				if err != nil {
					return fmt.Errorf("invalid manifest created file %q: %w", created, err)
				}
				p := filepath.Join(projectDir, cleanCreated)
				// Already gone is fine: a failed download removes its
				// own partial file. A removal that actually fails is
				// not, though — callers read a nil error as "the tree
				// is back to its pre-execute state".
				if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("remove created file %s: %w", cleanCreated, err)
				}
			}

			// Restore backed up files
			for _, backedUp := range manifest.BackedUpFiles {
				cleanBackedUp, err := validateAndCleanRelPath(backedUp)
				if err != nil {
					return fmt.Errorf("invalid manifest backed up file %q: %w", backedUp, err)
				}
				src := filepath.Join(rollbackPath, cleanBackedUp)
				dst := filepath.Join(projectDir, cleanBackedUp)
				if _, statErr := os.Stat(src); statErr == nil {
					if err := clearDirForFileRestore(dst); err != nil {
						return fmt.Errorf("restore file %s: %w", cleanBackedUp, err)
					}
					if err := copyFile(src, dst); err != nil {
						return fmt.Errorf("restore file %s: %w", cleanBackedUp, err)
					}
				}
			}

			return os.RemoveAll(rollbackPath)
		}
	}

	// Fallback for stale/legacy rollback trees without a valid manifest: walk rollbackDir
	walkErr := filepath.Walk(rollbackPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(rollbackPath, path)
		if err != nil {
			return fmt.Errorf("relative path calculation for %s: %w", path, err)
		}
		if rel == manifestFileName || strings.HasPrefix(filepath.Base(rel), manifestFileName+".tmp.") {
			return nil
		}
		dst := filepath.Join(projectDir, rel)
		if err := clearDirForFileRestore(dst); err != nil {
			return fmt.Errorf("restore file %s: %w", rel, err)
		}

		return copyFile(path, dst)
	})
	if walkErr != nil {
		return fmt.Errorf("restore rollback tree: %w", walkErr)
	}

	return os.RemoveAll(rollbackPath)
}

// clearDirForFileRestore removes a directory that took over a path the
// rollback manifest recorded as a regular file, so the original bytes can
// be written back over it.
//
// The remote can replace a file with a directory (foo becomes foo/bar).
// Execute frees the path — the conflict rename moves foo aside — and the
// download for foo/bar then re-creates foo as a directory. Without this,
// a later failure leaves Restore unable to write foo back ("is a
// directory"), so .rollback/ survives, every later Plan fails in
// preflight recovery, and the original bytes never come back.
//
// Only empty scaffolding is cleared. Every file this sync wrote under the
// path is a manifest CreatedFile that Restore removed just above, so what
// is left should be directories; a surviving regular file is not this
// sync's to delete and is reported instead.
func clearDirForFileRestore(dst string) error {
	// Nothing to clear unless a directory is actually in the way: an
	// absent or unreadable path is copyFile's to create and report on.
	if !isDir(dst) {
		return nil
	}

	if err := filepath.WalkDir(dst, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(dst, path)
		if relErr != nil {
			rel = path
		}

		return fmt.Errorf("a directory holding %s occupies the path", filepath.ToSlash(rel))
	}); err != nil {
		return err
	}

	return os.RemoveAll(dst)
}

// Discard removes the rollback tree without restoring files.
func (r *RollbackTree) Discard() error {
	if _, err := os.Stat(r.rollbackDir); err == nil {
		return os.RemoveAll(r.rollbackDir)
	}
	return nil
}

// isDir reports whether path is a directory. A path that cannot be
// stat'ed is not one.
func isDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

func (r *RollbackTree) saveManifest() error {
	manifestPath := filepath.Join(r.rollbackDir, manifestFileName)
	data, err := json.MarshalIndent(r.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal rollback manifest: %w", err)
	}
	return atomicWriteFile(manifestPath, data)
}

func atomicWriteFile(path string, data []byte) (err error) {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}

	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file %s: %w", tmp.Name(), err)
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file %s: %w", tmp.Name(), err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmp.Name(), err)
	}
	if err = os.Chmod(tmp.Name(), 0600); err != nil {
		return fmt.Errorf("chmod temp file %s: %w", tmp.Name(), err)
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmp.Name(), path, err)
	}

	if d, derr := os.Open(dir); derr == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	return nil
}

func validateAndCleanRelPath(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path cannot be absolute")
	}
	cleaned := filepath.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path escapes root directory")
	}
	// Guard both state locations: .datarobot/ (current root, which also
	// holds other CLI tool state) and a legacy .wapi/ the CLI has not
	// migrated yet. Backing either up would let a rollback rewrite the
	// very state that describes the rollback.
	parts := strings.Split(filepath.ToSlash(cleaned), "/")
	if len(parts) > 0 && (parts[0] == wapi.RootDirName || parts[0] == wapi.LegacyDirName) {
		return "", fmt.Errorf("path targets the %s state directory", parts[0])
	}
	return cleaned, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Sync()
}

func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
