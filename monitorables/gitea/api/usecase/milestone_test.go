//go:build !faker

package usecase

import (
	"errors"
	"testing"
	"time"

	uiConfigModels "github.com/monitoror/monitoror/api/config/models"
	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api/mocks"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	. "github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
)

func withNow(t *testing.T, now time.Time) {
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = time.Now })
}

func TestMilestone(t *testing.T) {
	rome, err := time.LoadLocation("Europe/Rome")
	if !assert.NoError(t, err) {
		return
	}
	now := time.Date(2026, 11, 22, 15, 0, 0, 0, rome)
	withNow(t, now)
	due := func(days int) *time.Time { return ToTime(now.AddDate(0, 0, days)) }

	for _, testcase := range []struct {
		name      string
		milestone models.Milestone
		status    coreModels.TileStatus
		message   string
		value     string
	}{
		{"on track", models.Milestone{Title: "v2", OpenIssues: 3, ClosedIssues: 5, DueOn: due(6)}, coreModels.SuccessStatus, "v2 · due in 6 days", "62%"},
		{"close to due date", models.Milestone{Title: "v2", OpenIssues: 1, ClosedIssues: 1, DueOn: due(1)}, coreModels.WarningStatus, "v2 · due tomorrow", "50%"},
		{"due today", models.Milestone{Title: "v2", OpenIssues: 1, DueOn: due(0)}, coreModels.WarningStatus, "v2 · due today", "0%"},
		{"overdue", models.Milestone{Title: "v2", OpenIssues: 2, ClosedIssues: 2, DueOn: due(-1)}, coreModels.FailedStatus, "v2 · 1 day overdue", "50%"},
		{"overdue but done", models.Milestone{Title: "v2", ClosedIssues: 4, DueOn: due(-3)}, coreModels.SuccessStatus, "v2 · 3 days overdue", "100%"},
		{"no due date, no issue", models.Milestone{Title: "Backlog"}, coreModels.SuccessStatus, "Backlog", ""},
	} {
		mockRepository := new(mocks.Repository)
		mockRepository.On("GetMilestones", "test", "repo").Return([]models.Milestone{testcase.milestone}, nil)

		tile, err := NewGiteaUsecase(mockRepository).Milestone(&models.MilestoneParams{Owner: "test", Repository: "repo"})
		if assert.NoError(t, err, testcase.name) {
			assert.Equal(t, "repo", tile.Label, testcase.name)
			assert.Equal(t, testcase.status, tile.Status, testcase.name)
			assert.Equal(t, testcase.message, tile.Message, testcase.name)
			if testcase.value == "" {
				assert.Nil(t, tile.Metrics, testcase.name)
			} else {
				assert.Equal(t, []string{testcase.value}, tile.Metrics.Values, testcase.name)
			}
		}
	}
}

func TestMilestone_Errors(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetMilestones", "test", "error").Return(nil, errors.New("boom"))
	mockRepository.On("GetMilestones", "test", "none").Return([]models.Milestone{}, nil)
	gu := NewGiteaUsecase(mockRepository)

	_, err := gu.Milestone(&models.MilestoneParams{Owner: "test", Repository: "error"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load milestones", err.Error())
	}

	_, err = gu.Milestone(&models.MilestoneParams{Owner: "test", Repository: "none"})
	if assert.Error(t, err) {
		assert.Equal(t, "no open milestone", err.Error())
		assert.Equal(t, coreModels.UnknownStatus, err.(*coreModels.MonitororError).ErrorStatus)
	}
}

func TestNextMilestone(t *testing.T) {
	early := ToTime(time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC))
	late := ToTime(time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC))

	assert.Nil(t, nextMilestone(nil))
	assert.Equal(t, int64(3), nextMilestone([]models.Milestone{{ID: 1}, {ID: 2, DueOn: late}, {ID: 3, DueOn: early}}).ID)
	assert.Equal(t, int64(2), nextMilestone([]models.Milestone{{ID: 4}, {ID: 2}}).ID)
	assert.Equal(t, int64(1), nextMilestone([]models.Milestone{{ID: 5, DueOn: early}, {ID: 1, DueOn: early}}).ID)
}

func TestDaysUntil_DaylightSavingTime(t *testing.T) {
	rome, err := time.LoadLocation("Europe/Rome")
	if !assert.NoError(t, err) {
		return
	}
	// 25 October 2026 lasts 25 hours in Rome
	now := time.Date(2026, 10, 24, 23, 30, 0, 0, rome)
	assert.Equal(t, 2, daysUntil(now, time.Date(2026, 10, 26, 0, 30, 0, 0, rome)))
	// Due date stored in UTC, still the next day in Rome
	assert.Equal(t, 1, daysUntil(now, time.Date(2026, 10, 24, 22, 30, 0, 0, time.UTC)))
}

func TestMilestonesGenerator(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetRepositories", "org").Return([]models.Repository{
		{Name: "api"}, {Name: "web"}, {Name: "old", Archived: true}, {Name: "wiki"},
	}, nil)
	mockRepository.On("GetMilestones", "org", "api").Return([]models.Milestone{{ID: 1}}, nil)
	mockRepository.On("GetMilestones", "org", "web").Return([]models.Milestone{}, nil)
	mockRepository.On("GetMilestones", "org", "wiki").Return(nil, errors.New("issues disabled"))

	tiles, err := NewGiteaUsecase(mockRepository).MilestonesGenerator(&models.MilestoneGeneratorParams{Owner: "org"})
	if assert.NoError(t, err) {
		assert.Equal(t, []uiConfigModels.GeneratedTile{
			{Label: "api", Params: &models.MilestoneParams{Owner: "org", Repository: "api"}},
		}, tiles)
		mockRepository.AssertNotCalled(t, "GetMilestones", "org", "old")
	}
}

func TestMilestonesGenerator_Error(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetRepositories", "org").Return(nil, errors.New("boom"))

	tiles, err := NewGiteaUsecase(mockRepository).MilestonesGenerator(&models.MilestoneGeneratorParams{Owner: "org"})
	assert.Error(t, err)
	assert.Nil(t, tiles)
}
