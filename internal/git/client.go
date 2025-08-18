package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// gitClient handles raw git command execution (internal interface)
type gitClient interface {
	run(args ...string) (string, error)
	getCurrentBranchName() (string, error)
	getMergedBranchNames(baseBranch string) ([]string, error)
	getAllBranchNames() ([]string, error)
	getBranchCommitInfo(branchName string) (string, error) // Returns formatted commit info
	deleteLocalBranch(branchName string) error
	deleteRemoteBranch(remote, branchName string) error
	hasUnpushedCommits(branchName string) (bool, error)
	getCurrentUserName() (string, error)
	getCurrentUserEmail() (string, error)
}

type defaultGitClient struct{}

func newGitClient() gitClient {
	return &defaultGitClient{}
}

func (c *defaultGitClient) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w", err)
	}
	return string(output), nil
}

func (c *defaultGitClient) getCurrentBranchName() (string, error) {
	output, err := c.run("branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(output), nil
}

func (c *defaultGitClient) getMergedBranchNames(baseBranch string) ([]string, error) {
	output, err := c.run("branch", "--merged", baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get merged branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var branches []string

	for _, line := range lines {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" && branch != baseBranch {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

func (c *defaultGitClient) getAllBranchNames() ([]string, error) {
	output, err := c.run("branch", "--all")
	if err != nil {
		return nil, fmt.Errorf("failed to get all branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var branches []string

	for _, line := range lines {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" && branch != "origin/HEAD" {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

func (c *defaultGitClient) getBranchCommitInfo(branchName string) (string, error) {
	output, err := c.run("log", "-1", "--format=%ci|%an|%ae|%h", branchName)
	if err != nil {
		return "", fmt.Errorf("failed to get branch commit info for %s: %w", branchName, err)
	}
	return strings.TrimSpace(output), nil
}

func (c *defaultGitClient) deleteLocalBranch(branchName string) error {
	_, err := c.run("branch", "-d", branchName)
	if err != nil {
		// Try force delete if regular delete fails
		_, forceErr := c.run("branch", "-D", branchName)
		if forceErr != nil {
			return fmt.Errorf("failed to delete local branch %s: %w", branchName, err)
		}
	}
	return nil
}

func (c *defaultGitClient) deleteRemoteBranch(remote, branchName string) error {
	_, err := c.run("push", remote, "--delete", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete remote branch %s/%s: %w", remote, branchName, err)
	}
	return nil
}

func (c *defaultGitClient) hasUnpushedCommits(branchName string) (bool, error) {
	output, err := c.run("rev-list", "--count", branchName+"@{upstream}.."+branchName)
	if err != nil {
		// If there's no upstream, assume no unpushed commits
		return false, nil
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return false, fmt.Errorf("failed to parse commit count: %w", err)
	}

	return count > 0, nil
}

// GetCurrentUserName retrieves the git user.name configuration
func (c *defaultGitClient) getCurrentUserName() (string, error) {
	output, err := c.run("config", "user.name")
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(output)
	if name == "" {
		return "", fmt.Errorf("user.name is not set in git config")
	}
	return name, nil
}

func (c *defaultGitClient) getCurrentUserEmail() (string, error) {
	output, err := c.run("config", "user.email")
	if err != nil {
		return "", err
	}
	email := strings.TrimSpace(output)
	if email == "" {
		return "", fmt.Errorf("user.email is not set in git config")
	}
	return email, nil
}
