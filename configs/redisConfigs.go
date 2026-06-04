package configs

import (
	"gopkg.in/yaml.v3"
	"os"
)

type BloomFilterKey string
type BloomFilterConfig struct {
	Key      BloomFilterKey `yaml:"key"`
	Capacity int64          `yaml:"capacity"`
	FPRate   float64        `yaml:"fp_rate"`
}

type RedisConfigurations struct {
	Host         string              `yaml:"host"`
	Port         string              `yaml:"port"`
	Password     string              `yaml:"password"`
	Bloom        bool                `yaml:"bloom_enabled"`
	Cell         bool                `yaml:"cell_enabled"`
	BloomFilters []BloomFilterConfig `yaml:"bloom_filters"`
	DB           int                 `yaml:"db"`
	PoolSize     int                 `yaml:"pool_size"`
	MinIdleConns int                 `yaml:"min_idle_conns"`
}

// TODO: PASS THE YML FILE PATH FROM OUTSIDE
func initializeRedisConfigs() (*RedisConfigurations, error) {
	redisConfigs := &RedisConfigurations{}

	// Load structure from YAML
	yamlBytes, err := os.ReadFile("configs/redis_config.yml")
	if err != nil {
		panic("config: cannot read configs/redis_config.yml: " + err.Error())
	}

	// Swap out the ${ENV_VARS} with their actual values
	expandedYaml := os.ExpandEnv(string(yamlBytes))

	// Unmarshal the modified string into your Configs struct
	err = yaml.Unmarshal([]byte(expandedYaml), &redisConfigs)
	if err != nil {
		panic("config: bad yaml: " + err.Error())
	}

	return redisConfigs, nil

}
