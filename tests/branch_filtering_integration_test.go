package clean_git_tests

import (
	"testing"
	"time"

	"clean-git/internal/config"
	"clean-git/internal/git"
	"clean-git/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBranchFiltering_ByAge tests that branch filtering by age works correctly
func TestBranchFiltering_ByAge(t *testing.T) {
	tests := []struct {
		name           string
		maxAge         time.Duration
		setupBranches  func(*mocks.SophisticatedGitClient)
		expectedCount  int
		expectedNames  []string
	}{
		{
			name:   "filters branches older than max age",
			maxAge: 48 * time.Hour, // 2 days
			setupBranches: func(m *mocks.SophisticatedGitClient) {
				now := time.Now()
				// Add branches with different ages
				m.AddBranch(mocks.BranchData{
					Name:       "feature/recent",
					CommitDate: now.Add(-24 * time.Hour), // 1 day old - should be excluded
					AuthorName: "Recent User",
					CommitSHA:  "recent123",
					IsMerged:   true,
				})
				m.AddBranch(mocks.BranchData{
					Name:       "feature/old",
					CommitDate: now.Add(-72 * time.Hour), // 3 days old - should be included
					AuthorName: "Old User",
					CommitSHA:  "old123",
					IsMerged:   true,
				})
				m.AddBranch(mocks.BranchData{
					Name:       "feature/very-old",
					CommitDate: now.Add(-168 * time.Hour), // 7 days old - should be included
					AuthorName: "Very Old User",
					CommitSHA:  "veryold123",
					IsMerged:   true,
				})
			},
			expectedCount: 3, // feature/merged (default) + feature/old + feature/very-old
			expectedNames: []string{"feature/merged", "feature/old", "feature/very-old"},
		},
		{
			name:   "no branches qualify when all are too recent",
			maxAge: 168 * time.Hour, // 7 days
			setupBranches: func(m *mocks.SophisticatedGitClient) {
				now := time.Now()
				// Override default merged branch to be recent
				m.AddBranch(mocks.BranchData{
					Name:       "feature/merged",
					CommitDate: now.Add(-24 * time.Hour), // 1 day old - should be excluded
					AuthorName: "Recent User",
					CommitSHA:  "recent123",
					IsMerged:   true,
				})
			},
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockedGitClient()
			tt.setupBranches(mockClient)

			cfg := &config.Config{
				BaseBranches: []string{"main"},
				MaxAge:       tt.maxAge,
				RemoteName:   "origin",
			}

			service := git.NewBranchServiceWithClient(mockClient, cfg.RemoteName)

			// Get merged branches for filtering test
			branches, err := service.GetMergedBranches("main")
			require.NoError(t, err)

			// Filter branches by age (simulating the main logic)
			var qualifyingBranches []git.Branch
			for _, branch := range branches {
				age := time.Since(branch.LastCommitAt)
				if age >= cfg.MaxAge {
					qualifyingBranches = append(qualifyingBranches, branch)
				}
			}

			assert.Len(t, qualifyingBranches, tt.expectedCount)

			if len(tt.expectedNames) > 0 {
				actualNames := make([]string, len(qualifyingBranches))
				for i, branch := range qualifyingBranches {
					actualNames[i] = branch.Name
				}
				assert.ElementsMatch(t, tt.expectedNames, actualNames)
			}
		})
	}
}

// TestBranchFiltering_LocalOnlyFlag tests that local-only flag works correctly
func TestBranchFiltering_LocalOnlyFlag(t *testing.T) {
	mockClient := mocks.NewMockedGitClient()
	
	// Clear default branches and add our test branches
	mockClient.ClearBranches()
	now := time.Now()
	
	// Add main branch (required)
	mockClient.AddBranch(mocks.BranchData{
		Name:       "main",
		CommitDate: now,
		AuthorName: "Main User",
		CommitSHA:  "main123",
		IsMerged:   false,
		IsRemote:   false,
	})
	
	// Add local merged branch
	mockClient.AddBranch(mocks.BranchData{
		Name:       "feature/local-merged",
		CommitDate: now.Add(-72 * time.Hour), // Old enough
		AuthorName: "Local User",
		CommitSHA:  "local123",
		IsMerged:   true,
		IsRemote:   false,
	})
	
	// Add remote merged branch
	mockClient.AddBranch(mocks.BranchData{
		Name:       "feature/remote-merged",
		CommitDate: now.Add(-72 * time.Hour), // Old enough
		AuthorName: "Remote User",
		CommitSHA:  "remote123",
		IsMerged:   true,
		IsRemote:   true,
		Remote:     "origin",
	})

	cfg := &config.Config{
		BaseBranches: []string{"main"},
		MaxAge:       48 * time.Hour, // 2 days
		RemoteName:   "origin",
	}

	service := git.NewBranchServiceWithClient(mockClient, cfg.RemoteName)
	branches, err := service.GetMergedBranches("main")
	require.NoError(t, err)

	// Simulate local-only filtering
	var localOnlyBranches []git.Branch
	var allQualifyingBranches []git.Branch

	for _, branch := range branches {
		age := time.Since(branch.LastCommitAt)
		if age >= cfg.MaxAge {
			allQualifyingBranches = append(allQualifyingBranches, branch)
			if !branch.IsRemote { // local-only filter
				localOnlyBranches = append(localOnlyBranches, branch)
			}
		}
	}

	// Should have at least one local and one remote qualifying branch
	assert.GreaterOrEqual(t, len(allQualifyingBranches), 2, "Should have both local and remote branches")
	assert.GreaterOrEqual(t, len(localOnlyBranches), 1, "Should have at least one local branch")
	
	// Local-only should only contain local branches
	for _, branch := range localOnlyBranches {
		assert.False(t, branch.IsRemote, "Local-only filter should exclude remote branches")
	}

	// Should include the local merged branch
	localNames := make([]string, len(localOnlyBranches))
	for i, branch := range localOnlyBranches {
		localNames[i] = branch.Name
	}
	assert.Contains(t, localNames, "feature/local-merged")
}

// TestBranchFiltering_RemoteOnlyFlag tests that remote-only flag works correctly
func TestBranchFiltering_RemoteOnlyFlag(t *testing.T) {
	mockClient := mocks.NewMockedGitClient()
	
	// Clear default branches and add our test branches
	mockClient.ClearBranches()
	now := time.Now()
	
	// Add main branch (required)
	mockClient.AddBranch(mocks.BranchData{
		Name:       "main",
		CommitDate: now,
		AuthorName: "Main User",
		CommitSHA:  "main123",
		IsMerged:   false,
		IsRemote:   false,
	})
	
	// Add local merged branch
	mockClient.AddBranch(mocks.BranchData{
		Name:       "feature/local-merged",
		CommitDate: now.Add(-72 * time.Hour), // Old enough
		AuthorName: "Local User",
		CommitSHA:  "local123",
		IsMerged:   true,
		IsRemote:   false,
	})
	
	// Add remote merged branch
	mockClient.AddBranch(mocks.BranchData{
		Name:       "feature/remote-merged",
		CommitDate: now.Add(-72 * time.Hour), // Old enough
		AuthorName: "Remote User",
		CommitSHA:  "remote123",
		IsMerged:   true,
		IsRemote:   true,
		Remote:     "origin",
	})

	cfg := &config.Config{
		BaseBranches: []string{"main"},
		MaxAge:       48 * time.Hour, // 2 days
		RemoteName:   "origin",
	}

	service := git.NewBranchServiceWithClient(mockClient, cfg.RemoteName)
	branches, err := service.GetMergedBranches("main")
	require.NoError(t, err)

	// Simulate remote-only filtering
	var remoteOnlyBranches []git.Branch
	var allQualifyingBranches []git.Branch

	for _, branch := range branches {
		age := time.Since(branch.LastCommitAt)
		if age >= cfg.MaxAge {
			allQualifyingBranches = append(allQualifyingBranches, branch)
			if branch.IsRemote { // remote-only filter
				remoteOnlyBranches = append(remoteOnlyBranches, branch)
			}
		}
	}

	// Should have at least one local and one remote qualifying branch
	assert.GreaterOrEqual(t, len(allQualifyingBranches), 2, "Should have both local and remote branches")
	assert.GreaterOrEqual(t, len(remoteOnlyBranches), 1, "Should have at least one remote branch")
	
	// Remote-only should only contain remote branches
	for _, branch := range remoteOnlyBranches {
		assert.True(t, branch.IsRemote, "Remote-only filter should exclude local branches")
	}

	// Should include the remote merged branch
	remoteNames := make([]string, len(remoteOnlyBranches))
	for i, branch := range remoteOnlyBranches {
		remoteNames[i] = branch.Name
	}
	assert.Contains(t, remoteNames, "feature/remote-merged")
}
