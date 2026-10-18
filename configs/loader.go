package configs

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads the YAML file at path into T, expanding $ENV vars first.
func Load[T any](path string) (*T, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config not found: %w", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config read failed: %w", err)
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg T
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("bad yaml in %s: %w", filepath.Base(path), err)
	}
	return &cfg, nil
}
