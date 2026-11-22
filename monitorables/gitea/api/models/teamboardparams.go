package models

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
	"github.com/monitoror/monitoror/internal/pkg/validator"
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

// namePattern matches Gitea owner and repository names
var namePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// validateNames rejects names that would change the path of the Gitea API request ("..", "/", ...).
// Empty names are left to the required tags.
func validateNames(names map[string]string) []validator.Error {
	var errs []validator.Error
	for _, field := range []string{"Owner", "Repository", "User", "Organization"} {
		name, ok := names[field]
		if !ok || name == "" {
			continue
		}
		if name == "." || name == ".." || !namePattern.MatchString(name) {
			errs = append(errs, validator.NewDefaultError(field, "a Gitea name (letters, digits, '-', '_', '.')"))
		}
	}
	return errs
}

// validateRef rejects refs with "." or ".." path segments
func validateRef(ref string) []validator.Error {
	for _, segment := range strings.Split(ref, "/") {
		if segment == "." || segment == ".." {
			return []validator.Error{validator.NewDefaultError("Ref", "a branch, tag or commit")}
		}
	}
	return nil
}

func (p *ListParams) Validate() []validator.Error {
	return validateNames(map[string]string{"Owner": p.Owner})
}

func (p *IssuesListParams) Validate() []validator.Error {
	return validateNames(map[string]string{"Owner": p.Owner})
}

func (p *RepositoryParams) Validate() []validator.Error {
	return append(validateNames(map[string]string{"Owner": p.Owner, "Repository": p.Repository}), validateRef(p.Ref)...)
}

func (p *RepositoryGeneratorParams) Validate() []validator.Error {
	return validateNames(map[string]string{"Owner": p.Owner})
}
