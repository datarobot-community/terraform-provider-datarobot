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

// Manifest is .wapi/manifest.json — BASE from the last successful sync.
type Manifest struct {
	Version         int                 `json:"version"`
	SyncedAt        *time.Time          `json:"syncedAt"`
	SyncedVersionID *string             `json:"syncedVersionId"`
	Files           map[string]FileMeta `json:"files"`
}

// LoadManifest reads .wapi/manifest.json. Nil Files becomes an empty map.
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

	return m, nil
}

// SaveManifest atomically writes .wapi/manifest.json.
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
	if m.Files == nil {
		m.Files = map[string]FileMeta{}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return atomicWriteFile(manifestPath(projectDir), data)
}
