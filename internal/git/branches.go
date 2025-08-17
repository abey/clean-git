package git

import (
	"strings"
)

// GetMergedBranches returns a list of local branches that are merged into the base branch (e.g., "main").
func GetMergedBranches(client GitClient, baseBranch string) ([]string, error) {
	output, err := client.Run("git", "branch", "--merged", baseBranch)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	var branches []string

	for _, line := range lines {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" && branch != baseBranch {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

func GetAllBranches(client GitClient) ([]string, error) {
	output, err := client.Run("git", "branch", "--all")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	var branches []string

	for _, line := range lines {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}
