//go:generate mockery --name Usecase

package api

import (
	uiConfigModels "github.com/monitoror/monitoror/api/config/models"
	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"
)

const (
	GiteaCountTileType         coreModels.TileType = "GITEA-COUNT"
	GiteaChecksTileType        coreModels.TileType = "GITEA-CHECKS"
	GiteaPullRequestTileType   coreModels.TileType = "GITEA-PULLREQUEST"
	GiteaContributionsTileType coreModels.TileType = "GITEA-CONTRIBUTIONS"
	GiteaMilestoneTileType     coreModels.TileType = "GITEA-MILESTONE"

	// Team board
	GiteaPullRequestsTileType coreModels.TileType = "GITEA-PULLREQUESTS"
	GiteaIssuesTileType       coreModels.TileType = "GITEA-ISSUES"
	GiteaActionsTileType      coreModels.TileType = "GITEA-ACTIONS"
	GiteaMilestonesTileType   coreModels.TileType = "GITEA-MILESTONES"
	GiteaCommitsTileType      coreModels.TileType = "GITEA-COMMITS"
	GiteaRepositoryTileType   coreModels.TileType = "GITEA-REPOSITORY"
)

type (
	Usecase interface {
		Count(params *models.CountParams) (*coreModels.Tile, error)
		Checks(params *models.ChecksParams) (*coreModels.Tile, error)
		PullRequest(params *models.PullRequestParams) (*coreModels.Tile, error)
		Contributions(params *models.ContributionsParams) (*coreModels.Tile, error)
		Milestone(params *models.MilestoneParams) (*coreModels.Tile, error)
		PullRequests(params *models.ListParams) (*coreModels.Tile, error)
		Issues(params *models.IssuesListParams) (*coreModels.Tile, error)
		Actions(params *models.ListParams) (*coreModels.Tile, error)
		Milestones(params *models.ListParams) (*coreModels.Tile, error)
		Commits(params *models.ListParams) (*coreModels.Tile, error)
		Repository(params *models.RepositoryParams) (*coreModels.Tile, error)

		ChecksGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error)
		PullRequestsGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error)
		MilestonesGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error)
		RepositoriesGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error)
	}
)
