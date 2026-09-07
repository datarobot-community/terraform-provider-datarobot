package wapi

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CLI source: cli/internal/workload/wapi/init.go
//
// Provider differences: no history.log; no .drignore write (PR2).
//
// The CLI leaves an incomplete tree in place for the user to inspect. The
// provider rolls it back instead: a half-written state directory would make the
// next apply resolve a config.json with no manifest beside it, and a Terraform
// run that failed should not change what the following one reads.

// InitOptions are persisted to config.json. Empty catalog/version become JSON null.
type InitOptions struct {
	ArtifactID          string
	CatalogID           string
	LastSyncedVersionID string
}

// Initialize creates the state directory with config.json, an empty BASE
// manifest.json, and .gitignore "*". Returns ErrAlreadyLinked if the project is
// already linked at either the current or the legacy location.
func Initialize(projectDir string, opts InitOptions) (err error) {
	if opts.ArtifactID == "" {
		return fmt.Errorf("artifactId is required")
	}

	// Dir resolves to the legacy location when one is present, so an
	// un-migrated project reports ErrAlreadyLinked rather than acquiring a
	// second state directory the CLI would then refuse to merge.
	stateDir := Dir(projectDir)

	rootDir := filepath.Dir(stateDir)
	if err = os.MkdirAll(rootDir, 0o755); err != nil {
		return fmt.Errorf("create %s directory: %w", rootDir, err)
	}

	if err = os.Mkdir(stateDir, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrAlreadyLinked
		}
		return fmt.Errorf("create %s directory: %w", stateDir, err)
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(stateDir)
			// Best effort: succeeds only if this call created it and nothing
			// else has since been written under it.
			_ = os.Remove(rootDir)
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
