package clean_git_tests

import (
	"errors"
	"testing"
	"time"

	"clean-git/internal/git"
	"clean-git/tests/mocks"

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
			mockClient := mocks.NewSophisticatedGitClient()
			mockClient.SetCurrentBranch(tt.currentBranch)
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient)

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
			mockClient := mocks.NewSophisticatedGitClient()
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient)

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
			mockClient := mocks.NewSophisticatedGitClient()
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient)

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
			name: "remote branch without remote configured",
			branch: &git.Branch{
				Name:     "feature/no-remote",
				IsRemote: true,
				Remote:   "", // No remote configured
			},
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expectedError: true,
			errorContains: "no remote configured",
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
			mockClient := mocks.NewSophisticatedGitClient()
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient)

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
			mockClient := mocks.NewSophisticatedGitClient()
			service := git.NewBranchServiceWithClient(mockClient)

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
			mockClient := mocks.NewSophisticatedGitClient()
			tt.setupMock(mockClient)

			service := git.NewBranchServiceWithClient(mockClient)

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

func TestBranchService_EdgeCases(t *testing.T) {
	t.Run("remote branch name cleaning", func(t *testing.T) {
		mockClient := mocks.NewSophisticatedGitClient()
		mockClient.AddBranch(mocks.BranchData{
			Name:       "feature/remote-test",
			IsRemote:   true,
			Remote:     "origin",
			AuthorName: "Remote User",
			CommitSHA:  "remote123",
		})

		service := git.NewBranchServiceWithClient(mockClient)
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
		mockClient := mocks.NewSophisticatedGitClient()
		specialBranchName := "feature/fix-issue-#123_with-special.chars"
		mockClient.AddBranch(mocks.BranchData{
			Name:       specialBranchName,
			AuthorName: "Special User",
			CommitSHA:  "special123",
		})

		service := git.NewBranchServiceWithClient(mockClient)
		branch, err := service.GetBranchByName(specialBranchName)

		require.NoError(t, err)
		assert.Equal(t, specialBranchName, branch.Name)
	})

	t.Run("very old branch", func(t *testing.T) {
		mockClient := mocks.NewSophisticatedGitClient()
		oldDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		mockClient.AddBranch(mocks.BranchData{
			Name:       "feature/very-old",
			CommitDate: oldDate,
			AuthorName: "Old User",
			CommitSHA:  "old123",
		})

		service := git.NewBranchServiceWithClient(mockClient)
		branch, err := service.GetBranchByName("feature/very-old")

		require.NoError(t, err)
		assert.True(t, branch.LastCommitAt.Equal(oldDate), "Expected %v, got %v", oldDate, branch.LastCommitAt)
		assert.True(t, time.Since(branch.LastCommitAt) > 365*24*time.Hour)
	})
}
