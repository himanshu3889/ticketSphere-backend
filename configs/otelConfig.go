package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

// OpenTelemetry tracing config struct
type TracingConfig struct {
	ServiceName        string  `yaml:"service_name"`        // eg: "booking-service"
	OTLPEndpoint       string  `yaml:"otel_endpoint"`       // eg: "otel-collector:4317"
	SampleRate         float64 `yaml:"sample_rate"`         // eg: 0.8
	ServiceVersion     string  `yaml:"service_version"`     // eg: "v1.0.0"
	ServiceEnvironment string  `yaml:"service_environment"` // eg: "production"
}

// TODO: PASS THE YML FILE PATH FROM OUTSIDE
func initializeOTELConfigs() (*TracingConfig, error) {
	tracingConfigs := &TracingConfig{}

	// Load structure from YAML
	yamlBytes, err := os.ReadFile("configs/otel_config.yml")
	if err != nil {
		panic("config: cannot read configs/otel_config.yml: " + err.Error())
	}

	// Swap out the ${ENV_VARS} with their actual values
	expandedYaml := os.ExpandEnv(string(yamlBytes))

	// Unmarshal the modified string into your Configs struct
	err = yaml.Unmarshal([]byte(expandedYaml), &tracingConfigs)
	if err != nil {
		panic("config: bad yaml: " + err.Error())
	}

	return tracingConfigs, nil

}
