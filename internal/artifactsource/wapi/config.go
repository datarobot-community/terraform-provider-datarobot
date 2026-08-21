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
// JSON field names match the CLI so a mixed CLI/TF tree can share .wapi/.
// CLIVersion is the JSON key; Initialize writes ProviderWriter.
// The Go field is still called CLIVersion because the JSON key is cliVersion. 
// We do not invent a providerVersion key. On init we store "terraform-provider-datarobot" in that field.

const ProviderWriter = "terraform-provider-datarobot"

// Config is .wapi/config.json — artifact identity and last-synced catalog pointers.
type Config struct {
	ArtifactID          string    `json:"artifactId"`
	CatalogID           *string   `json:"catalogId"`
	LastSyncedVersionID *string   `json:"lastSyncedVersionId"`
	CreatedAt           time.Time `json:"createdAt"`
	CLIVersion          string    `json:"cliVersion"`
}

// LoadConfig reads .wapi/config.json.
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

// SaveConfig atomically writes .wapi/config.json.
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
