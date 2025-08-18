package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	BaseBranches   []string      `yaml:"baseBranches,omitempty"`
	MaxAge         time.Duration `yaml:"maxAge,omitempty"`
	ProtectedRegex []string      `yaml:"protectedRegex,omitempty"`
	IncludeRegex   []string      `yaml:"includeRegex,omitempty"`
	RemoteName     string        `yaml:"remoteName,omitempty"`
}

type Service interface {
	Config() *Config
	Save() error
	Update(cfg *Config) error
	IsOnboarded() bool
}

func DefaultConfig() *Config {
	return &Config{
		BaseBranches:   []string{"main", "master", "develop"},
		MaxAge:         720 * time.Hour * 24, // 30 days
		ProtectedRegex: []string{"release/*", "hotfix/*"},
		IncludeRegex:   []string{".*"},
		RemoteName:     "origin",
	}
}

func getGlobalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".clean-git", "config.yaml"), nil
}

func ensureConfigDirExists(configPath string) error {
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	return nil
}
