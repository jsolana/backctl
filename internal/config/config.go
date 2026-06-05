package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BaseURL   string `yaml:"base_url"`
	TokenFile string `yaml:"token_file"`
	Namespace string `yaml:"namespace"`
	Output    string `yaml:"output"`
	Timeout   string `yaml:"timeout"`
	NoAuth    bool   `yaml:"no_auth"`
	Verbose   bool   `yaml:"verbose"`
}

const defaultConfigDir = "backctl"
const defaultConfigFile = "config.yaml"

// Load reads the config file from the given path.
// If path is empty, it defaults to ~/.config/backctl/config.yaml (XDG_CONFIG_HOME aware).
// Returns a zero Config (no error) if the file does not exist.
func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	cfg.TokenFile = expandTilde(cfg.TokenFile)
	return cfg, nil
}

// DefaultPath returns the default config file path, respecting XDG_CONFIG_HOME.
func DefaultPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, defaultConfigDir, defaultConfigFile)
}

func expandTilde(path string) string {
	if len(path) < 2 || path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
