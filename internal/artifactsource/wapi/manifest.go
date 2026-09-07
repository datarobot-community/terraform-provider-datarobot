package wapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// CLI source: cli/internal/workload/wapi/manifest.go

// FileMeta is one BASE file: SHA-256 hex hash and size.
type FileMeta struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// Manifest is the state directory's manifest.json — BASE from the last
// successful sync.
type Manifest struct {
	Version         int                 `json:"version"`
	SyncedAt        *time.Time          `json:"syncedAt"`
	SyncedVersionID *string             `json:"syncedVersionId"`
	Files           map[string]FileMeta `json:"files"`
}

// LoadManifest reads manifest.json. Nil Files becomes an empty map, and a
// version this build does not support is reported as corrupted.
func LoadManifest(projectDir string) (Manifest, error) {
	path := manifestPath(projectDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, ErrNotInitialized
		}
		return Manifest{}, &CorruptedError{Path: path, Err: err}
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, &CorruptedError{Path: path, Err: err}
	}
	if m.Files == nil {
		m.Files = map[string]FileMeta{}
	}
	// The CLI pins this with `validate:"eq=1"`. Reading a version this build
	// does not know would treat some future BASE format as if it were v1, and a
	// misread BASE is a wrong three-way merge rather than a failed one.
	if m.Version != ManifestVersion {
		return Manifest{}, &CorruptedError{
			Path: path,
			Err:  fmt.Errorf("unsupported manifest version %d, want %d", m.Version, ManifestVersion),
		}
	}

	return m, nil
}

// SaveManifest atomically writes manifest.json. A zero Version is written as
// ManifestVersion.
func SaveManifest(projectDir string, m Manifest) error {
	if err := writeManifest(projectDir, m); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotInitialized
		}
		return err
	}
	return nil
}

func writeManifest(projectDir string, m Manifest) error {
	// Both defaults keep a manifest built from a filesystem walk, rather than
	// loaded from disk, writable without the caller restating constants: nil
	// Files would emit "files": null, and a zero Version would emit
	// "version": 0, which the CLI rejects as corrupted.
	if m.Version == 0 {
		m.Version = ManifestVersion
	}
	if m.Files == nil {
		m.Files = map[string]FileMeta{}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return atomicWriteFile(manifestPath(projectDir), data)
}
