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
)

type (
	Usecase interface {
		Count(params *models.CountParams) (*coreModels.Tile, error)
		Checks(params *models.ChecksParams) (*coreModels.Tile, error)
		PullRequest(params *models.PullRequestParams) (*coreModels.Tile, error)
		Contributions(params *models.ContributionsParams) (*coreModels.Tile, error)

		ChecksGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error)
		PullRequestsGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error)
	}
)
