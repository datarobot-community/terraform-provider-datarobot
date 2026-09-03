package wapi

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// CLI source: cli/internal/workload/wapi/init.go
//
// Provider differences: no history.log; no .drignore write (PR2).

// InitOptions are persisted to config.json. Empty catalog/version become JSON null.
type InitOptions struct {
	ArtifactID          string
	CatalogID           string
	LastSyncedVersionID string
}

// Initialize creates .wapi/ with config.json, empty BASE manifest.json, and
// .gitignore "*". Returns ErrAlreadyLinked if .wapi/ already exists.
func Initialize(projectDir string, opts InitOptions) (err error) {
	if opts.ArtifactID == "" {
		return fmt.Errorf("artifactId is required")
	}

	if err = os.Mkdir(wapiDir(projectDir), 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrAlreadyLinked
		}
		return fmt.Errorf("create .wapi/ directory: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(wapiDir(projectDir))
		}
	}()

	if err = atomicWriteFile(gitignorePath(projectDir), []byte(gitignoreContents)); err != nil {
		return err
	}

	now := time.Now().UTC()
	cfg := Config{
		ArtifactID:          opts.ArtifactID,
		CatalogID:           stringPtr(opts.CatalogID),
		LastSyncedVersionID: stringPtr(opts.LastSyncedVersionID),
		CreatedAt:           now,
		CLIVersion:          ProviderWriter,
	}
	if err = writeConfig(projectDir, cfg); err != nil {
		return err
	}
	if err = writeManifest(projectDir, Manifest{Version: ManifestVersion}); err != nil {
		return err
	}

	return nil
}
