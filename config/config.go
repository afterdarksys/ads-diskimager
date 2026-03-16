package config

import (
	"encoding/json"
	"os"
)

// ServerConfig holds the server configuration.
type ServerConfig struct {
	BindAddress string   `json:"bind_address"`
	StoragePath string   `json:"storage_path"`
	AuthMode    string   `json:"auth_mode"` // "api_key", "mtls"
	APIKeys     []string `json:"api_keys,omitempty"`
	TLSCA       string   `json:"tls_ca"`
	TLSCert     string   `json:"tls_cert"`
	TLSKey      string   `json:"tls_key"`
}

// Config holds the top-level configuration.
type Config struct {
	Server ServerConfig `json:"server"`
}

// LoadConfig loads configuration from a JSON file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
