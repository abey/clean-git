package git

import "time"

type Branch struct {
	Name               string
	IsCurrent          bool
	IsRemote           bool
	IsMerged           bool
	LastCommitAt       time.Time
	LastCommitSHA      string
	AuthorUserName     string
	AuthorEmail        string
	HasUnpushedCommits bool
	Remote             string
}
