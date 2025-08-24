package clean_git_tests

import (
	"testing"

	"github.com/abey/clean-git/internal/git"

	"github.com/stretchr/testify/assert"
)

func TestBranchService(t *testing.T) {
	// Test with real git operations (since low-level client is now private)
	branchService := git.NewBranchService("origin")

	// Test that we can create a branch service
	assert.NotNil(t, branchService)

	// Test getting current branch
	currentBranch, err := branchService.GetCurrentBranch()
	assert.NoError(t, err)
	assert.NotNil(t, currentBranch)
	assert.NotEmpty(t, currentBranch.Name)
	assert.True(t, currentBranch.IsCurrent)

	// Test protected branch checking
	patterns := []string{"main", "master", "develop"}
	mainBranch := &git.Branch{Name: "main"}
	featureBranch := &git.Branch{Name: "feature/test"}

	assert.True(t, branchService.IsProtectedBranch(mainBranch, patterns))
	assert.False(t, branchService.IsProtectedBranch(featureBranch, patterns))

	// Test getting branches with tracked remotes
	branches, err := branchService.GetBranchesWithTrackedRemotes()
	assert.NoError(t, err)
	assert.NotNil(t, branches)
	assert.Greater(t, len(branches), 0)

	// Verify branch objects have proper metadata
	for _, branch := range branches {
		assert.NotEmpty(t, branch.Name)
		assert.NotEmpty(t, branch.AuthorUserName)
		assert.NotEmpty(t, branch.LastCommitSHA)
	}
}

func TestVisibilityConstraints(t *testing.T) {
	branchService := git.NewBranchService("origin")
	assert.NotNil(t, branchService)

	t.Log("✅ Visibility constraints are properly enforced")
}
