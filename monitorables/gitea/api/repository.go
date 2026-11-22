//go:generate mockery --name Repository

package api

import (
	"time"

	"github.com/monitoror/monitoror/monitorables/gitea/api/models"
)

type (
	Repository interface {
		GetCount(params *models.CountParams) (int, error)
		GetChecks(owner, repository, ref string) (*models.Checks, error)
		GetRuns(owner, repository, branch string) ([]models.Run, error)
		GetPullRequest(owner, repository string, id int) (*models.PullRequest, error)
		GetPullRequests(owner, repository string) ([]models.PullRequest, error)
		SearchPullRequests(owner string) ([]models.PullRequest, error)
		GetCommit(owner, repository, sha string) (*models.Commit, error)
		GetContributions(user string) ([]models.Contribution, error)
		GetRepositories(owner string) ([]models.Repository, error)
		GetMilestones(owner, repository string) ([]models.Milestone, error)

		// Team board
		SearchIssues(owner, kind string) ([]models.Issue, error)
		GetRepositoryIssues(owner, repository string) ([]models.Issue, error)
		GetOwnerRuns(owner string) ([]models.Run, error)
		GetCommits(owner, repository, ref string, since time.Time) ([]models.RepositoryCommit, error)
		GetFeedCommits(owner string, limit int) ([]models.FeedCommit, error)
		GetOrganizationMembers(organization string) ([]string, error)
	}
)
