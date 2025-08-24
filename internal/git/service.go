package git

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type BranchService interface {
	GetCurrentBranch() (*Branch, error)
	GetMergedBranches(baseBranch string) ([]Branch, error)
	GetAllBranches() ([]Branch, error)
	GetBranchByName(branchName string) (*Branch, error)
	DeleteBranch(branch *Branch) error
	IsProtectedBranch(branch *Branch, patterns []string) bool
}

type TestableGitClient interface {
	GetCurrentBranchName() (string, error)
	GetMergedBranchNames(baseBranch string) ([]string, error)
	GetAllBranchNames() ([]string, error)
	GetBranchCommitInfo(branchName string) (string, error)
	DeleteLocalBranch(branchName string) error
	DeleteRemoteBranch(remote, branchName string) error
	HasUnpushedCommits(branchName string) (bool, error)
}

type DefaultBranchService struct {
	Client     gitClient
	RemoteName string
}

func NewBranchService(remoteName string) BranchService {
	return &DefaultBranchService{
		Client:     newGitClient(),
		RemoteName: remoteName,
	}
}

func NewBranchServiceWithClient(client TestableGitClient, remoteName string) BranchService {
	return &TestableBranchService{
		client:     client,
		RemoteName: remoteName,
	}
}

type TestableBranchService struct {
	client     TestableGitClient
	RemoteName string
}

func (s *DefaultBranchService) GetCurrentBranch() (*Branch, error) {
	branchName, err := s.Client.getCurrentBranchName()
	if err != nil {
		return nil, err
	}
	return s.GetBranchByName(branchName)
}

func (s *DefaultBranchService) GetMergedBranches(baseBranch string) ([]Branch, error) {
	branchNames, err := s.Client.getMergedBranchNames(baseBranch)
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, name := range branchNames {
		branch, err := s.GetBranchByName(name)
		if err != nil {
			continue
		}
		branch.IsMerged = true
		branches = append(branches, *branch)
	}

	return branches, nil
}

func (s *DefaultBranchService) GetAllBranches() ([]Branch, error) {
	branchNames, err := s.Client.getAllBranchNames()
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, name := range branchNames {
		if name == "origin/HEAD" {
			continue
		}

		branch, err := s.createBranchFromName(name)
		if err != nil {
			continue
		}
		branches = append(branches, *branch)
	}

	return branches, nil
}

func (s *DefaultBranchService) GetBranchByName(branchName string) (*Branch, error) {
	return s.createBranchFromName(branchName)
}

func (s *DefaultBranchService) DeleteBranch(branch *Branch) error {
	if branch.IsRemote {
		if branch.Remote == "" {
			if s.RemoteName != "" {
				branch.Remote = s.RemoteName
			} else {
				branch.Remote = "origin" // Fallback to origin when no remote is configured
			}
		}
		return s.Client.deleteRemoteBranch(branch.Remote, branch.Name)
	}
	return s.Client.deleteLocalBranch(branch.Name)
}

func (s *DefaultBranchService) IsProtectedBranch(branch *Branch, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, branch.Name)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func (s *DefaultBranchService) createBranchFromName(branchName string) (*Branch, error) {
	remoteName := "origin"
	if s.RemoteName != "" {
		remoteName = s.RemoteName
	}

	isRemote := strings.HasPrefix(branchName, remoteName+"/")
	actualName := branchName
	remote := ""

	if isRemote {
		actualName = strings.TrimPrefix(branchName, remoteName+"/")
		remote = remoteName
	}

	branchNameForCommitInfo := actualName
	if isRemote {
		branchNameForCommitInfo = branchName
	}

	commitInfo, err := s.Client.getBranchCommitInfo(branchNameForCommitInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info for branch %s: %w", branchNameForCommitInfo, err)
	}

	parts := strings.Split(commitInfo, "|")
	if len(parts) != 4 {
		return nil, fmt.Errorf("unexpected commit info format for branch %s", actualName)
	}

	commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", parts[0])
	if err != nil {
		commitDate = time.Time{}
	}

	currentBranchName, err := s.Client.getCurrentBranchName()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	hasUnpushed := false
	if !isRemote {
		hasUnpushed, _ = s.Client.hasUnpushedCommits(actualName)
	}

	branch := &Branch{
		Name:               actualName,
		IsCurrent:          actualName == currentBranchName && !isRemote,
		IsRemote:           isRemote,
		IsMerged:           false,
		LastCommitAt:       commitDate,
		LastCommitSHA:      strings.TrimSpace(parts[3]),
		AuthorUserName:     strings.TrimSpace(parts[1]),
		AuthorEmail:        strings.TrimSpace(parts[2]),
		HasUnpushedCommits: hasUnpushed,
		Remote:             remote,
	}

	return branch, nil
}

func (s *TestableBranchService) GetCurrentBranch() (*Branch, error) {
	branchName, err := s.client.GetCurrentBranchName()
	if err != nil {
		return nil, err
	}
	return s.GetBranchByName(branchName)
}

func (s *TestableBranchService) GetMergedBranches(baseBranch string) ([]Branch, error) {
	branchNames, err := s.client.GetMergedBranchNames(baseBranch)
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, name := range branchNames {
		branch, err := s.GetBranchByName(name)
		if err != nil {
			continue
		}
		branch.IsMerged = true
		branches = append(branches, *branch)
	}

	return branches, nil
}

func (s *TestableBranchService) GetAllBranches() ([]Branch, error) {
	branchNames, err := s.client.GetAllBranchNames()
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, name := range branchNames {
		if name == "origin/HEAD" {
			continue
		}

		branch, err := s.createBranchFromName(name)
		if err != nil {
			continue
		}
		branches = append(branches, *branch)
	}

	return branches, nil
}

func (s *TestableBranchService) GetBranchByName(branchName string) (*Branch, error) {
	return s.createBranchFromName(branchName)
}

func (s *TestableBranchService) DeleteBranch(branch *Branch) error {
	if branch.IsRemote {
		if branch.Remote == "" {
			if s.RemoteName != "" {
				branch.Remote = s.RemoteName
			} else {
				branch.Remote = "origin" // Fallback to origin when no remote is configured
			}
		}
		return s.client.DeleteRemoteBranch(branch.Remote, branch.Name)
	}
	return s.client.DeleteLocalBranch(branch.Name)
}

func (s *TestableBranchService) IsProtectedBranch(branch *Branch, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, branch.Name)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func (s *TestableBranchService) createBranchFromName(branchName string) (*Branch, error) {
	remoteName := "origin"
	if s.RemoteName != "" {
		remoteName = s.RemoteName
	}

	isRemote := strings.HasPrefix(branchName, remoteName+"/")
	actualName := branchName
	remote := ""

	if isRemote {
		actualName = strings.TrimPrefix(branchName, remoteName+"/")
		remote = remoteName
	}

	branchNameForCommitInfo := actualName
	if isRemote {
		branchNameForCommitInfo = branchName
	}

	commitInfo, err := s.client.GetBranchCommitInfo(branchNameForCommitInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info for branch %s: %w", branchNameForCommitInfo, err)
	}

	parts := strings.Split(commitInfo, "|")
	if len(parts) != 4 {
		return nil, fmt.Errorf("unexpected commit info format for branch %s", actualName)
	}

	commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", parts[0])
	if err != nil {
		commitDate = time.Time{}
	}

	currentBranchName, err := s.client.GetCurrentBranchName()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	hasUnpushed := false
	if !isRemote {
		hasUnpushed, _ = s.client.HasUnpushedCommits(actualName)
	}

	branch := &Branch{
		Name:               actualName,
		IsCurrent:          actualName == currentBranchName && !isRemote,
		IsRemote:           isRemote,
		IsMerged:           false,
		LastCommitAt:       commitDate,
		LastCommitSHA:      strings.TrimSpace(parts[3]),
		AuthorUserName:     strings.TrimSpace(parts[1]),
		AuthorEmail:        strings.TrimSpace(parts[2]),
		HasUnpushedCommits: hasUnpushed,
		Remote:             remote,
	}

	return branch, nil
}
