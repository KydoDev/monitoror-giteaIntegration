package http

import (
	"net/http"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/delivery"
	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	"github.com/labstack/echo/v4"
)

// serveTile binds and validates the params, then returns the tile computed by load
func serveTile[P params.Validator](c echo.Context, p P, load func(P) (*coreModels.Tile, error)) error {
	if err := delivery.BindAndValidateParams(c, p); err != nil {
		return err
	}

	tile, err := load(p)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, tile)
}

func (h *GiteaDelivery) GetPullRequests(c echo.Context) error {
	return serveTile(c, &models.ListParams{}, h.giteaUsecase.PullRequests)
}

func (h *GiteaDelivery) GetIssues(c echo.Context) error {
	return serveTile(c, &models.IssuesListParams{}, h.giteaUsecase.Issues)
}

func (h *GiteaDelivery) GetActions(c echo.Context) error {
	return serveTile(c, &models.ListParams{}, h.giteaUsecase.Actions)
}

func (h *GiteaDelivery) GetMilestones(c echo.Context) error {
	return serveTile(c, &models.ListParams{}, h.giteaUsecase.Milestones)
}

func (h *GiteaDelivery) GetCommits(c echo.Context) error {
	return serveTile(c, &models.ListParams{}, h.giteaUsecase.Commits)
}

func (h *GiteaDelivery) GetRepository(c echo.Context) error {
	return serveTile(c, &models.RepositoryParams{}, h.giteaUsecase.Repository)
}
