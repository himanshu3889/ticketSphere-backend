package configs

import (
	"fmt"
)

type BloomFilterKey string

type BloomFilterConfig struct {
	Key      BloomFilterKey `yaml:"key"`
	Capacity int64          `yaml:"capacity"`
	FPRate   float64        `yaml:"fp_rate"`
}

type AppConfig struct {
	Name     string `yaml:"name"`
	LogLevel string `yaml:"log_level"`
	Env      string `yaml:"env"`
}

type ServerConfig struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	MachineID int64  `yaml:"machine_id"`
}

func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type AuthConfig struct {
	JWTSecret     string `yaml:"jwt_secret"`
	JWTAudience   string `yaml:"jwt_audience"`
	JWTIssuer     string `yaml:"jwt_issuer"`
	GatewaySecret string `yaml:"gateway_secret"`
}

type PostgresConfig struct {
	Host               string `yaml:"host"`
	Port               string `yaml:"port"`
	User               string `yaml:"user"`
	DB                 string `yaml:"db"`
	Password           string `yaml:"password"`
	MaxOpenConns       int    `yaml:"max_open_conns"`
	MaxIdleConns       int    `yaml:"max_idle_conns"`
	ConnMaxLifetimeMin int    `yaml:"conn_max_lifetime_min"`
}

type MongoConfig struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	Database    string `yaml:"database"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	MinPoolSize uint64 `yaml:"min_pool_size"`
	MaxPoolSize uint64 `yaml:"max_pool_size"`
}

type RedisConfig struct {
	Host         string              `yaml:"host"`
	Port         string              `yaml:"port"`
	Password     string              `yaml:"password"`
	Bloom        bool                `yaml:"bloom_enabled"`
	Cell         bool                `yaml:"cell_enabled"`
	BloomFilters []BloomFilterConfig `yaml:"bloom_filters"`
	DB           int                 `yaml:"db"`
	PoolSize     int                 `yaml:"pool_size"`
	MinIdleConns int                 `yaml:"min_idle_conns"`

	PoolTimeoutMs  int `yaml:"pool_timeout_ms"`
	ReadTimeoutMs  int `yaml:"read_timeout_ms"`
	WriteTimeoutMs int `yaml:"write_timeout_ms"`
	DialTimeoutMs  int `yaml:"dial_timeout_ms"`
}

type ConsumerTuning struct {
	Workers         int `yaml:"workers"`
	Prefetch        int `yaml:"prefetch"`
	BatchSize       int `yaml:"batch_size"`
	FlushIntervalMs int `yaml:"flush_interval_ms"`
}

type RMQConsumersConfig struct {
	Postgres ConsumerTuning `yaml:"postgres"`
}

type RMQConfig struct {
	Host      string             `yaml:"host"`
	Port      string             `yaml:"port"`
	User      string             `yaml:"user"`
	Password  string             `yaml:"password"`
	Consumers RMQConsumersConfig `yaml:"consumers"`
}

type OTELConfig struct {
	ServiceName        string  `yaml:"service_name"`
	OTLPEndpoint       string  `yaml:"otel_endpoint"`
	SampleRate         float64 `yaml:"sample_rate"`
	ServiceVersion     string  `yaml:"service_version"`
	ServiceEnvironment string  `yaml:"service_environment"`
}

type PrometheusConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     string `yaml:"port"`
}

type LokiConfig struct {
	Url string `yaml:"url"`
}

type RateLimitConfig struct {
	PerMinute int `yaml:"per_minute"`
}

type PaymentGatewayConfig struct {
	BaseURL       string `yaml:"base_url"`
	TimeoutSec    int    `yaml:"timeout_sec"`
	WebhookSecret string `yaml:"webhook_secret"`
	WebhookURL    string `yaml:"webhook_url"`
}
