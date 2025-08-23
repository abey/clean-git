package config

import (
	"os"
	"path/filepath"
)

func FindGitRepoRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Recursively search for .git directory up from current directory
	for {
		gitDir := filepath.Join(currentDir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)

		if parentDir == currentDir {
			return "", os.ErrNotExist
		}

		currentDir = parentDir
	}
}

func GetDefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ConfigDir, GlobalConfigFile), nil
}
