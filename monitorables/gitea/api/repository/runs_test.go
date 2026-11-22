package repository

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	"github.com/stretchr/testify/assert"
)

func TestGiteaRepository_GetRuns_Success(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/repos/owner/repo/actions/runs", r.URL.EscapedPath())
		assert.Equal(t, "feature/test", r.URL.Query().Get("branch"))
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		_, _ = fmt.Fprint(w, `{"total_count":2,"workflow_runs":[
			{"id":11,"run_attempt":1,"run_number":8,"path":"long.yml@refs/heads/feature/test","head_sha":"abc","status":"queued",
			 "created_at":"2026-11-22T13:30:00Z","started_at":"1970-01-01T00:00:00Z","completed_at":"1970-01-01T00:00:00Z"},
			{"id":10,"run_attempt":2,"run_number":7,"path":"ci.yml@refs/heads/feature/test","head_sha":"abc","status":"completed","conclusion":"failure",
			 "created_at":"2026-11-22T13:20:00Z","started_at":"2026-11-22T13:21:00Z","completed_at":"2026-11-22T13:25:00Z"}
		]}`)
	})

	runs, err := repository.GetRuns("owner", "repo", "feature/test")
	if assert.NoError(t, err) {
		assert.Equal(t, []models.Run{
			{
				ID: 11, Attempt: 1, Number: 8, Workflow: "long.yml", HeadSHA: "abc", Status: "queued",
				CreatedAt: time.Date(2026, 11, 22, 13, 30, 0, 0, time.UTC),
			},
			{
				ID: 10, Attempt: 2, Number: 7, Workflow: "ci.yml", HeadSHA: "abc", Status: "completed", Conclusion: "failure",
				CreatedAt:   time.Date(2026, 11, 22, 13, 20, 0, 0, time.UTC),
				StartedAt:   time.Date(2026, 11, 22, 13, 21, 0, 0, time.UTC),
				CompletedAt: time.Date(2026, 11, 22, 13, 25, 0, 0, time.UTC),
			},
		}, runs)
	}
}

func TestGiteaRepository_GetRuns_NotFound(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	runs, err := repository.GetRuns("owner", "repo", "main")
	assert.Error(t, err)
	assert.Nil(t, runs)
}

func TestGiteaRepository_GetChecks_TargetURL(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `[{"id":1,"status":"pending","context":"ci / build (push)","target_url":"/owner/repo/actions/runs/10/jobs/14",
			"created_at":"2026-11-22T13:22:35Z","updated_at":"2026-11-22T13:22:35Z"}]`)
	})

	checks, err := repository.GetChecks("owner", "repo", "main")
	if assert.NoError(t, err) && assert.Len(t, checks.Statuses, 1) {
		assert.Equal(t, "/owner/repo/actions/runs/10/jobs/14", checks.Statuses[0].TargetURL)
	}
}
