package git

import "time"

type Branch struct {
	Name             string
	IsCurrent        bool
	IsRemote         bool
	IsMerged         bool
	LastCommitAt     time.Time
	// LatestCommitSHA  string
	AuthorUserName   string
}