package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/monitoror/monitoror/monitorables/github/api"
	"github.com/monitoror/monitoror/monitorables/github/api/models"
	"github.com/monitoror/monitoror/monitorables/github/config"
	"github.com/monitoror/monitoror/pkg/gogithub"
	"github.com/monitoror/monitoror/pkg/gravatar"

	githubApi "github.com/google/go-github/github"
	"github.com/sourcegraph/httpcache"
	"golang.org/x/oauth2"
)

type (
	githubRepository struct {
		searchService       gogithub.SearchService
		checksService       gogithub.ChecksService
		repositoriesService gogithub.RepositoriesService
		pullRequestService  gogithub.PullRequestService
		gitService          gogithub.GitService

		// GraphQL client, used for the contributions calendar (not available in the REST API)
		httpClient *http.Client
		graphqlURL string

		config *config.Github
	}

	contributionsResponse struct {
		Data struct {
			User *struct {
				ContributionsCollection struct {
					ContributionCalendar struct {
						Weeks []struct {
							ContributionDays []struct {
								Date              string `json:"date"`
								ContributionCount int    `json:"contributionCount"`
							} `json:"contributionDays"`
						} `json:"weeks"`
					} `json:"contributionCalendar"`
				} `json:"contributionsCollection"`
			} `json:"user"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
)

const (
	contributionsQuery = `query($login: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    contributionsCollection(from: $from, to: $to) {
      contributionCalendar { weeks { contributionDays { date contributionCount } } }
    }
  }
}`
	maxGraphqlBodySize    = 1 << 20 // 1 MiB
	maxGraphqlErrorLength = 200
)

func NewGithubRepository(config *config.Github) api.Repository {
	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			// Use NewMemoryCacheTransport to save github rate limit
			Base:   httpcache.NewMemoryCacheTransport(),
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.Token}),
		},
		Timeout: time.Duration(config.Timeout) * time.Millisecond,
		// The oauth2 transport adds the token to every request, redirects included:
		// only follow redirects to the same host, so the token never reaches another host
		CheckRedirect: sameHostRedirect,
	}

	// Init Github Client
	client := githubApi.NewClient(httpClient)

	// Add / if missing
	if !strings.HasSuffix(config.URL, "/") {
		config.URL += "/"
	}

	// Set baseURL
	baseURL, _ := url.Parse(config.URL) // Err check already done in Monitorable.Validate
	client.BaseURL = baseURL
	client.UploadURL = nil

	return &githubRepository{
		searchService:       client.Search,
		checksService:       client.Checks,
		repositoriesService: client.Repositories,
		pullRequestService:  client.PullRequests,
		gitService:          client.Git,
		httpClient:          httpClient,
		graphqlURL:          graphqlURL(config.URL),
		config:              config,
	}
}

func (gr *githubRepository) GetCount(query string) (int, error) {
	issuesResult, _, err := gr.searchService.Issues(context.TODO(), query, nil)
	if err != nil {
		return 0, err
	}

	return issuesResult.GetTotal(), err
}

func (gr *githubRepository) GetChecks(owner, repository, ref string) (*models.Checks, error) {
	checks := &models.Checks{Runs: []models.Run{}, Statuses: []models.Status{}}

	checkRuns, _, err := gr.checksService.ListCheckRunsForRef(context.TODO(), owner, repository, ref, nil)
	if err != nil {
		return nil, err
	}

	for _, checkRun := range checkRuns.CheckRuns {
		if checkRun.Name != nil && checkRun.Status != nil {
			run := models.Run{
				ID:         checkRun.GetID(),
				Title:      checkRun.GetName(),
				Status:     checkRun.GetStatus(),
				Conclusion: checkRun.GetConclusion(),
			}

			if checkRun.StartedAt != nil {
				run.StartedAt = &checkRun.StartedAt.Time
			}

			if checkRun.CompletedAt != nil {
				run.CompletedAt = &checkRun.CompletedAt.Time
			}

			checks.HeadCommit = checkRun.HeadSHA
			checks.Runs = append(checks.Runs, run)
		}
	}

	repoStatuses, _, err := gr.repositoriesService.ListStatuses(context.TODO(), owner, repository, ref, nil)
	if err != nil {
		return nil, err
	}

	for _, repoStatus := range repoStatuses {
		if repoStatus.Context != nil && repoStatus.State != nil && repoStatus.CreatedAt != nil && repoStatus.UpdatedAt != nil {
			status := models.Status{
				ID:        repoStatus.GetID(),
				Title:     repoStatus.GetContext(),
				State:     repoStatus.GetState(),
				CreatedAt: repoStatus.GetCreatedAt(),
				UpdatedAt: repoStatus.GetUpdatedAt(),
			}

			if repoStatus.URL != nil {
				headCommit := repoStatus.GetURL()
				headCommit = headCommit[strings.LastIndex(headCommit, "/")+1:]

				checks.HeadCommit = &headCommit
			}

			checks.Statuses = append(checks.Statuses, status)
		}
	}

	return checks, nil
}

func (gr *githubRepository) GetPullRequest(owner, repository string, id int) (*models.PullRequest, error) {
	pullRequest, _, err := gr.pullRequestService.Get(context.TODO(), owner, repository, id)
	if err != nil {
		return nil, err
	}

	return fillPullRequest(pullRequest), nil
}

func (gr *githubRepository) GetPullRequests(owner, repository string) ([]models.PullRequest, error) {
	pullRequests, _, err := gr.pullRequestService.List(context.TODO(), owner, repository, nil)
	if err != nil {
		return nil, err
	}

	var result []models.PullRequest
	for _, pullRequest := range pullRequests {
		result = append(result, *fillPullRequest(pullRequest))
	}

	return result, nil
}

func fillPullRequest(pr *githubApi.PullRequest) *models.PullRequest {
	pullRequest := &models.PullRequest{
		ID:               pr.GetNumber(),
		Title:            pr.GetTitle(),
		SourceOwner:      pr.GetHead().GetUser().GetLogin(),
		SourceRepository: pr.GetHead().GetRepo().GetName(),
		SourceBranch:     pr.GetHead().GetRef(),
		CommitSHA:        pr.GetHead().GetSHA(),
	}

	if pr.GetUser() != nil {
		pullRequest.Author.Name = pr.GetUser().GetName()
		pullRequest.Author.AvatarURL = pr.GetUser().GetAvatarURL()

		if pullRequest.Author.Name == "" {
			pullRequest.Author.Name = pr.GetUser().GetLogin()
		}
	}

	return pullRequest
}

func (gr *githubRepository) GetCommit(owner, repository, sha string) (*models.Commit, error) {
	commit, _, err := gr.gitService.GetCommit(context.TODO(), owner, repository, sha)
	if err != nil {
		return nil, err
	}

	result := &models.Commit{
		SHA: sha,
	}

	if commit.GetAuthor() != nil {
		result.Author.Name = commit.GetAuthor().GetName()
		result.Author.AvatarURL = gravatar.GetGravatarURL(commit.GetAuthor().GetEmail())

		if result.Author.Name == "" {
			result.Author.Name = commit.GetAuthor().GetLogin()
		}
	}

	return result, nil
}

// GetContributions loads the contributions calendar of a user between from and to (one year at most).
func (gr *githubRepository) GetContributions(user string, from, to time.Time) ([]models.Contribution, error) {
	body, err := json.Marshal(map[string]interface{}{
		"query": contributionsQuery,
		"variables": map[string]string{
			"login": user,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, gr.graphqlURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := gr.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, maxGraphqlBodySize))
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github graphql returned status %d: %s", response.StatusCode, truncate(strings.TrimSpace(string(data))))
	}

	result := &contributionsResponse{}
	if err = json.Unmarshal(data, result); err != nil {
		return nil, err
	}

	// GraphQL reports most errors with a 200 status
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("github graphql error: %s", truncate(result.Errors[0].Message))
	}
	if result.Data.User == nil {
		return nil, fmt.Errorf("github user %q not found", user)
	}

	var contributions []models.Contribution
	for _, week := range result.Data.User.ContributionsCollection.ContributionCalendar.Weeks {
		for _, day := range week.ContributionDays {
			contributions = append(contributions, models.Contribution{Date: day.Date, Count: day.ContributionCount})
		}
	}

	return contributions, nil
}

// graphqlURL returns the GraphQL endpoint of a GitHub REST API URL:
// https://api.github.com/ -> https://api.github.com/graphql
// https://github.example.com/api/v3/ (GitHub Enterprise Server) -> https://github.example.com/api/graphql
func graphqlURL(restURL string) string {
	if strings.HasSuffix(restURL, "/api/v3/") {
		return strings.TrimSuffix(restURL, "v3/") + "graphql"
	}

	return restURL + "graphql"
}

const maxRedirects = 10

// sameHostRedirect follows redirects on the same scheme and host only (e.g. renamed repositories).
// A redirect to another host returns the redirect response instead of following it.
func sameHostRedirect(request *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("stopped after 10 redirects")
	}
	if request.URL.Host != via[0].URL.Host || request.URL.Scheme != via[0].URL.Scheme {
		return http.ErrUseLastResponse
	}

	return nil
}

func truncate(message string) string {
	if len(message) <= maxGraphqlErrorLength {
		return message
	}

	return strings.ToValidUTF8(message[:maxGraphqlErrorLength], "") + "..."
}
