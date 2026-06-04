package configs

import (
	"errors"
	"fmt"
	"os"
)

type Configs struct {
	App    *AppConfigurations
	Redis  *RedisConfigurations
	Server *ServerConfigurations
}

// Initialize all the configs
func InitializeConfigs(server_yml_file_path string) (*Configs, error) {
	// Check if file exist or not
	if _, err := os.Stat(server_yml_file_path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("configuration file not found: %s", server_yml_file_path)
		}
		// Catch other potential errors like permission issues
		return nil, fmt.Errorf("error accessing configuration file: %w", err)
	}

	// App configs
	appConfigs, err := initializeAppConfigs()
	if err != nil {
		return nil, err
	}

	// Redis configs
	redisConfigs, err := initializeRedisConfigs()
	if err != nil {
		return nil, err
	}

	// Server configs
	serverConfigs, err := initializeServerConfigs(server_yml_file_path)
	if err != nil {
		return nil, err
	}

	// Build and return the config struct
	configs := &Configs{App: appConfigs, Redis: redisConfigs, Server: serverConfigs}
	return configs, nil
}
