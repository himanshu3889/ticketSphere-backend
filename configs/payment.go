package configs

type PaymentConfig struct {
	App            AppConfig            `yaml:"app"`
	Server         ServerConfig         `yaml:"server"`
	Auth           AuthConfig           `yaml:"auth"`
	Postgres       PostgresConfig       `yaml:"postgres"`
	Redis          RedisConfig          `yaml:"redis"`
	RMQ            RMQConfig            `yaml:"rmq"`
	PaymentGateway PaymentGatewayConfig `yaml:"payment_gateway"`
	OTEL           OTELConfig           `yaml:"otel"`
	Prometheus     PrometheusConfig     `yaml:"prometheus"`
	Loki           LokiConfig           `yaml:"loki"`
}
