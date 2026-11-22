package http

import (
	"errors"
	"net/http"
	"testing"

	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api"
	"github.com/monitoror/monitoror/monitorables/gitea/api/mocks"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	"github.com/stretchr/testify/assert"
)

func TestDelivery_GetMilestone(t *testing.T) {
	tile := coreModels.NewTile(api.GiteaMilestoneTileType)
	params := &models.MilestoneParams{Owner: "test", Repository: "test"}

	for _, testcase := range []struct {
		name       string
		withParams bool
		err        error
	}{
		{name: "success", withParams: true},
		{name: "usecase error", withParams: true, err: errors.New("boom")},
		{name: "missing params"},
	} {
		ctx, res := initEcho()
		if testcase.withParams {
			ctx.QueryParams().Set("owner", "test")
			ctx.QueryParams().Set("repository", "test")
		}

		mockUsecase := new(mocks.Usecase)
		if testcase.withParams {
			if testcase.err != nil {
				mockUsecase.On("Milestone", params).Return(nil, testcase.err)
			} else {
				mockUsecase.On("Milestone", params).Return(tile, nil)
			}
		}

		err := NewGiteaDelivery(mockUsecase).GetMilestone(ctx)
		if testcase.withParams && testcase.err == nil {
			assert.NoError(t, err, testcase.name)
			assert.Equal(t, http.StatusOK, res.Code, testcase.name)
		} else {
			assert.Error(t, err, testcase.name)
		}
		mockUsecase.AssertExpectations(t)
	}
}
