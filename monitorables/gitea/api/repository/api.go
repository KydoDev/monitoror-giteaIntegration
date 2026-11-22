package repository

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/monitoror/monitoror/monitorables/gitea/api"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"
	"github.com/monitoror/monitoror/monitorables/gitea/config"
	"github.com/monitoror/monitoror/pkg/gravatar"
)

type (
	giteaRepository struct {
		client *http.Client
		config *config.Gitea
	}

	workflowRun struct {
		ID          int64     `json:"id"`
		RunAttempt  int64     `json:"run_attempt"`
		RunNumber   int64     `json:"run_number"`
		Path        string    `json:"path"` // {workflow file}@{ref}
		HeadSHA     string    `json:"head_sha"`
		Status      string    `json:"status"`
		Conclusion  string    `json:"conclusion"`
		CreatedAt   time.Time `json:"created_at"`
		StartedAt   time.Time `json:"started_at"`
		CompletedAt time.Time `json:"completed_at"`
	}

	workflowRuns struct {
		WorkflowRuns []workflowRun `json:"workflow_runs"`
	}

	user struct {
		Login     string `json:"login"`
		FullName  string `json:"full_name"`
		AvatarURL string `json:"avatar_url"`
	}

	commitStatus struct {
		ID        int64     `json:"id"`
		Status    string    `json:"status"`
		Context   string    `json:"context"`
		URL       string    `json:"url"` // .../repos/{owner}/{repo}/statuses/{sha}
		TargetURL string    `json:"target_url"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	repository struct {
		Name  string `json:"name"`
		Owner *user  `json:"owner"`
	}

	repositoryDetails struct {
		Name          string `json:"name"`
		DefaultBranch string `json:"default_branch"`
		Archived      bool   `json:"archived"`
		Empty         bool   `json:"empty"`
	}

	pullRequestBranch struct {
		Ref  string      `json:"ref"`
		SHA  string      `json:"sha"`
		Repo *repository `json:"repo"` // nil when the fork has been deleted
	}

	pullRequest struct {
		Number int               `json:"number"`
		Title  string            `json:"title"`
		User   *user             `json:"user"`
		Head   pullRequestBranch `json:"head"`
	}

	commit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Author struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		} `json:"commit"`
		Author *user `json:"author"` // nil when the commit email is not linked to a Gitea user
	}

	heatmapItem struct {
		Timestamp     int64 `json:"timestamp"`
		Contributions int   `json:"contributions"`
	}
)

// PageSize is the number of items requested per page. 50 is Gitea's default MAX_RESPONSE_ITEMS.
var PageSize = 50

const (
	apiPath            = "/api/v1"
	maxErrorBodyLength = 200
	maxBodySize        = 10 << 20 // 10 MiB
	maxPages           = 100
	totalCountHeader   = "X-Total-Count"
)

func NewGiteaRepository(config *config.Gitea) api.Repository {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !config.SSLVerify},
	}
	client := &http.Client{Transport: tr, Timeout: time.Duration(config.Timeout) * time.Millisecond}

	// Remove last /
	config.URL = strings.TrimRight(config.URL, "/")

	return &giteaRepository{client, config}
}

func (r *giteaRepository) GetCount(params *models.CountParams) (int, error) {
	query := url.Values{}
	query.Set("type", defaultString(params.Type, "issues"))
	query.Set("state", defaultString(params.State, "open"))
	if params.Labels != "" {
		query.Set("labels", params.Labels)
	}
	if params.Query != "" {
		query.Set("q", params.Query)
	}
	// Only the total count header is used
	query.Set("limit", "1")

	header, err := r.get(fmt.Sprintf("/repos/%s/%s/issues", url.PathEscape(params.Owner), url.PathEscape(params.Repository)), query, nil)
	if err != nil {
		return 0, err
	}

	rawCount := header.Get(totalCountHeader)
	if rawCount == "" {
		return 0, fmt.Errorf("gitea response is missing the %s header", totalCountHeader)
	}

	count, err := strconv.Atoi(rawCount)
	if err != nil {
		return 0, fmt.Errorf("invalid %s header: %q", totalCountHeader, rawCount)
	}

	return count, nil
}

// GetChecks loads the commit status history of ref (one row per state change, most recent first).
// The history is used instead of the combined status: the combined status only keeps the last row
// of each context, which hides when a build started and makes every duration zero.
// Only the most recent page is loaded, which is enough to cover the current build.
func (r *giteaRepository) GetChecks(owner, repository, ref string) (*models.Checks, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(PageSize))

	var result []commitStatus
	path := fmt.Sprintf("/repos/%s/%s/commits/%s/statuses", url.PathEscape(owner), url.PathEscape(repository), url.PathEscape(ref))
	if _, err := r.get(path, query, &result); err != nil {
		return nil, err
	}

	checks := &models.Checks{Statuses: []models.Status{}}
	for _, s := range result {
		if s.Context == "" || s.Status == "" {
			continue
		}

		if checks.HeadCommit == nil && s.URL != "" {
			sha := s.URL[strings.LastIndex(s.URL, "/")+1:]
			checks.HeadCommit = &sha
		}

		checks.Statuses = append(checks.Statuses, models.Status{
			ID:        s.ID,
			Title:     s.Context,
			State:     s.Status,
			TargetURL: s.TargetURL,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}

	return checks, nil
}

// GetRuns returns the latest Gitea Actions runs of a branch, newest first.
// It fails on Gitea versions without this endpoint and when Actions are disabled.
func (r *giteaRepository) GetRuns(owner, repository, branch string) ([]models.Run, error) {
	query := url.Values{}
	query.Set("branch", branch)
	query.Set("limit", strconv.Itoa(PageSize))

	var result workflowRuns
	path := fmt.Sprintf("/repos/%s/%s/actions/runs", url.PathEscape(owner), url.PathEscape(repository))
	if _, err := r.get(path, query, &result); err != nil {
		return nil, err
	}

	runs := make([]models.Run, 0, len(result.WorkflowRuns))
	for _, run := range result.WorkflowRuns {
		workflow := run.Path
		if i := strings.LastIndex(workflow, "@"); i >= 0 {
			workflow = workflow[:i]
		}

		runs = append(runs, models.Run{
			ID:          run.ID,
			Attempt:     run.RunAttempt,
			Number:      run.RunNumber,
			Workflow:    workflow,
			HeadSHA:     run.HeadSHA,
			Status:      run.Status,
			Conclusion:  run.Conclusion,
			CreatedAt:   run.CreatedAt,
			StartedAt:   validTime(run.StartedAt),
			CompletedAt: validTime(run.CompletedAt),
		})
	}

	return runs, nil
}

func (r *giteaRepository) GetPullRequest(owner, repository string, id int) (*models.PullRequest, error) {
	result := &pullRequest{}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", url.PathEscape(owner), url.PathEscape(repository), id)
	if _, err := r.get(path, nil, result); err != nil {
		return nil, err
	}

	return toPullRequest(result), nil
}

func (r *giteaRepository) GetPullRequests(owner, repository string) ([]models.PullRequest, error) {
	query := url.Values{}
	query.Set("state", "open")

	pullRequests, err := getAllPages[pullRequest](r, fmt.Sprintf("/repos/%s/%s/pulls", url.PathEscape(owner), url.PathEscape(repository)), query)
	if err != nil {
		return nil, err
	}

	var results []models.PullRequest
	for i := range pullRequests {
		results = append(results, *toPullRequest(&pullRequests[i]))
	}

	return results, nil
}

// GetRepositories loads the repositories of a user or an organization (organizations are users in this endpoint).
func (r *giteaRepository) GetRepositories(owner string) ([]models.Repository, error) {
	repositories, err := getAllPages[repositoryDetails](r, fmt.Sprintf("/users/%s/repos", url.PathEscape(owner)), url.Values{})
	if err != nil {
		return nil, err
	}

	var results []models.Repository
	for _, repository := range repositories {
		results = append(results, models.Repository{
			Name:          repository.Name,
			DefaultBranch: repository.DefaultBranch,
			Archived:      repository.Archived,
			Empty:         repository.Empty,
		})
	}

	return results, nil
}

// getAllPages loads every page of a list endpoint (at most maxPages).
func getAllPages[T any](r *giteaRepository, path string, query url.Values) ([]T, error) {
	var results []T

	for page := 1; page <= maxPages; page++ {
		query.Set("page", strconv.Itoa(page))
		query.Set("limit", strconv.Itoa(PageSize))

		var items []T
		header, err := r.get(path, query, &items)
		if err != nil {
			return nil, err
		}

		if len(items) == 0 {
			return results, nil
		}
		results = append(results, items...)

		// Gitea may return fewer items than requested (MAX_RESPONSE_ITEMS), so rely on the total when available
		if total, err := strconv.Atoi(header.Get(totalCountHeader)); err == nil {
			if len(results) >= total {
				return results, nil
			}
		} else if len(items) < PageSize {
			return results, nil
		}
	}

	return results, nil
}

func (r *giteaRepository) GetCommit(owner, repository, sha string) (*models.Commit, error) {
	result := &commit{}
	path := fmt.Sprintf("/repos/%s/%s/git/commits/%s", url.PathEscape(owner), url.PathEscape(repository), url.PathEscape(sha))
	if _, err := r.get(path, nil, result); err != nil {
		return nil, err
	}

	c := &models.Commit{SHA: sha}
	c.Author.Name = result.Commit.Author.Name

	// Prefer the Gitea avatar: self-hosted instances often don't use Gravatar
	if result.Author != nil && result.Author.AvatarURL != "" {
		c.Author.AvatarURL = result.Author.AvatarURL
	} else if result.Commit.Author.Email != "" {
		c.Author.AvatarURL = gravatar.GetGravatarURL(result.Commit.Author.Email)
	}

	if c.Author.Name == "" && result.Author != nil {
		c.Author.Name = defaultString(result.Author.FullName, result.Author.Login)
	}

	return c, nil
}

// GetContributions loads the activity heatmap of a user (about one year, grouped in 15 minutes slots).
// Gitea returns 404 when the user does not exist, is not visible, or when the heatmap is disabled.
func (r *giteaRepository) GetContributions(user string) ([]models.Contribution, error) {
	var items []heatmapItem
	if _, err := r.get(fmt.Sprintf("/users/%s/heatmap", url.PathEscape(user)), nil, &items); err != nil {
		return nil, err
	}

	contributions := make([]models.Contribution, 0, len(items))
	for _, item := range items {
		contributions = append(contributions, models.Contribution{Time: time.Unix(item.Timestamp, 0), Count: item.Contributions})
	}

	return contributions, nil
}

func (r *giteaRepository) get(path string, query url.Values, result interface{}) (http.Header, error) {
	rawURL := r.config.URL + apiPath + path
	if len(query) > 0 {
		rawURL += "?" + query.Encode()
	}

	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	if r.config.Token != "" {
		request.Header.Set("Authorization", "token "+r.config.Token)
	}

	response, err := r.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, maxBodySize))
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea returned status %d: %s", response.StatusCode, truncate(strings.TrimSpace(string(data))))
	}

	if result != nil {
		if err = json.Unmarshal(data, result); err != nil {
			return nil, err
		}
	}

	return response.Header, nil
}

func toPullRequest(pr *pullRequest) *models.PullRequest {
	result := &models.PullRequest{
		ID:           pr.Number,
		Title:        pr.Title,
		SourceBranch: pr.Head.Ref,
		CommitSHA:    pr.Head.SHA,
	}

	if pr.Head.Repo != nil {
		result.SourceRepository = pr.Head.Repo.Name
		if pr.Head.Repo.Owner != nil {
			result.SourceOwner = pr.Head.Repo.Owner.Login
		}
	}

	if pr.User != nil {
		result.Author.Name = defaultString(pr.User.FullName, pr.User.Login)
		result.Author.AvatarURL = pr.User.AvatarURL
	}

	return result
}

// validTime maps the Unix epoch, used by Gitea for an unset date, to the zero time.
func validTime(t time.Time) time.Time {
	if t.Unix() <= 0 {
		return time.Time{}
	}
	return t
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func truncate(s string) string {
	if len(s) <= maxErrorBodyLength {
		return s
	}
	return strings.ToValidUTF8(s[:maxErrorBodyLength], "") + "..."
}
