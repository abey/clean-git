package clean_git_tests

import (
	"errors"
	"testing"
	"time"

	"github.com/abey/clean-git/internal/config"
	"github.com/abey/clean-git/internal/git"
	"github.com/abey/clean-git/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchService_GetCurrentBranch(t *testing.T) {
	tests := []struct {
		name          string
		currentBranch string
		setupMock     func(*mocks.SophisticatedGitClient)
		expectedName  string
		expectedError bool
	}{
		{
			name:          "successful current branch retrieval",
			currentBranch: "main",
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expectedName:  "main",
			expectedError: false,
		},
		{
			name:          "current branch is feature branch",
			currentBranch: "feature/test",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCurrentBranch("feature/test")
			},
			expectedName:  "feature/test",
			expectedError: false,
		},
		{
			name:          "git command fails",
			currentBranch: "main",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("GetCurrentBranchName", errors.New("git command failed"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			mockClient.SetCurrentBranch(tt.currentBranch)
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient, "origin")

			branch, err := service.GetCurrentBranch()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, branch)
			} else {
				require.NoError(t, err)
				require.NotNil(t, branch)
				assert.Equal(t, tt.expectedName, branch.Name)
				assert.True(t, branch.IsCurrent)
				assert.False(t, branch.IsRemote)
			}
		})
	}
}

func TestBranchService_GetMergedBranches(t *testing.T) {
	tests := []struct {
		name          string
		baseBranch    string
		setupMock     func(*mocks.SophisticatedGitClient)
		expectedCount int
		expectedNames []string
		expectedError bool
	}{
		{
			name:       "get merged branches successfully",
			baseBranch: "main",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.AddBranch(mocks.BranchData{
					Name:       "feature/merged-1",
					IsMerged:   true,
					AuthorName: "Alice",
					CommitSHA:  "sha1",
				})
				m.AddBranch(mocks.BranchData{
					Name:       "feature/merged-2",
					IsMerged:   true,
					AuthorName: "Bob",
					CommitSHA:  "sha2",
				})
			},
			expectedCount: 3, // feature/merged + feature/merged-1 + feature/merged-2
			expectedNames: []string{"feature/merged", "feature/merged-1", "feature/merged-2"},
			expectedError: false,
		},
		{
			name:       "no merged branches",
			baseBranch: "main",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				// Remove the default merged branch by overriding it as not merged
				m.AddBranch(mocks.BranchData{
					Name:     "feature/merged",
					IsMerged: false,
				})
			},
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:       "git command fails",
			baseBranch: "main",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("GetMergedBranchNames", errors.New("git failed"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient, "origin")

			branches, err := service.GetMergedBranches(tt.baseBranch)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, branches)
			} else {
				require.NoError(t, err)
				assert.Len(t, branches, tt.expectedCount)

				// Verify all returned branches are marked as merged
				for _, branch := range branches {
					assert.True(t, branch.IsMerged, "Branch %s should be marked as merged", branch.Name)
					assert.False(t, branch.IsRemote, "Merged branches should be local")
					assert.NotEmpty(t, branch.AuthorUserName)
					assert.NotEmpty(t, branch.LastCommitSHA)
				}

				// Check specific branch names if provided
				if len(tt.expectedNames) > 0 {
					actualNames := make([]string, len(branches))
					for i, branch := range branches {
						actualNames[i] = branch.Name
					}
					assert.ElementsMatch(t, tt.expectedNames, actualNames)
				}
			}
		})
	}
}

func TestBranchService_GetAllBranches(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.SophisticatedGitClient)
		expectedLocal  int
		expectedRemote int
		expectedError  bool
	}{
		{
			name: "get all branches successfully",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.AddBranch(mocks.BranchData{
					Name:        "feature/new",
					IsRemote:    false,
					AuthorName:  "New User",
					AuthorEmail: "new@example.com",
					CommitSHA:   "new123",
				})
				m.AddBranch(mocks.BranchData{
					Name:        "develop",
					IsRemote:    true,
					Remote:      "origin",
					AuthorName:  "Dev User",
					AuthorEmail: "dev@example.com",
					CommitSHA:   "dev123",
				})
			},
			expectedLocal:  4, // main, feature/test, feature/merged, feature/new
			expectedRemote: 2, // origin/main, origin/develop
			expectedError:  false,
		},
		{
			name: "git command fails",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("GetAllBranchNames", errors.New("git failed"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient, "origin")

			branches, err := service.GetAllBranches()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, branches)
			} else {
				require.NoError(t, err)

				localCount := 0
				remoteCount := 0

				for _, branch := range branches {
					if branch.IsRemote {
						remoteCount++
						assert.NotEmpty(t, branch.Remote, "Remote branch should have remote set")
					} else {
						localCount++
					}

					// All branches should have metadata
					assert.NotEmpty(t, branch.Name)
					assert.NotEmpty(t, branch.AuthorUserName)
					assert.NotEmpty(t, branch.LastCommitSHA)
				}

				assert.Equal(t, tt.expectedLocal, localCount, "Local branch count mismatch")
				assert.Equal(t, tt.expectedRemote, remoteCount, "Remote branch count mismatch")
			}
		})
	}
}

func TestBranchService_DeleteBranch(t *testing.T) {
	tests := []struct {
		name          string
		branch        *git.Branch
		setupMock     func(*mocks.SophisticatedGitClient)
		expectedError bool
		errorContains string
	}{
		{
			name: "delete local branch successfully",
			branch: &git.Branch{
				Name:     "feature/to-delete",
				IsRemote: false,
			},
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expectedError: false,
		},
		{
			name: "delete remote branch successfully",
			branch: &git.Branch{
				Name:     "feature/remote-delete",
				IsRemote: true,
				Remote:   "origin",
			},
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expectedError: false,
		},
		{
			name: "cannot delete current branch",
			branch: &git.Branch{
				Name:     "main",
				IsRemote: false,
			},
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("DeleteLocalBranch", errors.New("cannot delete current branch main"))
			},
			expectedError: true,
			errorContains: "cannot delete current branch",
		},
		{
			name: "remote branch without remote configured uses fallback",
			branch: &git.Branch{
				Name:     "feature/no-remote",
				IsRemote: true,
				Remote:   "", // No remote configured - should use service's remote name as fallback
			},
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expectedError: false, // Should succeed with fallback to service's remote name
		},
		{
			name: "git delete command fails",
			branch: &git.Branch{
				Name:     "feature/delete-fail",
				IsRemote: false,
			},
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("DeleteLocalBranch", errors.New("permission denied"))
			},
			expectedError: true,
			errorContains: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient, "origin")

			err := service.DeleteBranch(tt.branch)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBranchService_IsProtectedBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   *git.Branch
		patterns []string
		expected bool
	}{
		{
			name:     "main branch is protected",
			branch:   &git.Branch{Name: "main"},
			patterns: []string{"main", "master", "develop"},
			expected: true,
		},
		{
			name:     "master branch is protected",
			branch:   &git.Branch{Name: "master"},
			patterns: []string{"main", "master", "develop"},
			expected: true,
		},
		{
			name:     "release branch matches pattern",
			branch:   &git.Branch{Name: "release/v1.0"},
			patterns: []string{"main", "master", "release/.*"},
			expected: true,
		},
		{
			name:     "feature branch is not protected",
			branch:   &git.Branch{Name: "feature/test"},
			patterns: []string{"main", "master", "develop"},
			expected: false,
		},
		{
			name:     "empty patterns - nothing protected",
			branch:   &git.Branch{Name: "main"},
			patterns: []string{},
			expected: false,
		},
		{
			name:     "invalid regex pattern is ignored",
			branch:   &git.Branch{Name: "test"},
			patterns: []string{"[invalid", "test"},
			expected: true, // Should match "test" pattern
		},
		{
			name:     "complex regex pattern",
			branch:   &git.Branch{Name: "hotfix/urgent-fix-123"},
			patterns: []string{"^(main|master|develop)$", "^(release|hotfix)/.*"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			service := git.NewBranchServiceWithClient(mockClient, "origin")

			result := service.IsProtectedBranch(tt.branch, tt.patterns)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBranchService_GetBranchByName(t *testing.T) {
	tests := []struct {
		name           string
		branchName     string
		setupMock      func(*mocks.SophisticatedGitClient)
		expectedError  bool
		validateBranch func(*testing.T, *git.Branch)
	}{
		{
			name:          "get existing branch",
			branchName:    "feature/test",
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expectedError: false,
			validateBranch: func(t *testing.T, branch *git.Branch) {
				assert.Equal(t, "feature/test", branch.Name)
				assert.False(t, branch.IsCurrent)
				assert.False(t, branch.IsRemote)
				assert.Equal(t, "Jane Smith", branch.AuthorUserName)
				assert.Equal(t, "def456", branch.LastCommitSHA)
			},
		},
		{
			name:          "get current branch",
			branchName:    "main",
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expectedError: false,
			validateBranch: func(t *testing.T, branch *git.Branch) {
				assert.Equal(t, "main", branch.Name)
				assert.True(t, branch.IsCurrent)
				assert.False(t, branch.IsRemote)
			},
		},
		{
			name:       "branch with unpushed commits",
			branchName: "feature/unpushed",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.AddBranch(mocks.BranchData{
					Name:       "feature/unpushed",
					AuthorName: "Dev User",
					CommitSHA:  "unpushed123",
				})
				m.SetUnpushedCommits("feature/unpushed", 3)
			},
			expectedError: false,
			validateBranch: func(t *testing.T, branch *git.Branch) {
				assert.Equal(t, "feature/unpushed", branch.Name)
				assert.True(t, branch.HasUnpushedCommits)
			},
		},
		{
			name:       "nonexistent branch",
			branchName: "nonexistent",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("GetBranchCommitInfo", errors.New("branch nonexistent not found"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient, "origin")

			branch, err := service.GetBranchByName(tt.branchName)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, branch)
			} else {
				require.NoError(t, err)
				require.NotNil(t, branch)

				if tt.validateBranch != nil {
					tt.validateBranch(t, branch)
				}
			}
		})
	}
}

func TestBranchService_ConfigurableRemoteName(t *testing.T) {
	tests := []struct {
		name       string
		remoteName string
		setupMock  func(*mocks.SophisticatedGitClient, string)
		validate   func(*testing.T, []git.Branch, string)
	}{
		{
			name:       "uses custom remote name for branch creation",
			remoteName: "upstream",
			setupMock: func(m *mocks.SophisticatedGitClient, remote string) {
				m.AddBranch(mocks.BranchData{
					Name:       remote + "/feature/custom-remote", // Full remote branch name
					IsRemote:   true,
					Remote:     remote,
					AuthorName: "Remote User",
					CommitSHA:  "remote123",
				})
			},
			validate: func(t *testing.T, branches []git.Branch, remoteName string) {
				var remoteBranch *git.Branch
				for _, branch := range branches {
					if branch.IsRemote && branch.Name == "feature/custom-remote" {
						remoteBranch = &branch
						break
					}
				}
				require.NotNil(t, remoteBranch, "Remote branch should be found")
				assert.Equal(t, remoteName, remoteBranch.Remote)
			},
		},
		{
			name:       "uses origin as default when empty remote name",
			remoteName: "",
			setupMock: func(m *mocks.SophisticatedGitClient, remote string) {
				m.AddBranch(mocks.BranchData{
					Name:       "origin/feature/fallback", // Full remote branch name
					IsRemote:   true,
					Remote:     "origin",
					AuthorName: "Fallback User",
					CommitSHA:  "fallback123",
				})
			},
			validate: func(t *testing.T, branches []git.Branch, remoteName string) {
				var remoteBranch *git.Branch
				for _, branch := range branches {
					if branch.IsRemote && branch.Name == "feature/fallback" {
						remoteBranch = &branch
						break
					}
				}
				require.NotNil(t, remoteBranch, "Remote branch should be found")
				assert.Equal(t, "origin", remoteBranch.Remote) // Should be origin as fallback
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			mockClient.ClearBranches() // Clear default branches first
			tt.setupMock(mockClient, tt.remoteName)

			var service git.BranchService
			if tt.remoteName != "" {
				service = git.NewBranchServiceWithClient(mockClient, tt.remoteName)
			} else {
				service = git.NewBranchServiceWithClient(mockClient, "origin")
			}

			branches, err := service.GetAllBranches()
			require.NoError(t, err)

			tt.validate(t, branches, tt.remoteName)
		})
	}
}

func TestBranchService_RemoteBranchDeletion(t *testing.T) {
	tests := []struct {
		name          string
		branch        *git.Branch
		config        *config.Config
		expectedError bool
		validateCall  func(*testing.T, *mocks.SophisticatedGitClient)
	}{
		{
			name: "sets remote name from config when branch.Remote is empty",
			branch: &git.Branch{
				Name:     "feature/no-remote",
				IsRemote: true,
				Remote:   "", // Empty remote
			},
			config: &config.Config{
				RemoteName: "upstream",
			},
			expectedError: false,
			validateCall: func(t *testing.T, m *mocks.SophisticatedGitClient) {
				// Verify that DeleteRemoteBranch was called with "upstream"
				calls := m.GetDeleteRemoteBranchCalls()
				require.Len(t, calls, 1)
				assert.Equal(t, "upstream", calls[0].Remote)
				assert.Equal(t, "feature/no-remote", calls[0].BranchName)
			},
		},
		{
			name: "uses existing remote name when already set",
			branch: &git.Branch{
				Name:     "feature/has-remote",
				IsRemote: true,
				Remote:   "origin", // Already has remote
			},
			config: &config.Config{
				RemoteName: "upstream",
			},
			expectedError: false,
			validateCall: func(t *testing.T, m *mocks.SophisticatedGitClient) {
				// Should use existing "origin", not config "upstream"
				calls := m.GetDeleteRemoteBranchCalls()
				require.Len(t, calls, 1)
				assert.Equal(t, "origin", calls[0].Remote)
			},
		},
		{
			name: "remote branch without remote configured uses origin fallback",
			branch: &git.Branch{
				Name:     "feature/no-config",
				IsRemote: true,
				Remote:   "", // Empty remote
			},
			config:        &config.Config{RemoteName: ""}, // Empty remote name - should fallback to "origin"
			expectedError: false,
			validateCall: func(t *testing.T, m *mocks.SophisticatedGitClient) {
				// Should call delete with "origin" as fallback
				calls := m.GetDeleteRemoteBranchCalls()
				require.Len(t, calls, 1)
				assert.Equal(t, "origin", calls[0].Remote) // Should use "origin" as fallback
				assert.Equal(t, "feature/no-config", calls[0].BranchName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()

			var service git.BranchService
			if tt.config != nil {
				service = git.NewBranchServiceWithClient(mockClient, tt.config.RemoteName)
			} else {
				service = git.NewBranchServiceWithClient(mockClient, "origin")
			}

			err := service.DeleteBranch(tt.branch)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.validateCall(t, mockClient)
			}
		})
	}
}

func TestBranchService_EdgeCases(t *testing.T) {
	t.Run("remote branch name cleaning", func(t *testing.T) {
		mockClient := mocks.NewMockedGitClient()
		mockClient.AddBranch(mocks.BranchData{
			Name:       "feature/remote-test",
			IsRemote:   true,
			Remote:     "origin",
			AuthorName: "Remote User",
			CommitSHA:  "remote123",
		})

		service := git.NewBranchServiceWithClient(mockClient, "origin")

		branches, err := service.GetAllBranches()

		require.NoError(t, err)

		// Find the remote branch
		var remoteBranch *git.Branch
		for _, branch := range branches {
			if branch.IsRemote && branch.Name == "feature/remote-test" {
				remoteBranch = &branch
				break
			}
		}

		require.NotNil(t, remoteBranch, "Remote branch should be found")
		assert.Equal(t, "feature/remote-test", remoteBranch.Name)
		assert.Equal(t, "origin", remoteBranch.Remote)
		assert.True(t, remoteBranch.IsRemote)
		assert.False(t, remoteBranch.HasUnpushedCommits) // Remote branches don't have unpushed commits
	})

	t.Run("branch with special characters", func(t *testing.T) {
		mockClient := mocks.NewMockedGitClient()
		specialBranchName := "feature/fix-issue-#123_with-special.chars"
		mockClient.AddBranch(mocks.BranchData{
			Name:       specialBranchName,
			AuthorName: "Special User",
			CommitSHA:  "special123",
		})

		service := git.NewBranchServiceWithClient(mockClient, "origin")

		branch, err := service.GetBranchByName(specialBranchName)

		require.NoError(t, err)
		assert.Equal(t, specialBranchName, branch.Name)
	})

	t.Run("very old branch", func(t *testing.T) {
		mockClient := mocks.NewMockedGitClient()
		oldDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		mockClient.AddBranch(mocks.BranchData{
			Name:       "feature/very-old",
			CommitDate: oldDate,
			AuthorName: "Old User",
			CommitSHA:  "old123",
		})

		service := git.NewBranchServiceWithClient(mockClient, "origin")

		branch, err := service.GetBranchByName("feature/very-old")

		require.NoError(t, err)
		assert.True(t, branch.LastCommitAt.Equal(oldDate), "Expected %v, got %v", oldDate, branch.LastCommitAt)
		assert.True(t, time.Since(branch.LastCommitAt) > 365*24*time.Hour)
	})
}
