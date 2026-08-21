package wapi

import (
	"os"
	"path/filepath"
)

// CLI source: cli/internal/workload/wapi/paths.go
//
// History, checkouts, and .wapiignore paths are not ported.

const (
	DirName         = ".wapi"
	ManifestVersion = 1

	configFile        = "config.json"
	manifestFile      = "manifest.json"
	gitignoreFile     = ".gitignore"
	gitignoreContents = "*\n"
)

func wapiDir(projectDir string) string {
	return filepath.Join(projectDir, DirName)
}

func configPath(projectDir string) string {
	return filepath.Join(wapiDir(projectDir), configFile)
}

func manifestPath(projectDir string) string {
	return filepath.Join(wapiDir(projectDir), manifestFile)
}

func gitignorePath(projectDir string) string {
	return filepath.Join(wapiDir(projectDir), gitignoreFile)
}

// Exists reports whether projectDir contains a .wapi/ directory.
func Exists(projectDir string) bool {
	info, err := os.Stat(wapiDir(projectDir))
	return err == nil && info.IsDir()
}
