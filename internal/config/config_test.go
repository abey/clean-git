package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Helpers

func setupHome(t *testing.T, tempDir string) (string, func()) {
	t.Helper()
	homeDir := filepath.Join(tempDir, "home")
	originalHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", homeDir))
	return homeDir, func() { _ = os.Setenv("HOME", originalHome) }
}

func newServiceFor(t *testing.T, tempDir string) Service {
	t.Helper()
	svc, err := NewService(tempDir)
	require.NoError(t, err)
	return svc
}

func newOnboardingServiceFor(t *testing.T, tempDir string) Service {
	t.Helper()
	svc, err := NewOnboardingService(tempDir)
	require.NoError(t, err)
	return svc
}

// Tests

func TestConfigService(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		tempDir := t.TempDir()
		_, restore := setupHome(t, tempDir)
		defer restore()
		service := newServiceFor(t, tempDir)

		cfg := service.Config()
		assert.Equal(t, []string{"main", "master", "develop"}, cfg.BaseBranches)
		assert.Equal(t, []string{".*"}, cfg.IncludeRegex)
		assert.Equal(t, 720*time.Hour*24, cfg.MaxAge)
		assert.Equal(t, []string{"release/*", "hotfix/*"}, cfg.ProtectedRegex)
		assert.Equal(t, []string{".*"}, cfg.IncludeRegex)
		assert.Equal(t, "origin", cfg.RemoteName)
	})

	t.Run("SaveAndLoadConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		_, restore := setupHome(t, tempDir)
		defer restore()
		service := newServiceFor(t, tempDir)

		cfg := service.Config()
		cfg.BaseBranches = []string{"main", "develop"}
		cfg.MaxAge = 168 * time.Hour // 7 days
		cfg.ProtectedRegex = []string{"feature/*"}
		cfg.IncludeRegex = []string{"feature/.*"}
		cfg.RemoteName = "upstream"

		err := service.Save()
		require.NoError(t, err)

		service, err = NewService(tempDir)
		require.NoError(t, err)

		cfg = service.Config()
		assert.Equal(t, []string{"main", "develop"}, cfg.BaseBranches)
		assert.Equal(t, 168*time.Hour, cfg.MaxAge)
		assert.Equal(t, []string{"feature/*"}, cfg.ProtectedRegex)
		assert.Equal(t, []string{"feature/.*"}, cfg.IncludeRegex)
		assert.Equal(t, "upstream", cfg.RemoteName)
	})

	t.Run("IsOnboarded", func(t *testing.T) {
		tempDir := t.TempDir()
		_, restore := setupHome(t, tempDir)
		defer restore()
		service := newServiceFor(t, tempDir)

		assert.False(t, service.IsOnboarded())

		err := service.Save()
		require.NoError(t, err)
		assert.True(t, service.IsOnboarded())
	})
}

func TestFindGitRepoRoot(t *testing.T) {
	t.Run("FindFromNestedDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		repoRoot := filepath.Join(tempDir, "myrepo")
		err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755)
		require.NoError(t, err)

		nestedDir := filepath.Join(repoRoot, "nested", "directory")
		err = os.MkdirAll(nestedDir, 0755)
		require.NoError(t, err)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(nestedDir)
		require.NoError(t, err)

		foundRepoRoot, err := findGitRepoRoot()
		require.NoError(t, err)
		expectedPath := repoRoot // macOS hack
		if strings.HasPrefix(foundRepoRoot, "/private") {
			expectedPath = "/private" + expectedPath
		}
		assert.Equal(t, expectedPath, foundRepoRoot)
	})

	t.Run("NoGitRepo", func(t *testing.T) {
		tempDir := t.TempDir()
		err := os.Chdir(tempDir)
		require.NoError(t, err)

		_, err = findGitRepoRoot()
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestNewService_ErrorCases(t *testing.T) {
	t.Run("InvalidConfigDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir, restore := setupHome(t, tempDir)
		defer restore()

		err := os.MkdirAll(homeDir, 0755) // 0755: read + write + execute
		require.NoError(t, err)

		configDirPath := filepath.Join(homeDir, ".clean-git")
		err = os.WriteFile(configDirPath, []byte("blocking file"), 0644) // 0644: read + write + execute
		require.NoError(t, err)

		_, err = NewService(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create config directory")
	})

	t.Run("CorruptedConfigFile", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir, restore := setupHome(t, tempDir)
		defer restore()
		configDir := filepath.Join(homeDir, ".clean-git")
		err := os.MkdirAll(configDir, 0755) // 0755: read + write + execute
		require.NoError(t, err)

		configPath := filepath.Join(configDir, "config.yaml")
		err = os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644) // 0644: read + write + execute
		require.NoError(t, err)

		_, err = NewService(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})
}

func TestNewOnboardingService(t *testing.T) {
	t.Run("SuccessfulCreation", func(t *testing.T) {
		tempDir := t.TempDir()
		_, restore := setupHome(t, tempDir)
		defer restore()
		service := newOnboardingServiceFor(t, tempDir)

		cfg := service.Config()
		assert.Equal(t, []string{"main", "master", "develop"}, cfg.BaseBranches)

		assert.False(t, service.IsOnboarded())

		err := service.Save()
		require.NoError(t, err)
		assert.True(t, service.IsOnboarded())
	})

	t.Run("DirectoryCreationError", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir, restore := setupHome(t, tempDir)
		err := os.MkdirAll(homeDir, 0755) // 0755: read + write + execute
		require.NoError(t, err)

		configDirPath := filepath.Join(homeDir, ".clean-git")
		err = os.WriteFile(configDirPath, []byte("blocking file"), 0644) // 0644: read + write + execute
		require.NoError(t, err)

		defer restore()

		_, err = NewOnboardingService(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create config directory")
	})
}

func TestConfigService_Update(t *testing.T) {
	tempDir := t.TempDir()
	_, restore := setupHome(t, tempDir)
	defer restore()
	service := newServiceFor(t, tempDir)

	customConfig := &Config{
		BaseBranches:   []string{"main", "staging"},
		MaxAge:         48 * time.Hour,
		ProtectedRegex: []string{"prod/*"},
		RemoteName:     "upstream",
	}

	err := service.Update(customConfig)
	require.NoError(t, err)

	cfg := service.Config()
	assert.Equal(t, []string{"main", "staging"}, cfg.BaseBranches)
	assert.Equal(t, 48*time.Hour, cfg.MaxAge)
	assert.Equal(t, []string{"prod/*"}, cfg.ProtectedRegex)
	assert.Equal(t, "upstream", cfg.RemoteName)

	newService, err := NewService(tempDir)
	require.NoError(t, err)
	newCfg := newService.Config()
	assert.Equal(t, customConfig.BaseBranches, newCfg.BaseBranches)
	assert.Equal(t, customConfig.MaxAge, newCfg.MaxAge)
}

func TestConfigService_SaveErrors(t *testing.T) {
	t.Run("ReadOnlyConfigFile", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir, restore := setupHome(t, tempDir)
		configDir := filepath.Join(homeDir, ".clean-git")
		err := os.MkdirAll(configDir, 0755) // 0755: read + write + execute
		require.NoError(t, err)

		configPath := filepath.Join(configDir, "config.yaml")
		validYAML := `baseBranches: [main]
maxAge: 24h`
		err = os.WriteFile(configPath, []byte(validYAML), 0644)
		require.NoError(t, err)

		err = os.Chmod(configPath, 0444)
		require.NoError(t, err)

		defer restore()
		service := newServiceFor(t, tempDir)

		err = service.Save()
		assert.Error(t, err)
	})
}

func TestConfigService_LoadErrors(t *testing.T) {
	t.Run("ReadPermissionDenied", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir, restore := setupHome(t, tempDir)
		configDir := filepath.Join(homeDir, ".clean-git")
		err := os.MkdirAll(configDir, 0755) // 0755: read + write + execute
		require.NoError(t, err)

		configPath := filepath.Join(configDir, "config.yaml")
		err = os.WriteFile(configPath, []byte("baseBranches: [main]"), 0000) // 0000: no permissions
		require.NoError(t, err)

		defer restore()

		_, err = NewService(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
}

func TestConfigEdgeCases(t *testing.T) {
	t.Run("EmptyConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir, restore := setupHome(t, tempDir)
		configDir := filepath.Join(homeDir, ".clean-git")
		err := os.MkdirAll(configDir, 0755) // 0755: read + write + execute
		require.NoError(t, err)

		configPath := filepath.Join(configDir, "config.yaml")
		err = os.WriteFile(configPath, []byte(""), 0644) // 0644: read + write + execute
		require.NoError(t, err)

		defer restore()
		service := newServiceFor(t, tempDir)

		cfg := service.Config()
		assert.NotNil(t, cfg)
	})

	t.Run("PartialConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir, restore := setupHome(t, tempDir)
		configDir := filepath.Join(homeDir, ".clean-git")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		configPath := filepath.Join(configDir, "config.yaml")
		partialYAML := `baseBranches: [main]
maxAge: 168h`
		err = os.WriteFile(configPath, []byte(partialYAML), 0644)
		require.NoError(t, err)

		defer restore()
		service := newServiceFor(t, tempDir)

		cfg := service.Config()
		assert.Equal(t, []string{"main"}, cfg.BaseBranches)
		assert.Equal(t, 168*time.Hour, cfg.MaxAge)
		assert.Empty(t, cfg.ProtectedRegex)
		assert.Empty(t, cfg.IncludeRegex)
		assert.Empty(t, cfg.RemoteName)
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("GetGlobalConfigPath", func(t *testing.T) {
		tempDir := t.TempDir()
		homeDir, restore := setupHome(t, tempDir)
		defer restore()

		configPath, err := getGlobalConfigPath()
		require.NoError(t, err)

		expectedPath := filepath.Join(homeDir, ".clean-git", "config.yaml")
		assert.Equal(t, expectedPath, configPath)
	})

	t.Run("EnsureConfigDirExists", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "nested", "config", "config.yaml")

		configDir := filepath.Dir(configPath)
		_, err := os.Stat(configDir)
		assert.True(t, os.IsNotExist(err))
		err = ensureConfigDirExists(configPath)
		require.NoError(t, err)

		stat, err := os.Stat(configDir)
		require.NoError(t, err)
		assert.True(t, stat.IsDir())
	})

	t.Run("EnsureConfigDirExists_PermissionError", func(t *testing.T) {
		tempDir := t.TempDir()

		restrictedDir := filepath.Join(tempDir, "restricted")
		err := os.MkdirAll(restrictedDir, 0555) // 0555: read + execute only
		require.NoError(t, err)

		configPath := filepath.Join(restrictedDir, "subdir", "config.yaml")

		err = ensureConfigDirExists(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create config directory")
	})
}
