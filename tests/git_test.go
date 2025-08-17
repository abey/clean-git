package clean_git_tests

import (
	"testing"

	"clean-git/internal/git"
	"clean-git/tests/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetMergedBranches(t *testing.T) {
	mockClient := mocks.NewMockGitClient("merged_branch_names.txt")

	branches, err := git.GetMergedBranches(mockClient, "main")
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"feature/JIRA-1234-do-stuff",
		"abinodh/final-final-v2",
		"bugfix/oh-god-why",
		"release/v1.2.3-beta",
	}, branches)
}

func TestGetAllBranches(t *testing.T) {
	mockClient := mocks.NewMockGitClient("all_branch_names.txt")

	branches, err := git.GetAllBranches(mockClient)
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"main",
		"feature/JIRA-1234-do-stuff",
		"feature/someone-is-working-on-stuff",
		"abinodh/final-final-v2",
		"bugfix/oh-god-why",
		"release/v1.2.3-beta",
		"abinodh/final-stuff",
		"hotfix/murphys-law-is-true",
	}, branches)
}
