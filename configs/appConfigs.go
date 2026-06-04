package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

type AppConfigurations struct {
	App struct {
		Name     string `yaml:"name"`
		LogLevel string `yaml:"log_level"`
		Env      string `yaml:"env"`
	} `yaml:"app"`

	Auth struct {
		JWTSecret     string `yaml:"JWT_SECRET"`
		GatewaySecret string `yaml:"GATEWAY_SECRET"`
	} `yaml:"-"`

	Postgres struct {
		Host               string `yaml:"host"`
		Port               string `yaml:"port"`
		User               string `yaml:"user"`
		DB                 string `yaml:"db"`
		Password           string `yaml:"password"`
		MaxOpenConns       int    `yaml:"max_open_conns"`
		MaxIdleConns       int    `yaml:"max_idle_conns"`
		ConnMaxLifetimeMin int    `yaml:"conn_max_lifetime_min"`
	} `yaml:"postgres"`

	MongoDB struct {
		Host        string `yaml:"host"`
		Port        string `yaml:"port"`
		Database    string `yaml:"database"`
		Username    string `yaml:"username"`
		Password    string `yaml:"password"`
		MinPoolSize uint64 `yaml:"MinPoolSize"`
		MaxPoolSize uint64 `yaml:"MaxPoolSize"`
	} `yaml:"mongodb"`

	Elasticsearch struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"elasticsearch"`

	Kafka struct {
		Host    string   `yaml:"host"`
		Port    string   `yaml:"port"`
		Brokers []string `yaml:"brokers"`
	} `yaml:"kafka"`

	Prometheus struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Port     string `yaml:"port"`
	} `yaml:"prometheus"`

	Grafana struct {
		Port string `yaml:"port"`
	} `yaml:"grafana"`

	Loki struct {
		Url string `yaml:"url"`
	} `yaml:"loki"`

	RateLimit struct {
		PerMinute int `yaml:"per_minute"`
	} `yaml:"rate_limit"`
}

// TODO: LATER PASS THE FILE NAME FROM OUTSIDE
// DON'T CREAT THE PANIC HERE RETURN THE ERROR
func initializeAppConfigs() (*AppConfigurations, error) {
	appConfigs := &AppConfigurations{}

	// Load structure from YAML
	yamlBytes, err := os.ReadFile("configs/app_config.yml")
	if err != nil {
		panic("config: cannot read configs/app_config.yml: " + err.Error())
	}

	// Swap out the ${ENV_VARS} with their actual values
	expandedYaml := os.ExpandEnv(string(yamlBytes))

	// Unmarshal the modified string into your Configs struct
	err = yaml.Unmarshal([]byte(expandedYaml), &appConfigs)
	if err != nil {
		panic("config: bad yaml: " + err.Error())
	}
	return appConfigs, nil
}
