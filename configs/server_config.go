package configs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfigurations struct {
	ServerName string `yaml:"server_name"`
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Addr       string `yaml:"-"` // Build it using host and port
	MachineID  int64  `yaml:"machine_id"`
}

func initializeServerConfigs(yml_file_path string) (*ServerConfigurations, error) {
	serverConfigs := &ServerConfigurations{}

	// Load structure from YAML
	yamlBytes, err := os.ReadFile(yml_file_path)
	if err != nil {
		msg := fmt.Sprintf("config: cannot read %s: %s", yml_file_path, err.Error())
		panic(msg)
	}

	// Swap out the ${ENV_VARS} with their actual values
	expandedYaml := os.ExpandEnv(string(yamlBytes))

	// Unmarshal the modified string into your Configs struct
	err = yaml.Unmarshal([]byte(expandedYaml), &serverConfigs)
	if err != nil {
		panic("config: bad yaml: " + err.Error())
	}

	// Build the address from the host and port
	addr := fmt.Sprintf("%s:%d", serverConfigs.Host, serverConfigs.Port)
	serverConfigs.Addr = addr
	return serverConfigs, nil

}
