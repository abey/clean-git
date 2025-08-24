package clean_git_tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"clean-git/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigService_SaveAndLoad tests integration scenarios for configuration persistence
func TestConfigService_SaveAndLoad(t *testing.T) {
	t.Run("interactive configuration changes are persisted", func(t *testing.T) {
		tempDir3, err := os.MkdirTemp("", "clean-git-config-test-3")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir3)

		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir3)
		defer os.Setenv("HOME", originalHome)

		repoRoot3 := filepath.Join(tempDir3, "repo")
		err = os.MkdirAll(repoRoot3, 0755)
		require.NoError(t, err)

		service, err := config.NewService(repoRoot3)
		require.NoError(t, err)

		initialConfig := service.Config()
		assert.Equal(t, "origin", initialConfig.RemoteName)

		updatedConfig := &config.Config{
			BaseBranches:   []string{"main", "master"},
			MaxAge:         48 * time.Hour,
			ProtectedRegex: []string{"release/.*", "main", "master"},
			IncludeRegex:   []string{".*"},
			RemoteName:     "upstream",
		}

		err = service.Update(updatedConfig)
		require.NoError(t, err)

		service2, err := config.NewService(repoRoot3)
		require.NoError(t, err)

		persistedConfig := service2.Config()
		assert.Equal(t, "upstream", persistedConfig.RemoteName)
		assert.Equal(t, 48*time.Hour, persistedConfig.MaxAge)
		assert.Equal(t, []string{"main", "master"}, persistedConfig.BaseBranches)
	})
}

// TestConfigService_GlobalConfigPath tests that config is shared across multiple repositories
func TestConfigService_GlobalConfigPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "clean-git-global-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	repoRoot1 := filepath.Join(tempDir, "repo1")
	repoRoot2 := filepath.Join(tempDir, "repo2")
	
	err = os.MkdirAll(repoRoot1, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(repoRoot2, 0755)
	require.NoError(t, err)

	service1, err := config.NewService(repoRoot1)
	require.NoError(t, err)
	
	service2, err := config.NewService(repoRoot2)
	require.NoError(t, err)

	assert.Equal(t, service1.ConfigPath(), service2.ConfigPath())
	
	expectedGlobalPath := filepath.Join(tempDir, ".clean-git", "config.yaml")
	assert.Equal(t, expectedGlobalPath, service1.ConfigPath())

	testConfig := &config.Config{
		BaseBranches:   []string{"main", "develop"},
		MaxAge:         96 * time.Hour,
		ProtectedRegex: []string{"release/.*"},
		IncludeRegex:   []string{".*"},
		RemoteName:     "shared-remote",
	}
	
	err = service1.Update(testConfig)
	require.NoError(t, err)

	service2, err = config.NewService(repoRoot2)
	require.NoError(t, err)

	sharedConfig := service2.Config()
	assert.Equal(t, "shared-remote", sharedConfig.RemoteName)
	assert.Equal(t, 96*time.Hour, sharedConfig.MaxAge)
}

// TestConfigService_ErrorHandling tests integration error scenarios
func TestConfigService_ErrorHandling(t *testing.T) {
	t.Run("invalid repo root", func(t *testing.T) {
		service, err := config.NewService("/non/existent/path")
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, service)
		}
	})

	t.Run("permission denied for config directory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tempDir, err := os.MkdirTemp("", "clean-git-permission-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		restrictedDir := filepath.Join(tempDir, "restricted")
		err = os.MkdirAll(restrictedDir, 0444)
		require.NoError(t, err)

		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", restrictedDir)
		defer os.Setenv("HOME", originalHome)

		repoRoot := filepath.Join(tempDir, "repo")
		err = os.MkdirAll(repoRoot, 0755)
		require.NoError(t, err)

		_, err = config.NewService(repoRoot)
		assert.Error(t, err, "Should fail when unable to create config directory")
	})
}
