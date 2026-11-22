package models

import (
	coreModels "github.com/monitoror/monitoror/models"
)

type PullRequest struct {
	ID int
	// Repository of the pull request, set when searching every repository of an owner
	Owner      string
	Repository string
	Title      string
	Author     coreModels.Author

	SourceOwner      string
	SourceRepository string
	SourceBranch     string
	CommitSHA        string
}
