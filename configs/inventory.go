package configs

type InventoryConfig struct {
	App        AppConfig        `yaml:"app"`
	Server     ServerConfig     `yaml:"server"`
	Auth       AuthConfig       `yaml:"auth"`
	Postgres   PostgresConfig   `yaml:"postgres"`
	Redis      RedisConfig      `yaml:"redis"`
	OTEL       OTELConfig       `yaml:"otel"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Loki       LokiConfig       `yaml:"loki"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
}
