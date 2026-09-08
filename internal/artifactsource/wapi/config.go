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
// directory. A key the CLI writes that this struct has no field for survives
// a provider save through Extra (see passthrough.go), so the two tools do not
// have to release in step for config.json to round-trip.
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

	// Extra holds every config.json key this build has no field for, and
	// SaveConfig writes it back unchanged. Nil when there are none.
	Extra map[string]json.RawMessage `json:"-"`
}

func (c *Config) UnmarshalJSON(data []byte) error {
	type plain Config
	var known plain
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	extra, err := unknownKeys(data, known)
	if err != nil {
		return err
	}
	known.Extra = extra
	*c = Config(known)
	return nil
}

func (c Config) MarshalJSON() ([]byte, error) {
	type plain Config
	data, err := json.Marshal(plain(c))
	if err != nil {
		return nil, err
	}
	return withUnknownKeys(data, c.Extra)
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
