package vectorstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const embeddingConfigFile = "embedding_config.json"

// ConfigStore handles disk persistence of embedding config.
type ConfigStore struct {
	dataDir string
}

func NewConfigStore(dataDir string) *ConfigStore {
	return &ConfigStore{dataDir: dataDir}
}

// Save writes embedding config to disk.
func (c *ConfigStore) Save(cfg EmbeddingConfig) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("baseUrl is required")
	}
	if cfg.Model == "" {
		return fmt.Errorf("model is required")
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal embedding config: %w", err)
	}
	if err := os.MkdirAll(c.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(c.dataDir, embeddingConfigFile), b, 0644); err != nil {
		return fmt.Errorf("write embedding config: %w", err)
	}
	return nil
}

// Load reads embedding config from disk.
func (c *ConfigStore) Load() (*EmbeddingConfig, error) {
	path := filepath.Join(c.dataDir, embeddingConfigFile)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no embedding config found — call set_embedding_config first")
		}
		return nil, fmt.Errorf("read embedding config: %w", err)
	}
	var cfg EmbeddingConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal embedding config: %w", err)
	}
	return &cfg, nil
}

// Exists returns true if embedding config file exists.
func (c *ConfigStore) Exists() bool {
	path := filepath.Join(c.dataDir, embeddingConfigFile)
	_, err := os.Stat(path)
	return err == nil
}
