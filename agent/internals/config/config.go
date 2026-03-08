package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root agent configuration.
type Config struct {
	Agent  AgentConfig   `yaml:"agent"`
	Inputs []InputConfig `yaml:"inputs"`
	Output OutputConfig  `yaml:"output"`
}

// AgentConfig controls pipeline behavior.
type AgentConfig struct {
	FlushInterval time.Duration `yaml:"flush_interval"`
	BatchSize     int           `yaml:"batch_size"`
	LogLevel      string        `yaml:"log_level"`
}

// InputConfig describes a single input source.
type InputConfig struct {
	Type    string            `yaml:"type"`
	Options map[string]string `yaml:"options"`
}

// OutputConfig describes the export destination.
type OutputConfig struct {
	Endpoint    string        `yaml:"endpoint"`
	Timeout     time.Duration `yaml:"timeout"`
	MaxRetries  int           `yaml:"max_retries"`
	Compression bool          `yaml:"compression"`
}

// DefaultConfig returns sane production defaults.
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			FlushInterval: 2 * time.Second,
			BatchSize:     500,
			LogLevel:      "info",
		},
		Inputs: []InputConfig{
			{Type: "docker"},
		},
		Output: OutputConfig{
			Endpoint:    "http://localhost:3010",
			Timeout:     5 * time.Second,
			MaxRetries:  3,
			Compression: true,
		},
	}
}

// Load reads configuration from a YAML file, falling back to defaults.
// It checks (in order): the path passed in, $FEATURETRACE_CONFIG env var,
// then ./config.yaml.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		path = os.Getenv("FEATURETRACE_CONFIG")
	}
	if path == "" {
		path = "config.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file — run with defaults
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}

