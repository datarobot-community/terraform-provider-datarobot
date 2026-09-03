package wapi

import (
	"os"
	"path/filepath"
)

// CLI source: cli/internal/workload/wapi/paths.go
//
// History, checkouts, rollback, and .wapiignore paths are not ported.

const (
	// RootDirName is the single directory a project gives DataRobot. The CLI
	// already keeps its tool state under it, so sync state joins it rather
	// than claiming a second dot-entry at the project root.
	RootDirName = ".datarobot"

	// StateDirName is the sync state directory, nested inside RootDirName.
	StateDirName = "workload"

	// LegacyDirName is where sync state lived before it moved under
	// RootDirName. The CLI's EnsureMigrated relocates it on any dr command
	// that touches state; the provider reads it where it still stands but
	// never moves it.
	LegacyDirName = ".wapi"

	ManifestVersion = 1

	configFile        = "config.json"
	manifestFile      = "manifest.json"
	gitignoreFile     = ".gitignore"
	gitignoreContents = "*\n"
)

// Dir is the directory holding this project's BASE sync state:
// <projectDir>/.datarobot/workload, falling back to a legacy <projectDir>/.wapi
// that the CLI's EnsureMigrated has not moved yet.
//
// The fallback is a stat on every call rather than cached state, which is what
// keeps a mixed CLI/Terraform tree on one file: whichever side wrote last, the
// other resolves the same location. The provider does not perform the migration
// itself. Relocating a directory is the CLI's job, and a Terraform apply that
// moved files the configuration never named would be doing it behind the user's
// back.
func Dir(projectDir string) string {
	current := filepath.Join(projectDir, RootDirName, StateDirName)
	if dirExists(current) {
		return current
	}

	if legacy := legacyDir(projectDir); dirExists(legacy) {
		return legacy
	}

	return current
}

func legacyDir(projectDir string) string {
	return filepath.Join(projectDir, LegacyDirName)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func configPath(projectDir string) string {
	return filepath.Join(Dir(projectDir), configFile)
}

func manifestPath(projectDir string) string {
	return filepath.Join(Dir(projectDir), manifestFile)
}

func gitignorePath(projectDir string) string {
	return filepath.Join(Dir(projectDir), gitignoreFile)
}

// Exists reports whether projectDir holds a state directory, at either the
// current or the legacy location.
func Exists(projectDir string) bool {
	return dirExists(Dir(projectDir))
}
