package repository

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	"github.com/stretchr/testify/assert"
)

func TestGiteaRepository_GetCount_EveryRepository(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/repos/issues/search", r.URL.Path)
		assert.Equal(t, "my org", r.URL.Query().Get("owner"))
		assert.Equal(t, "pulls", r.URL.Query().Get("type"))
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		assert.Equal(t, "1", r.URL.Query().Get("limit"))
		w.Header().Set("X-Total-Count", "7")
		_, _ = fmt.Fprint(w, `[]`)
	})

	count, err := repository.GetCount(&models.CountParams{Owner: "my org", Type: "pulls"})
	if assert.NoError(t, err) {
		assert.Equal(t, 7, count)
	}
}

func TestGiteaRepository_SearchPullRequests(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/repos/issues/search", r.URL.Path)
		assert.Equal(t, "org", r.URL.Query().Get("owner"))
		assert.Equal(t, "pulls", r.URL.Query().Get("type"))
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		w.Header().Set("X-Total-Count", "2")
		_, _ = fmt.Fprint(w, `[
			{"number":4,"title":"a","repository":{"id":1,"name":"api","owner":"org","full_name":"org/api"}},
			{"number":2,"title":"b","repository":{"id":2,"name":"web","owner":"org","full_name":"org/web"}}
		]`)
	})

	pullRequests, err := repository.SearchPullRequests("org")
	if assert.NoError(t, err) {
		assert.Equal(t, []models.PullRequest{
			{ID: 4, Owner: "org", Repository: "api"},
			{ID: 2, Owner: "org", Repository: "web"},
		}, pullRequests)
	}
}

func TestGiteaRepository_SearchPullRequests_Error(t *testing.T) {
	repository := initRepository(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	pullRequests, err := repository.SearchPullRequests("unknown")
	assert.Error(t, err)
	assert.Nil(t, pullRequests)
}
