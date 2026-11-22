package repository

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
)

func TestGiteaRepository_GetMilestones(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/repos/owner/my%20repo/milestones", r.URL.EscapedPath())
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		w.Header().Set("X-Total-Count", "2")
		_, _ = fmt.Fprint(w, `[
			{"id":1,"title":"v2.0","open_issues":3,"closed_issues":5,"due_on":"2026-11-28T17:00:00Z"},
			{"id":2,"title":"Backlog","open_issues":0,"closed_issues":0,"due_on":null}
		]`)
	})

	milestones, err := repository.GetMilestones("owner", "my repo")
	if assert.NoError(t, err) {
		assert.Equal(t, []models.Milestone{
			{ID: 1, Title: "v2.0", OpenIssues: 3, ClosedIssues: 5, DueOn: pointer.ToTime(time.Date(2026, 11, 28, 17, 0, 0, 0, time.UTC))},
			{ID: 2, Title: "Backlog"},
		}, milestones)
	}
}

func TestGiteaRepository_GetMilestones_Error(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	milestones, err := repository.GetMilestones("owner", "repo")
	assert.Error(t, err)
	assert.Nil(t, milestones)
}
