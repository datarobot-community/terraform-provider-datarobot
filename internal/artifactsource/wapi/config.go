package wapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// CLI source: cli/internal/workload/wapi/config.go
//
// JSON field names match the CLI so a mixed CLI/TF tree can share one state
// directory. Every key the CLI writes has a field here, because SaveConfig
// re-marshals the whole struct: a key this struct does not carry is a key a
// provider write silently deletes from a CLI-written config.json.
//
// CLIVersion is the JSON key; Initialize writes ProviderWriter.
// The Go field is still called CLIVersion because the JSON key is cliVersion.
// We do not invent a providerVersion key. On init we store "terraform-provider-datarobot" in that field.

const ProviderWriter = "terraform-provider-datarobot"

// Config is the state directory's config.json — artifact identity and
// last-synced catalog pointers.
type Config struct {
	ArtifactID          string  `json:"artifactId"`
	CatalogID           *string `json:"catalogId"`
	LastSyncedVersionID *string `json:"lastSyncedVersionId"`

	// LastBuiltVersionID is the code version the CLI last built an image from.
	// The provider never sets it: it is carried so that loading a CLI-written
	// config and saving it back preserves the value, because the CLI reads it to
	// decide whether code moved since that build.
	LastBuiltVersionID *string   `json:"lastBuiltVersionId"`
	CreatedAt          time.Time `json:"createdAt"`
	CLIVersion         string    `json:"cliVersion"`
}

// LoadConfig reads config.json.
func LoadConfig(projectDir string) (Config, error) {
	path := configPath(projectDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, ErrNotInitialized
		}
		return Config{}, &CorruptedError{Path: path, Err: err}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, &CorruptedError{Path: path, Err: err}
	}

	return cfg, nil
}

// SaveConfig atomically writes config.json.
func SaveConfig(projectDir string, c Config) error {
	if err := writeConfig(projectDir, c); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotInitialized
		}
		return err
	}
	return nil
}

func writeConfig(projectDir string, c Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return atomicWriteFile(configPath(projectDir), data)
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
