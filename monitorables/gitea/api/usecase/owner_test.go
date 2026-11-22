package usecase

import (
	"errors"
	"testing"

	uiConfigModels "github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api/mocks"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	. "github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
)

func TestPullRequestsGenerator_EveryRepository(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("SearchPullRequests", "org").
		Return([]models.PullRequest{{ID: 4, Owner: "org", Repository: "api"}, {ID: 2, Owner: "org", Repository: "web"}}, nil)

	gu := NewGiteaUsecase(mockRepository)

	tiles, err := gu.PullRequestsGenerator(&models.PullRequestGeneratorParams{Owner: "org"})
	if assert.NoError(t, err) {
		assert.Equal(t, []uiConfigModels.GeneratedTile{
			{Params: &models.PullRequestParams{Owner: "org", Repository: "api", ID: ToInt(4)}},
			{Params: &models.PullRequestParams{Owner: "org", Repository: "web", ID: ToInt(2)}},
		}, tiles)
		mockRepository.AssertNotCalled(t, "GetPullRequests")
	}
}

func TestPullRequestsGenerator_EveryRepositoryError(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("SearchPullRequests", "org").Return(nil, errors.New("boom"))

	gu := NewGiteaUsecase(mockRepository)

	tiles, err := gu.PullRequestsGenerator(&models.PullRequestGeneratorParams{Owner: "org"})
	assert.Error(t, err)
	assert.Nil(t, tiles)
}
