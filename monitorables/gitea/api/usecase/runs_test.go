package usecase

import (
	"testing"
	"time"

	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api/mocks"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	. "github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
)

func TestChecks_ManualRunRunning(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	runs := []models.Run{
		{ID: 9, Attempt: 1, Number: 7, Workflow: "long.yml", HeadSHA: "abc", Status: "in_progress", CreatedAt: now.Add(-time.Minute), StartedAt: now.Add(-time.Minute)},
		{ID: 6, Attempt: 1, Number: 5, Workflow: "long.yml", HeadSHA: "abc", Status: "completed", Conclusion: "success",
			CreatedAt: now.Add(-time.Hour), StartedAt: now.Add(-time.Hour), CompletedAt: now.Add(-time.Hour + 10*time.Minute)},
		{ID: 4, Attempt: 1, Number: 3, Workflow: "long.yml", HeadSHA: "abc", Status: "completed", Conclusion: "cancelled",
			CreatedAt: now.Add(-2 * time.Hour), StartedAt: now.Add(-2 * time.Hour), CompletedAt: now.Add(-2*time.Hour + time.Minute)},
	}

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetChecks", "test", "test", "main").Return(&models.Checks{Statuses: []models.Status{}}, nil)
	mockRepository.On("GetRuns", "test", "test", "main").Return(runs, nil)

	gu := NewGiteaUsecase(mockRepository)

	tile, err := gu.Checks(&models.ChecksParams{Owner: "test", Repository: "test", Ref: "main"})
	if assert.NoError(t, err) {
		assert.Equal(t, coreModels.RunningStatus, tile.Status)
		assert.Equal(t, ToString("7"), tile.Build.ID)
		assert.Equal(t, now.Add(-time.Minute), *tile.Build.StartedAt)
		assert.Nil(t, tile.Build.FinishedAt)
		// No history in cache: the estimate comes from the previous successful run
		assert.Equal(t, ToInt64(600), tile.Build.EstimatedDuration)
		mockRepository.AssertExpectations(t)
	}
}

func TestChecks_RunsReplaceActionsStatuses(t *testing.T) {
	refTime := time.Now().Add(-time.Hour)
	checks := &models.Checks{
		HeadCommit: ToString("abc"),
		Statuses: []models.Status{
			// Written by run 10 while it was running: superseded by the run itself
			{ID: 1, Title: "ci / build (push)", State: "pending", TargetURL: "/test/test/actions/runs/10/jobs/14", CreatedAt: refTime, UpdatedAt: refTime},
			// Another CI stays
			{ID: 2, Title: "drone", State: "success", TargetURL: "https://drone/test/1", CreatedAt: refTime, UpdatedAt: refTime},
		},
	}
	runs := []models.Run{
		// Newer run on another commit: not part of this build
		{ID: 12, Attempt: 1, Number: 9, Workflow: "other.yml", HeadSHA: "def", Status: "completed", Conclusion: "failure",
			CreatedAt: refTime, StartedAt: refTime.Add(-24 * time.Hour), CompletedAt: refTime},
		{ID: 10, Attempt: 1, Number: 7, Workflow: "ci.yml", HeadSHA: "abc", Status: "completed", Conclusion: "success",
			CreatedAt: refTime, StartedAt: refTime, CompletedAt: refTime.Add(5 * time.Minute)},
	}

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetChecks", "test", "test", "main").Return(checks, nil)
	mockRepository.On("GetRuns", "test", "test", "main").Return(runs, nil)

	gu := NewGiteaUsecase(mockRepository)

	tile, err := gu.Checks(&models.ChecksParams{Owner: "test", Repository: "test", Ref: "main"})
	if assert.NoError(t, err) {
		assert.Equal(t, coreModels.SuccessStatus, tile.Status)
		assert.Equal(t, ToString("7"), tile.Build.ID)
		assert.Equal(t, refTime, *tile.Build.StartedAt)
		assert.Equal(t, refTime.Add(5*time.Minute), *tile.Build.FinishedAt)
		mockRepository.AssertExpectations(t)
	}
}

func TestChecks_RunFailureAuthor(t *testing.T) {
	refTime := time.Now().Add(-time.Hour)
	runs := []models.Run{
		{ID: 5, Attempt: 1, Number: 4, Workflow: "checks.yml", HeadSHA: "abc", Status: "completed", Conclusion: "failure",
			CreatedAt: refTime, StartedAt: refTime, CompletedAt: refTime.Add(time.Minute)},
	}

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetChecks", "test", "test", "main").Return(&models.Checks{Statuses: []models.Status{}}, nil)
	mockRepository.On("GetRuns", "test", "test", "main").Return(runs, nil)
	mockRepository.On("GetCommit", "test", "test", "abc").
		Return(&models.Commit{SHA: "abc", Author: coreModels.Author{Name: "me", AvatarURL: "avatar"}}, nil)

	gu := NewGiteaUsecase(mockRepository)

	tile, err := gu.Checks(&models.ChecksParams{Owner: "test", Repository: "test", Ref: "main"})
	if assert.NoError(t, err) {
		assert.Equal(t, coreModels.FailedStatus, tile.Status)
		assert.Equal(t, &coreModels.Author{Name: "me", AvatarURL: "avatar"}, tile.Build.Author)
		mockRepository.AssertExpectations(t)
	}
}

func TestParseRun(t *testing.T) {
	for _, testcase := range []struct {
		status, conclusion string
		expected           coreModels.TileStatus
	}{
		{"completed", "success", coreModels.SuccessStatus},
		{"completed", "failure", coreModels.FailedStatus},
		{"completed", "cancelled", coreModels.CanceledStatus},
		{"completed", "skipped", coreModels.DisabledStatus},
		{"completed", "unexpected", coreModels.UnknownStatus},
		{"in_progress", "", coreModels.RunningStatus},
		{"queued", "", coreModels.QueuedStatus},
		{"waiting", "", coreModels.QueuedStatus},
		{"unexpected", "", coreModels.UnknownStatus},
	} {
		assert.Equal(t, testcase.expected, parseRun(&models.Run{Status: testcase.status, Conclusion: testcase.conclusion}), testcase)
	}
}

func TestConvertChecks_QueuedRunUsesCreatedAt(t *testing.T) {
	createdAt := time.Date(2026, 11, 22, 13, 30, 0, 0, time.UTC)
	statuses, startedAt, finishedAt, _ := convertChecks(&models.Checks{},
		[]models.Run{{ID: 11, Attempt: 1, Workflow: "long.yml", Status: "queued", CreatedAt: createdAt}})

	assert.Equal(t, []coreModels.TileStatus{coreModels.QueuedStatus}, statuses)
	assert.Equal(t, createdAt, *startedAt)
	assert.Equal(t, createdAt, *finishedAt)
}

func TestConvertChecks_RerunChangesBuildID(t *testing.T) {
	run := models.Run{ID: 10, Attempt: 1, Workflow: "ci.yml", Status: "completed", Conclusion: "success"}
	_, _, _, firstAttempt := convertChecks(&models.Checks{}, []models.Run{run})
	run.Attempt = 2
	_, _, _, secondAttempt := convertChecks(&models.Checks{}, []models.Run{run})

	assert.NotEqual(t, firstAttempt, secondAttempt)
}

func TestSelectBuildRuns_KeepsUnmatchedActionsStatuses(t *testing.T) {
	checks := &models.Checks{
		HeadCommit: ToString("abc"),
		Statuses:   []models.Status{{ID: 1, Title: "ci", State: "success", TargetURL: "/o/r/actions/runs/3/jobs/1"}},
	}
	runs := []models.Run{{ID: 10, Workflow: "ci.yml", HeadSHA: "abc", Status: "in_progress"}}

	assert.Len(t, selectBuildRuns(checks, runs), 1)
	assert.Len(t, checks.Statuses, 1)

	// No run on the head commit: statuses are left untouched
	assert.Empty(t, selectBuildRuns(checks, []models.Run{{ID: 3, Workflow: "ci.yml", HeadSHA: "old"}}))
	assert.Len(t, checks.Statuses, 1)
}

func TestActionsRunID(t *testing.T) {
	assert.Equal(t, "10", actionsRunID("/owner/repo/actions/runs/10/jobs/14"))
	assert.Equal(t, "10", actionsRunID("https://gitea/owner/repo/actions/runs/10"))
	assert.Equal(t, "", actionsRunID("https://drone/owner/repo/1"))
	assert.Equal(t, "", actionsRunID(""))
}

func TestEstimateDurationFromRuns(t *testing.T) {
	refTime := time.Now()
	finished := func(id int64, workflow, conclusion string, duration time.Duration) models.Run {
		return models.Run{ID: id, Workflow: workflow, Status: "completed", Conclusion: conclusion, StartedAt: refTime, CompletedAt: refTime.Add(duration)}
	}
	runs := []models.Run{
		{ID: 20, Workflow: "a.yml", Status: "in_progress", StartedAt: refTime},
		{ID: 19, Workflow: "b.yml", Status: "in_progress", StartedAt: refTime},
		finished(18, "a.yml", "success", 2*time.Minute),
		finished(17, "a.yml", "success", 4*time.Minute),
		finished(16, "a.yml", "failure", time.Hour),
		finished(15, "b.yml", "success", 5*time.Minute),
	}

	// Longest workflow average wins: a.yml = 3m, b.yml = 5m
	assert.Equal(t, 5*time.Minute, *estimateDurationFromRuns(runs, runs[:2]))
	// No successful history
	assert.Nil(t, estimateDurationFromRuns(runs[:2], runs[:2]))
}
