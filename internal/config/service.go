package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDir        = ".clean-git/configs"
	GlobalConfigFile = "global.yaml"
)

type repoConfigService struct {
	repoRoot   string
	configPath string
	config     *Config
	onboarding bool
}

func (s *repoConfigService) Config() *Config {
	if s.config == nil {
		s.config = DefaultConfig()
	}
	return s.config
}

func (s *repoConfigService) Save() error {
	if s.config == nil {
		s.config = DefaultConfig()
	}

	if err := ensureConfigDirExists(s.configPath); err != nil {
		return err
	}

	data, err := yaml.Marshal(s.config)
	if err != nil {
		return err
	}

	return os.WriteFile(s.configPath, data, 0644)
}

func (s *repoConfigService) IsOnboarded() bool {
	_, err := os.Stat(s.configPath)
	return err == nil
}

func NewService(repoRoot string) (Service, error) {
	configPath, err := getGlobalConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get global config path: %w", err)
	}

	service := &repoConfigService{
		repoRoot:   repoRoot,
		configPath: configPath,
	}

	if err := ensureConfigDirExists(configPath); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := service.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return service, nil
}

func NewOnboardingService(repoRoot string) (Service, error) {
	configPath, err := getGlobalConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine config path: %w", err)
	}

	if err := ensureConfigDirExists(configPath); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &repoConfigService{
		repoRoot:   repoRoot,
		configPath: configPath,
		onboarding: true,
	}, nil
}

func (s *repoConfigService) load() error {
	if s.onboarding {
		s.config = DefaultConfig()
		return nil
	}

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.config = DefaultConfig()
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	s.config = &config
	return nil
}

func (s *repoConfigService) Update(cfg *Config) error {
	s.config = cfg
	return s.Save()
}

func (s *repoConfigService) ConfigPath() string {
	return s.configPath
}
