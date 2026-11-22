package models

import (
	"fmt"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
)

type (
	// ListParams are the params of the owner-wide list tiles (pull requests, actions, milestones, commits)
	ListParams struct {
		params.Default

		Owner string `json:"owner" query:"owner" validate:"required"`
		Limit int    `json:"limit,omitempty" query:"limit" validate:"omitempty,gte=1,lte=20"`
	}

	// IssuesListParams are the params of the owner-wide issues tile
	IssuesListParams struct {
		params.Default

		Owner    string `json:"owner" query:"owner" validate:"required"`
		Limit    int    `json:"limit,omitempty" query:"limit" validate:"omitempty,gte=1,lte=20"`
		BugLabel string `json:"bugLabel,omitempty" query:"bugLabel"`
	}

	// RepositoryParams are the params of the repository summary tile
	RepositoryParams struct {
		params.Default

		Owner      string `json:"owner" query:"owner" validate:"required"`
		Repository string `json:"repository" query:"repository" validate:"required"`
		Ref        string `json:"ref,omitempty" query:"ref"`
		BugLabel   string `json:"bugLabel,omitempty" query:"bugLabel"`
	}

	RepositoryGeneratorParams struct {
		params.Default

		Owner    string `json:"owner" query:"owner" validate:"required"`
		BugLabel string `json:"bugLabel,omitempty" query:"bugLabel"`
	}
)

// Used by cache as identifier
func (p *ListParams) String() string {
	return fmt.Sprintf("LIST-%s-%d", p.Owner, p.Limit)
}

// Used by cache as identifier
func (p *IssuesListParams) String() string {
	return fmt.Sprintf("ISSUES-%s-%d-%s", p.Owner, p.Limit, p.BugLabel)
}

// Used by cache as identifier
func (p *RepositoryParams) String() string {
	return fmt.Sprintf("REPOSITORY-%s-%s-%s-%s", p.Owner, p.Repository, p.Ref, p.BugLabel)
}
