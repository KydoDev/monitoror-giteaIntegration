package repository

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	"github.com/stretchr/testify/assert"
)

func TestGiteaRepository_SearchIssues(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/repos/issues/search", r.URL.Path)
		assert.Equal(t, "acme", r.URL.Query().Get("owner"))
		assert.Equal(t, "issues", r.URL.Query().Get("type"))
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		w.Header().Set("X-Total-Count", "1")
		_, _ = fmt.Fprint(w, `[{"number":3,"title":"Crash","created_at":"2026-11-20T10:00:00Z",
			"labels":[{"name":"bug"},{"name":"ui"}],"repository":{"name":"mobile","owner":"acme"}}]`)
	})

	issues, err := repository.SearchIssues("acme", "issues")
	if assert.NoError(t, err) {
		assert.Equal(t, []models.Issue{{
			Number: 3, Title: "Crash", Owner: "acme", Repository: "mobile", Labels: []string{"bug", "ui"},
			CreatedAt: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
		}}, issues)
	}
}

func TestGiteaRepository_GetRepositoryIssues(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/repos/acme/mobile/issues", r.URL.Path)
		assert.Equal(t, "issues", r.URL.Query().Get("type"))
		w.Header().Set("X-Total-Count", "1")
		_, _ = fmt.Fprint(w, `[{"number":1,"title":"Crash","labels":[{"name":"Bug"}]}]`)
	})

	issues, err := repository.GetRepositoryIssues("acme", "mobile")
	if assert.NoError(t, err) && assert.Len(t, issues, 1) {
		assert.Equal(t, "mobile", issues[0].Repository)
		assert.Equal(t, []string{"Bug"}, issues[0].Labels)
	}
}

func TestGiteaRepository_GetOwnerRuns(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/orgs/acme/actions/runs", r.URL.Path)
		_, _ = fmt.Fprint(w, `{"workflow_runs":[{"id":3,"path":"ci.yml@refs/heads/main","status":"in_progress",
			"display_title":"feat: login","repository":{"name":"shop"},"started_at":"2026-11-22T10:00:00Z"}]}`)
	})

	runs, err := repository.GetOwnerRuns("acme")
	if assert.NoError(t, err) && assert.Len(t, runs, 1) {
		assert.Equal(t, "shop", runs[0].Repository)
		assert.Equal(t, "feat: login", runs[0].Title)
		assert.Equal(t, "ci.yml", runs[0].Workflow)
	}
}

func TestGiteaRepository_GetRuns_WithoutBranch(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, r.URL.Query().Has("branch"))
		_, _ = fmt.Fprint(w, `{"workflow_runs":[]}`)
	})

	runs, err := repository.GetRuns("acme", "shop", "")
	assert.NoError(t, err)
	assert.Empty(t, runs)
}

func TestGiteaRepository_GetCommits_StopsAtOldPage(t *testing.T) {
	defer func(size int) { PageSize = size }(PageSize)
	PageSize = 2
	since := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	pages := 0
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		pages++
		assert.Equal(t, "/api/v1/repos/acme/shop/commits", r.URL.Path)
		assert.Equal(t, "main", r.URL.Query().Get("sha"))
		assert.Equal(t, "false", r.URL.Query().Get("stat"))
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = fmt.Fprint(w, `[{"sha":"a","commit":{"committer":{"date":"2026-11-20T10:00:00Z"}}},
				{"sha":"b","commit":{"committer":{"date":"2026-11-02T10:00:00Z"}}}]`)
		case "2":
			_, _ = fmt.Fprint(w, `[{"sha":"c","commit":{"committer":{"date":"2026-10-20T10:00:00Z"}}},
				{"sha":"d","commit":{"committer":{"date":"2026-10-10T10:00:00Z"}}}]`)
		default:
			t.Error("page 3 must not be requested")
		}
	})

	commits, err := repository.GetCommits("acme", "shop", "main", since)
	if assert.NoError(t, err) {
		assert.Len(t, commits, 4)
		assert.Equal(t, 2, pages)
	}
}

func TestGiteaRepository_GetFeedCommits(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/orgs/alice/activities/feeds" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		assert.Equal(t, "/api/v1/users/alice/activities/feeds", r.URL.Path)
		_, _ = fmt.Fprint(w, `[
			{"op_type":"create_repo","repo":{"name":"x"}},
			{"op_type":"commit_repo","ref_name":"refs/heads/main","created":"2026-11-22T10:00:00Z","repo":{"name":"shop"},
			 "act_user":{"login":"alice"},"content":"{\"Commits\":[{\"Message\":\"feat: a\\nbody\",\"AuthorName\":\"Alice\"},{\"Message\":\"fix: b\",\"AuthorName\":\"\"}]}"},
			{"op_type":"commit_repo","repo":{"name":"shop"},"content":"not json"}
		]`)
	})

	commits, err := repository.GetFeedCommits("alice", 10)
	if assert.NoError(t, err) {
		pushedAt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
		assert.Equal(t, []models.FeedCommit{
			{Repository: "shop", Branch: "main", Author: "Alice", Message: "feat: a", PushedAt: pushedAt},
			{Repository: "shop", Branch: "main", Author: "alice", Message: "fix: b", PushedAt: pushedAt},
		}, commits)
	}

	commits, err = repository.GetFeedCommits("alice", 1)
	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
	}
}

func TestGiteaRepository_GetFeedCommits_Error(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := repository.GetFeedCommits("nobody", 10)
	assert.Error(t, err)
}

func TestGiteaRepository_GetOrganizationMembers(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/orgs/acme/members", r.URL.Path)
		w.Header().Set("X-Total-Count", "2")
		_, _ = fmt.Fprint(w, `[{"login":"alice"},{"login":"bob"}]`)
	})

	members, err := repository.GetOrganizationMembers("acme")
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"alice", "bob"}, members)
	}
}

func TestGiteaRepository_TeamBoardErrors(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := repository.SearchIssues("acme", "pulls")
	assert.Error(t, err)
	_, err = repository.GetRepositoryIssues("acme", "shop")
	assert.Error(t, err)
	_, err = repository.GetOwnerRuns("acme")
	assert.Error(t, err)
	_, err = repository.GetCommits("acme", "shop", "main", time.Now())
	assert.Error(t, err)
	_, err = repository.GetOrganizationMembers("acme")
	assert.Error(t, err)
}
