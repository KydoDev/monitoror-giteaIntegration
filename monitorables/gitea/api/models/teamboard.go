package models

import "time"

type (
	// Issue is an open issue or pull request found by an owner-wide search
	Issue struct {
		Number     int
		Title      string
		Owner      string
		Repository string
		Labels     []string
		CreatedAt  time.Time
	}

	// RepositoryCommit is a commit of a repository, used for commit activity
	RepositoryCommit struct {
		SHA  string
		Date time.Time
	}

	// FeedCommit is a pushed commit read from an activity feed
	FeedCommit struct {
		Repository string
		Branch     string
		Author     string
		Message    string
		PushedAt   time.Time
	}
)
