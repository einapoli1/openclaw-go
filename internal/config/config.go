package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the top-level configuration
type Config struct {
	Gateway GatewayConfig `json:"gateway"`
	Agents  AgentsConfig  `json:"agents"`
}

// GatewayConfig configures the gateway server
type GatewayConfig struct {
	Mode string `json:"mode"` // "local" or "remote"
	Port int    `json:"port"`
	Bind string `json:"bind"`
}

// AgentsConfig configures agents
type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list"`
}

// AgentDefaults are default settings for all agents
type AgentDefaults struct {
	Workspace string `json:"workspace"`
	Model     string `json:"model"`
}

// AgentConfig defines a single agent
type AgentConfig struct {
	ID        string        `json:"id"`
	Identity  AgentIdentity `json:"identity"`
	Workspace string        `json:"workspace"`
	Model     string        `json:"model"`
}

// AgentIdentity defines an agent's name and appearance
type AgentIdentity struct {
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "openclaw.json")
}

// Load reads and parses the config file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Apply defaults
	if cfg.Gateway.Port == 0 {
		cfg.Gateway.Port = 18789
	}
	if cfg.Gateway.Bind == "" {
		cfg.Gateway.Bind = "127.0.0.1"
	}
	if cfg.Agents.Defaults.Model == "" {
		cfg.Agents.Defaults.Model = "claude-sonnet-4-20250514"
	}

	// Expand workspace paths
	cfg.Agents.Defaults.Workspace = expandHome(cfg.Agents.Defaults.Workspace)
	for i := range cfg.Agents.List {
		if cfg.Agents.List[i].Workspace == "" {
			cfg.Agents.List[i].Workspace = cfg.Agents.Defaults.Workspace
		} else {
			cfg.Agents.List[i].Workspace = expandHome(cfg.Agents.List[i].Workspace)
		}
		if cfg.Agents.List[i].Model == "" {
			cfg.Agents.List[i].Model = cfg.Agents.Defaults.Model
		}
	}

	return &cfg, nil
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
