package configs

import (
	"gopkg.in/yaml.v3"
	"os"
)

type RMQConfigurations struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// TODO: PASS THE YML FILE PATH FROM OUTSIDE
func initializeRMQConfigs() (*RMQConfigurations, error) {
	rmqConfigs := &RMQConfigurations{}

	// Load structure from YAML
	yamlBytes, err := os.ReadFile("configs/rmq_config.yml")
	if err != nil {
		panic("config: cannot read configs/rmq_config.yml: " + err.Error())
	}

	// Swap out the ${ENV_VARS} with their actual values
	expandedYaml := os.ExpandEnv(string(yamlBytes))

	// Unmarshal the modified string into your Configs struct
	err = yaml.Unmarshal([]byte(expandedYaml), &rmqConfigs)
	if err != nil {
		panic("config: bad yaml: " + err.Error())
	}

	return rmqConfigs, nil

}
