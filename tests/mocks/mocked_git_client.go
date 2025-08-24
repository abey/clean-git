package mocks

import (
	"fmt"
	"strings"
	"time"
)

// DeleteRemoteBranchCall tracks calls to DeleteRemoteBranch for testing
type DeleteRemoteBranchCall struct {
	Remote     string
	BranchName string
}

// SophisticatedGitClient provides realistic git command simulation
type SophisticatedGitClient struct {
	currentBranch          string
	branches               map[string]BranchData
	remotes                map[string]string // branch -> remote
	unpushedCommits        map[string]int    // branch -> count
	commandFailures        map[string]error  // command -> error to return
	deleteRemoteBranchCalls []DeleteRemoteBranchCall // Track delete remote branch calls
}

type BranchData struct {
	Name        string
	CommitDate  time.Time
	AuthorName  string
	AuthorEmail string
	CommitSHA   string
	IsMerged    bool
	IsRemote    bool
	Remote      string
}

func NewMockedGitClient() *SophisticatedGitClient {
	now := time.Now()
	return &SophisticatedGitClient{
		currentBranch: "main",
		branches: map[string]BranchData{
			"main": {
				Name:        "main",
				CommitDate:  now,
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				CommitSHA:   "abc123",
				IsMerged:    false,
				IsRemote:    false,
			},
			"feature/test": {
				Name:        "feature/test",
				CommitDate:  now.Add(-24 * time.Hour),
				AuthorName:  "Jane Smith",
				AuthorEmail: "jane@example.com",
				CommitSHA:   "def456",
				IsMerged:    false,
				IsRemote:    false,
			},
			"feature/merged": {
				Name:        "feature/merged",
				CommitDate:  now.Add(-48 * time.Hour),
				AuthorName:  "Bob Wilson",
				AuthorEmail: "bob@example.com",
				CommitSHA:   "ghi789",
				IsMerged:    true,
				IsRemote:    false,
			},
			"origin/main": {
				Name:        "main",
				CommitDate:  now,
				AuthorName:  "John Doe",
				AuthorEmail: "john@example.com",
				CommitSHA:   "abc123",
				IsMerged:    false,
				IsRemote:    true,
				Remote:      "origin",
			},
		},
		remotes:                 map[string]string{},
		unpushedCommits:         map[string]int{},
		commandFailures:         map[string]error{},
		deleteRemoteBranchCalls: []DeleteRemoteBranchCall{},
	}
}

// Configuration methods for test setup
func (m *SophisticatedGitClient) SetCurrentBranch(branch string) {
	m.currentBranch = branch
}

func (m *SophisticatedGitClient) ClearBranches() {
	m.branches = make(map[string]BranchData)
}

func (m *SophisticatedGitClient) AddBranch(data BranchData) {
	m.branches[data.Name] = data
}

func (m *SophisticatedGitClient) SetUnpushedCommits(branch string, count int) {
	m.unpushedCommits[branch] = count
}

func (m *SophisticatedGitClient) SetCommandFailure(command string, err error) {
	m.commandFailures[command] = err
}

// GetDeleteRemoteBranchCalls returns all tracked DeleteRemoteBranch calls for testing
func (m *SophisticatedGitClient) GetDeleteRemoteBranchCalls() []DeleteRemoteBranchCall {
	return m.deleteRemoteBranchCalls
}

// GitClient interface implementation
func (m *SophisticatedGitClient) Run(args ...string) (string, error) {
	command := strings.Join(args, " ")

	// Check for configured failures
	if err, exists := m.commandFailures[command]; exists {
		return "", err
	}

	// Simulate basic git commands
	switch {
	case strings.HasPrefix(command, "branch --show-current"):
		return m.currentBranch, nil
	case strings.HasPrefix(command, "branch --merged"):
		return m.getMergedBranchesOutput(args), nil
	case strings.HasPrefix(command, "branch --all"):
		return m.getAllBranchesOutput(), nil
	case strings.HasPrefix(command, "log -1 --format="):
		if strings.Contains(command, "%h") {
			return "abc123", nil
		}
		return m.getCommitInfoOutput(args), nil
	case strings.HasPrefix(command, "rev-list --count"):
		return m.getUnpushedCountOutput(args), nil
	default:
		return "", nil
	}
}

func (m *SophisticatedGitClient) GetCurrentBranchName() (string, error) {
	if err, exists := m.commandFailures["GetCurrentBranchName"]; exists {
		return "", err
	}
	return m.currentBranch, nil
}

func (m *SophisticatedGitClient) GetMergedBranchNames(baseBranch string) ([]string, error) {
	if err, exists := m.commandFailures["GetMergedBranchNames"]; exists {
		return nil, err
	}

	var merged []string
	for name, data := range m.branches {
		if data.IsMerged && name != baseBranch {
			// Include both local and remote merged branches
			if data.IsRemote {
				// Use the actual remote name from the branch data, or fallback to "origin"
				remoteName := data.Remote
				if remoteName == "" {
					remoteName = "origin"
				}
				// If the branch name already includes the remote prefix, use it as-is
				if strings.HasPrefix(name, remoteName+"/") {
					merged = append(merged, name)
				} else {
					merged = append(merged, remoteName+"/"+name)
				}
			} else {
				merged = append(merged, name)
			}
		}
	}
	return merged, nil
}

func (m *SophisticatedGitClient) GetAllBranchNames() ([]string, error) {
	if err, exists := m.commandFailures["GetAllBranchNames"]; exists {
		return nil, err
	}

	var all []string
	for _, data := range m.branches {
		if data.IsRemote {
			// Use the actual remote name from the branch data, or fallback to "origin"
			remoteName := data.Remote
			if remoteName == "" {
				remoteName = "origin"
			}
			// If the branch name already includes the remote prefix, use it as-is
			if strings.HasPrefix(data.Name, remoteName+"/") {
				all = append(all, data.Name)
			} else {
				all = append(all, remoteName+"/"+data.Name)
			}
		} else {
			all = append(all, data.Name)
		}
	}
	return all, nil
}

func (m *SophisticatedGitClient) GetBranchCommitInfo(branchName string) (string, error) {
	if err, exists := m.commandFailures["GetBranchCommitInfo"]; exists {
		return "", err
	}

	// Try to find the branch by name, handling remote branch name variations
	data, exists := m.branches[branchName]
	if !exists {
		// For remote branches, try to find by the base name without remote prefix
		for storedName, storedData := range m.branches {
			if storedData.IsRemote {
				remoteName := storedData.Remote
				if remoteName == "" {
					remoteName = "origin"
				}
				// If the requested branch name matches the full remote name
				if branchName == storedName {
					data = storedData
					exists = true
					break
				}
				// If the requested branch name is the remote/branch format and stored name is just the branch
				if branchName == remoteName+"/"+storedName {
					data = storedData
					exists = true
					break
				}
			}
		}
	}
	
	if !exists {
		return "", fmt.Errorf("branch %s not found", branchName)
	}

	return fmt.Sprintf("%s|%s|%s|%s",
		data.CommitDate.Format("2006-01-02 15:04:05 -0700"),
		data.AuthorName,
		data.AuthorEmail,
		data.CommitSHA,
	), nil
}

func (m *SophisticatedGitClient) DeleteLocalBranch(branchName string) error {
	if err, exists := m.commandFailures["DeleteLocalBranch"]; exists {
		return err
	}

	if branchName == m.currentBranch {
		return fmt.Errorf("cannot delete current branch %s", branchName)
	}

	delete(m.branches, branchName)
	return nil
}

func (m *SophisticatedGitClient) DeleteRemoteBranch(remote, branchName string) error {
	if err, exists := m.commandFailures["DeleteRemoteBranch"]; exists {
		return err
	}

	// Track the call for testing
	m.deleteRemoteBranchCalls = append(m.deleteRemoteBranchCalls, DeleteRemoteBranchCall{
		Remote:     remote,
		BranchName: branchName,
	})

	key := remote + "/" + branchName
	delete(m.branches, key)
	return nil
}

func (m *SophisticatedGitClient) HasUnpushedCommits(branchName string) (bool, error) {
	if err, exists := m.commandFailures["HasUnpushedCommits"]; exists {
		return false, err
	}

	count, exists := m.unpushedCommits[branchName]
	return exists && count > 0, nil
}

func (m *SophisticatedGitClient) BranchExists(branchName string) (bool, error) {
	if err, exists := m.commandFailures["BranchExists"]; exists {
		return false, err
	}

	// Check if branch exists in our mock data
	_, exists := m.branches[branchName]
	if exists {
		return true, nil
	}

	// Also check for remote branches with different naming patterns
	for storedName, storedData := range m.branches {
		if storedData.IsRemote {
			remoteName := storedData.Remote
			if remoteName == "" {
				remoteName = "origin"
			}
			// Check if requested branch matches remote/branch format
			if branchName == remoteName+"/"+storedData.Name || branchName == storedName {
				return true, nil
			}
		}
	}

	return false, nil
}

// Helper methods for output simulation
func (m *SophisticatedGitClient) getMergedBranchesOutput(args []string) string {
	var output []string
	baseBranch := args[2] // --merged <baseBranch>

	for name, data := range m.branches {
		if data.IsMerged && name != baseBranch && !data.IsRemote {
			prefix := ""
			if name == m.currentBranch {
				prefix = "* "
			} else {
				prefix = "  "
			}
			output = append(output, prefix+name)
		}
	}
	return strings.Join(output, "\n")
}

func (m *SophisticatedGitClient) getAllBranchesOutput() string {
	var output []string

	for name, data := range m.branches {
		if !data.IsRemote {
			prefix := ""
			if name == m.currentBranch {
				prefix = "* "
			} else {
				prefix = "  "
			}
			output = append(output, prefix+name)
		}
	}

	for _, data := range m.branches {
		if data.IsRemote {
			output = append(output, "  remotes/origin/"+data.Name)
		}
	}

	return strings.Join(output, "\n")
}

func (m *SophisticatedGitClient) getCommitInfoOutput(args []string) string {
	// Extract branch name from log command
	branchName := args[len(args)-1]

	data, exists := m.branches[branchName]
	if !exists {
		return ""
	}

	return fmt.Sprintf("%s|%s|%s|%s",
		data.CommitDate.Format("2006-01-02 15:04:05 -0700"),
		data.AuthorName,
		data.AuthorEmail,
		data.CommitSHA,
	)
}

func (m *SophisticatedGitClient) getUnpushedCountOutput(args []string) string {
	// Extract branch name from rev-list command
	revRange := args[2] // branchName@{upstream}..branchName
	branchName := strings.Split(revRange, "@")[0]

	count, exists := m.unpushedCommits[branchName]
	if !exists {
		return "0"
	}
	return fmt.Sprintf("%d", count)
}
