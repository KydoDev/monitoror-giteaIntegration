package repository

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/monitoror/monitoror/monitorables/github/api/models"
	"github.com/monitoror/monitoror/monitorables/github/config"

	"github.com/stretchr/testify/assert"
)

func initGraphqlRepository(t *testing.T, handler http.HandlerFunc) *githubRepository {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	conf := &config.Github{URL: server.URL, Token: "xxx", Timeout: 1000, CountCacheExpiration: 10000}

	return NewGithubRepository(conf).(*githubRepository)
}

func TestRepository_GetContributions_Success(t *testing.T) {
	from := time.Date(2026, time.November, 8, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.November, 17, 23, 0, 0, 0, time.UTC)

	repository := initGraphqlRepository(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/graphql", r.URL.Path)
		assert.Equal(t, "Bearer xxx", r.Header.Get("Authorization"))

		var body struct {
			Query     string            `json:"query"`
			Variables map[string]string `json:"variables"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Contains(t, body.Query, "contributionCalendar")
		assert.Equal(t, map[string]string{"login": "john", "from": "2026-11-08T00:00:00Z", "to": "2026-11-17T23:00:00Z"}, body.Variables)

		_, _ = fmt.Fprint(w, `{"data":{"user":{"contributionsCollection":{"contributionCalendar":{"weeks":[
			{"contributionDays":[{"date":"2026-11-08","contributionCount":1},{"date":"2026-11-09","contributionCount":0}]},
			{"contributionDays":[{"date":"2026-11-15","contributionCount":3}]}]}}}}}`)
	})

	contributions, err := repository.GetContributions("john", from, to)
	if assert.NoError(t, err) {
		assert.Equal(t, []models.Contribution{
			{Date: "2026-11-08", Count: 1},
			{Date: "2026-11-09", Count: 0},
			{Date: "2026-11-15", Count: 3},
		}, contributions)
	}
}

func TestRepository_GetContributions_Errors(t *testing.T) {
	for _, testcase := range []struct {
		status   int
		body     string
		expected string
	}{
		{http.StatusOK, `{"data":{"user":null},"errors":[{"type":"NOT_FOUND","message":"Could not resolve to a User with the login of 'nobody'."}]}`, "github graphql error: Could not resolve"},
		{http.StatusOK, `{"data":{"user":null}}`, `github user "nobody" not found`},
		{http.StatusOK, `{"data":`, "unexpected end of JSON input"},
		{http.StatusUnauthorized, `{"message":"Bad credentials"}`, "github graphql returned status 401: {\"message\":\"Bad credentials\"}"},
		{http.StatusBadGateway, strings.Repeat("é", 300), "..."},
	} {
		repository := initGraphqlRepository(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(testcase.status)
			_, _ = fmt.Fprint(w, testcase.body)
		})

		_, err := repository.GetContributions("nobody", time.Now(), time.Now())
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), testcase.expected)
		}
	}
}

func TestGraphqlURL(t *testing.T) {
	assert.Equal(t, "https://api.github.com/graphql", graphqlURL("https://api.github.com/"))
	assert.Equal(t, "https://github.example.com/api/graphql", graphqlURL("https://github.example.com/api/v3/"))
}

func TestRepository_GetContributions_CrossHostRedirectKeepsToken(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the other host must not be called (Authorization: %q)", r.Header.Get("Authorization"))
	}))
	t.Cleanup(other.Close)

	repository := initGraphqlRepository(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/graphql", http.StatusTemporaryRedirect)
	})

	_, err := repository.GetContributions("john", time.Now(), time.Now())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "status 307")
	}
}

func TestRepository_GetContributions_SameHostRedirect(t *testing.T) {
	repository := initGraphqlRepository(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/graphql" {
			http.Redirect(w, r, "/moved/graphql", http.StatusTemporaryRedirect)
			return
		}
		assert.Equal(t, "Bearer xxx", r.Header.Get("Authorization"))
		_, _ = fmt.Fprint(w, `{"data":{"user":{"contributionsCollection":{"contributionCalendar":{"weeks":[]}}}}}`)
	})

	contributions, err := repository.GetContributions("john", time.Now(), time.Now())
	assert.NoError(t, err)
	assert.Empty(t, contributions)
}

func TestSameHostRedirect_TooManyRedirects(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://api.github.com/a", nil)
	via := make([]*http.Request, maxRedirects)
	for i := range via {
		via[i] = request
	}

	assert.EqualError(t, sameHostRedirect(request, via), "stopped after 10 redirects")
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "short", truncate("short"))
	assert.Equal(t, maxGraphqlErrorLength+3, len(truncate(strings.Repeat("a", 500))))
}
