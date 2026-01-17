package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	HTTPPort string `json:"http_port"`
	RTMPPort string `json:"rtmp_port"`
}

// Default configuration
var defaultConfig = Config{
	HTTPPort: "8080",
	RTMPPort: "1935",
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// Load loads configuration from file
func Load() Config {
	path := GetConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultConfig
	}

	// Validate and use defaults for empty values
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = defaultConfig.HTTPPort
	}
	if cfg.RTMPPort == "" {
		cfg.RTMPPort = defaultConfig.RTMPPort
	}

	return cfg
}

// Save saves configuration to file
func Save(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(GetConfigPath(), data, 0644)
}
