package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Logging  LoggingConfig  `yaml:"logging"`
	Auth     AuthConfig     `yaml:"auth"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

type MetricsConfig struct {
	VictoriaMetricsURL string `yaml:"victoriametrics_url"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type AuthConfig struct {
	JWTSecret     string `yaml:"jwt_secret"`
	EncryptionKey string `yaml:"encryption_key"`
}

// Load loads configuration from YAML file and environment variables
func Load(configPath string) (*Config, error) {
	cfg := &Config{}

	// Load from YAML file if provided
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Override with environment variables
	cfg.overrideFromEnv()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// overrideFromEnv overrides configuration values with environment variables
func (c *Config) overrideFromEnv() {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}

	if v := os.Getenv("DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Port = port
		}
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.Database.Database = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DB_SSL_MODE"); v != "" {
		c.Database.SSLMode = v
	}

	if v := os.Getenv("VICTORIAMETRICS_URL"); v != "" {
		c.Metrics.VictoriaMetricsURL = v
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		c.Logging.Format = v
	}

	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.Auth.JWTSecret = strings.TrimSpace(v)
	}
	if v := os.Getenv("ENCRYPTION_KEY"); v != "" {
		c.Auth.EncryptionKey = strings.TrimSpace(v)
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Validate server configuration
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errors = append(errors, "server.port must be between 1 and 65535")
	}

	// Validate database configuration
	if c.Database.Host == "" {
		errors = append(errors, "database.host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		errors = append(errors, "database.port must be between 1 and 65535")
	}
	if c.Database.Database == "" {
		errors = append(errors, "database.database is required")
	}
	if c.Database.User == "" {
		errors = append(errors, "database.user is required")
	}
	if c.Database.Password == "" {
		errors = append(errors, "database.password is required")
	}

	// Validate metrics configuration
	if c.Metrics.VictoriaMetricsURL == "" {
		errors = append(errors, "metrics.victoriametrics_url is required")
	}

	// Validate logging configuration
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(c.Logging.Level)] {
		errors = append(errors, "logging.level must be one of: debug, info, warn, error")
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[strings.ToLower(c.Logging.Format)] {
		errors = append(errors, "logging.format must be one of: json, text")
	}

	// Validate auth configuration
	if c.Auth.JWTSecret == "" {
		errors = append(errors, "auth.jwt_secret is required")
	}
	if len(c.Auth.JWTSecret) < 32 {
		errors = append(errors, "auth.jwt_secret must be at least 32 characters")
	}
	if c.Auth.EncryptionKey == "" {
		errors = append(errors, "auth.encryption_key is required")
	}
	if len(c.Auth.EncryptionKey) != 32 {
		errors = append(errors, "auth.encryption_key must be exactly 32 bytes")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
