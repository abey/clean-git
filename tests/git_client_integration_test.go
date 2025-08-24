package clean_git_tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/abey/clean-git/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitClient_GetCurrentBranchName(t *testing.T) {
	tests := []struct {
		name          string
		currentBranch string
		setupMock     func(*mocks.SophisticatedGitClient)
		expected      string
		expectedError bool
	}{
		{
			name:          "main branch",
			currentBranch: "main",
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expected:      "main",
			expectedError: false,
		},
		{
			name:          "feature branch",
			currentBranch: "feature/awesome-feature",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCurrentBranch("feature/awesome-feature")
			},
			expected:      "feature/awesome-feature",
			expectedError: false,
		},
		{
			name:          "detached HEAD state",
			currentBranch: "HEAD",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCurrentBranch("HEAD")
			},
			expected:      "HEAD",
			expectedError: false,
		},
		{
			name: "git command fails",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("GetCurrentBranchName", errors.New("not a git repository"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			if tt.currentBranch != "" {
				mockClient.SetCurrentBranch(tt.currentBranch)
			}
			tt.setupMock(mockClient)

			result, err := mockClient.GetCurrentBranchName()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGitClient_GetMergedBranchNames(t *testing.T) {
	tests := []struct {
		name          string
		baseBranch    string
		setupMock     func(*mocks.SophisticatedGitClient)
		expectedNames []string
		expectedError bool
	}{
		{
			name:       "get merged branches from main",
			baseBranch: "main",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.AddBranch(mocks.BranchData{Name: "feature/merged-1", IsMerged: true})
				m.AddBranch(mocks.BranchData{Name: "feature/merged-2", IsMerged: true})
				m.AddBranch(mocks.BranchData{Name: "feature/not-merged", IsMerged: false})
			},
			expectedNames: []string{"feature/merged", "feature/merged-1", "feature/merged-2"},
			expectedError: false,
		},
		{
			name:       "no merged branches",
			baseBranch: "main",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				// Override default merged branch as not merged
				m.AddBranch(mocks.BranchData{Name: "feature/merged", IsMerged: false})
			},
			expectedNames: []string{},
			expectedError: false,
		},
		{
			name:       "base branch excluded from results",
			baseBranch: "develop",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.AddBranch(mocks.BranchData{Name: "develop", IsMerged: true})
				m.AddBranch(mocks.BranchData{Name: "feature/merged-into-develop", IsMerged: true})
			},
			expectedNames: []string{"feature/merged", "feature/merged-into-develop"},
			expectedError: false,
		},
		{
			name:       "git command fails",
			baseBranch: "main",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("GetMergedBranchNames", errors.New("git merge-base failed"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupMock(mockClient)

			result, err := mockClient.GetMergedBranchNames(tt.baseBranch)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedNames, result)
			}
		})
	}
}

func TestGitClient_GetAllBranchNames(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.SophisticatedGitClient)
		expectedLocal  []string
		expectedRemote []string
		expectedError  bool
	}{
		{
			name: "get all branches",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.AddBranch(mocks.BranchData{Name: "develop", IsRemote: false})
				m.AddBranch(mocks.BranchData{Name: "feature/remote", IsRemote: true})
			},
			expectedLocal:  []string{"main", "feature/test", "feature/merged", "develop"},
			expectedRemote: []string{"origin/main", "origin/feature/remote"},
			expectedError:  false,
		},
		{
			name: "only local branches",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				// Clear all branches and add only local ones
				m.ClearBranches()
				m.AddBranch(mocks.BranchData{Name: "main", IsRemote: false, AuthorName: "User", CommitSHA: "abc"})
				m.AddBranch(mocks.BranchData{Name: "feature/test", IsRemote: false, AuthorName: "User", CommitSHA: "def"})
				m.AddBranch(mocks.BranchData{Name: "feature/merged", IsRemote: false, AuthorName: "User", CommitSHA: "ghi"})
			},
			expectedLocal:  []string{"main", "feature/test", "feature/merged"},
			expectedRemote: []string{},
			expectedError:  false,
		},
		{
			name: "git command fails",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("GetAllBranchNames", errors.New("git branch failed"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupMock(mockClient)

			result, err := mockClient.GetAllBranchNames()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)

				var localBranches, remoteBranches []string
				for _, branch := range result {
					if strings.HasPrefix(branch, "origin/") {
						remoteBranches = append(remoteBranches, branch)
					} else {
						localBranches = append(localBranches, branch)
					}
				}

				assert.ElementsMatch(t, tt.expectedLocal, localBranches)
				assert.ElementsMatch(t, tt.expectedRemote, remoteBranches)
			}
		})
	}
}

func TestGitClient_GetBranchCommitInfo(t *testing.T) {
	tests := []struct {
		name          string
		branchName    string
		setupMock     func(*mocks.SophisticatedGitClient)
		validateInfo  func(*testing.T, string)
		expectedError bool
	}{
		{
			name:       "get commit info for existing branch",
			branchName: "feature/test",
			setupMock:  func(m *mocks.SophisticatedGitClient) {},
			validateInfo: func(t *testing.T, info string) {
				parts := strings.Split(info, "|")
				require.Len(t, parts, 4)
				assert.Contains(t, parts[0], "2") // Date contains year
				assert.Equal(t, "Jane Smith", parts[1])
				assert.Equal(t, "jane@example.com", parts[2])
				assert.Equal(t, "def456", parts[3])
			},
			expectedError: false,
		},
		{
			name:       "nonexistent branch",
			branchName: "nonexistent",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("GetBranchCommitInfo", errors.New("branch nonexistent not found"))
			},
			expectedError: true,
		},
		{
			name:       "branch with special characters in commit info",
			branchName: "feature/special",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.AddBranch(mocks.BranchData{
					Name:        "feature/special",
					AuthorName:  "User With Spaces",
					AuthorEmail: "user+tag@domain.co.uk",
					CommitSHA:   "abc123def456",
				})
			},
			validateInfo: func(t *testing.T, info string) {
				parts := strings.Split(info, "|")
				require.Len(t, parts, 4)
				assert.Equal(t, "User With Spaces", parts[1])
				assert.Equal(t, "user+tag@domain.co.uk", parts[2])
				assert.Equal(t, "abc123def456", parts[3])
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupMock(mockClient)

			result, err := mockClient.GetBranchCommitInfo(tt.branchName)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)

				if tt.validateInfo != nil {
					tt.validateInfo(t, result)
				}
			}
		})
	}
}

func TestGitClient_HasUnpushedCommits(t *testing.T) {
	tests := []struct {
		name          string
		branchName    string
		setupMock     func(*mocks.SophisticatedGitClient)
		expected      bool
		expectedError bool
	}{
		{
			name:       "branch with unpushed commits",
			branchName: "feature/unpushed",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetUnpushedCommits("feature/unpushed", 3)
			},
			expected:      true,
			expectedError: false,
		},
		{
			name:       "branch without unpushed commits",
			branchName: "feature/clean",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetUnpushedCommits("feature/clean", 0)
			},
			expected:      false,
			expectedError: false,
		},
		{
			name:          "branch without upstream",
			branchName:    "feature/no-upstream",
			setupMock:     func(m *mocks.SophisticatedGitClient) {},
			expected:      false, // No upstream means no unpushed commits tracked
			expectedError: false,
		},
		{
			name:       "git command fails",
			branchName: "feature/error",
			setupMock: func(m *mocks.SophisticatedGitClient) {
				m.SetCommandFailure("HasUnpushedCommits", errors.New("no upstream branch"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupMock(mockClient)

			result, err := mockClient.HasUnpushedCommits(tt.branchName)

			if tt.expectedError {
				assert.Error(t, err)
				assert.False(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGitClient_DeleteBranches(t *testing.T) {
	t.Run("delete local branch", func(t *testing.T) {
		tests := []struct {
			name          string
			branchName    string
			setupMock     func(*mocks.SophisticatedGitClient)
			expectedError bool
			errorContains string
		}{
			{
				name:          "delete regular branch",
				branchName:    "feature/to-delete",
				setupMock:     func(m *mocks.SophisticatedGitClient) {},
				expectedError: false,
			},
			{
				name:       "cannot delete current branch",
				branchName: "main",
				setupMock: func(m *mocks.SophisticatedGitClient) {
					m.SetCommandFailure("DeleteLocalBranch", errors.New("cannot delete current branch main"))
				},
				expectedError: true,
				errorContains: "cannot delete current branch",
			},
			{
				name:       "branch has unmerged changes",
				branchName: "feature/unmerged",
				setupMock: func(m *mocks.SophisticatedGitClient) {
					m.SetCommandFailure("DeleteLocalBranch", errors.New("branch not fully merged"))
				},
				expectedError: true,
				errorContains: "not fully merged",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockClient := mocks.NewMockedGitClient()
				tt.setupMock(mockClient)

				err := mockClient.DeleteLocalBranch(tt.branchName)

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
	})

	t.Run("delete remote branch", func(t *testing.T) {
		tests := []struct {
			name          string
			remote        string
			branchName    string
			setupMock     func(*mocks.SophisticatedGitClient)
			expectedError bool
			errorContains string
		}{
			{
				name:          "delete remote branch successfully",
				remote:        "origin",
				branchName:    "feature/remote-delete",
				setupMock:     func(m *mocks.SophisticatedGitClient) {},
				expectedError: false,
			},
			{
				name:       "remote push fails",
				remote:     "origin",
				branchName: "feature/push-fail",
				setupMock: func(m *mocks.SophisticatedGitClient) {
					m.SetCommandFailure("DeleteRemoteBranch", errors.New("permission denied"))
				},
				expectedError: true,
				errorContains: "permission denied",
			},
			{
				name:       "network error",
				remote:     "origin",
				branchName: "feature/network-fail",
				setupMock: func(m *mocks.SophisticatedGitClient) {
					m.SetCommandFailure("DeleteRemoteBranch", errors.New("connection timeout"))
				},
				expectedError: true,
				errorContains: "connection timeout",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockClient := mocks.NewMockedGitClient()
				tt.setupMock(mockClient)

				err := mockClient.DeleteRemoteBranch(tt.remote, tt.branchName)

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
	})
}

func TestGitClient_RawCommandExecution(t *testing.T) {
	t.Run("run git commands", func(t *testing.T) {
		mockClient := mocks.NewMockedGitClient()

		// Test basic command execution
		output, err := mockClient.Run("branch", "--show-current")
		require.NoError(t, err)
		assert.Equal(t, "main", output)

		// Test command with multiple args
		output, err = mockClient.Run("log", "-1", "--format=%h")
		require.NoError(t, err)
		assert.NotEmpty(t, output)
	})

	t.Run("command failures", func(t *testing.T) {
		mockClient := mocks.NewMockedGitClient()
		mockClient.SetCommandFailure("status --porcelain", errors.New("not a git repository"))

		_, err := mockClient.Run("status", "--porcelain")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})
}
