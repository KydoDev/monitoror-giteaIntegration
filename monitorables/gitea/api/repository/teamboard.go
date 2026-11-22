package repository

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/monitoror/monitoror/monitorables/gitea/api/models"
)

// maxCommitPages limits the commits read for the commit activity of a repository
const maxCommitPages = 10

type (
	searchedIssue struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Repository struct {
			Name  string `json:"name"`
			Owner string `json:"owner"`
		} `json:"repository"`
	}

	repositoryCommit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}

	feedActivity struct {
		OpType  string    `json:"op_type"`
		RefName string    `json:"ref_name"`
		Created time.Time `json:"created"`
		Content string    `json:"content"`
		ActUser *user     `json:"act_user"`
		Repo    *struct {
			Name string `json:"name"`
		} `json:"repo"`
	}

	feedPushContent struct {
		Commits []struct {
			Message    string `json:"Message"`
			AuthorName string `json:"AuthorName"`
		} `json:"Commits"`
	}
)

// SearchIssues returns the open issues or pull requests (kind "issues" or "pulls") of every repository of an owner.
func (r *giteaRepository) SearchIssues(owner, kind string) ([]models.Issue, error) {
	query := url.Values{}
	query.Set("owner", owner)
	query.Set("type", kind)
	query.Set("state", "open")

	issues, err := getAllPages[searchedIssue](r, "/repos/issues/search", query)
	if err != nil {
		return nil, err
	}

	results := make([]models.Issue, 0, len(issues))
	for _, issue := range issues {
		labels := make([]string, 0, len(issue.Labels))
		for _, label := range issue.Labels {
			labels = append(labels, label.Name)
		}

		results = append(results, models.Issue{
			Number:     issue.Number,
			Title:      issue.Title,
			Owner:      issue.Repository.Owner,
			Repository: issue.Repository.Name,
			Labels:     labels,
			CreatedAt:  issue.CreatedAt,
		})
	}

	return results, nil
}

// GetRepositoryIssues returns the open issues (not pull requests) of a repository.
func (r *giteaRepository) GetRepositoryIssues(owner, repository string) ([]models.Issue, error) {
	query := url.Values{}
	query.Set("type", "issues")
	query.Set("state", "open")

	issues, err := getAllPages[searchedIssue](r, fmt.Sprintf("/repos/%s/%s/issues", url.PathEscape(owner), url.PathEscape(repository)), query)
	if err != nil {
		return nil, err
	}

	results := make([]models.Issue, 0, len(issues))
	for _, issue := range issues {
		labels := make([]string, 0, len(issue.Labels))
		for _, label := range issue.Labels {
			labels = append(labels, label.Name)
		}

		results = append(results, models.Issue{
			Number:     issue.Number,
			Title:      issue.Title,
			Owner:      owner,
			Repository: repository,
			Labels:     labels,
			CreatedAt:  issue.CreatedAt,
		})
	}

	return results, nil
}

// GetOwnerRuns returns the latest Gitea Actions runs of an organization, newest first.
// It fails for a user (no owner-wide endpoint) and on Gitea versions without it.
func (r *giteaRepository) GetOwnerRuns(owner string) ([]models.Run, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(PageSize))

	var result workflowRuns
	if _, err := r.get(fmt.Sprintf("/orgs/%s/actions/runs", url.PathEscape(owner)), query, &result); err != nil {
		return nil, err
	}

	return toRuns(result.WorkflowRuns), nil
}

// GetCommits returns the commits of a ref, newest first, stopping at the first page older than since.
// Commit order follows git history, so the limit is a number of pages, not a date.
func (r *giteaRepository) GetCommits(owner, repository, ref string, since time.Time) ([]models.RepositoryCommit, error) {
	path := fmt.Sprintf("/repos/%s/%s/commits", url.PathEscape(owner), url.PathEscape(repository))

	var results []models.RepositoryCommit
	for page := 1; page <= maxCommitPages; page++ {
		query := url.Values{}
		query.Set("sha", ref)
		query.Set("stat", "false")
		query.Set("verification", "false")
		query.Set("files", "false")
		query.Set("page", strconv.Itoa(page))
		query.Set("limit", strconv.Itoa(PageSize))

		var commits []repositoryCommit
		if _, err := r.get(path, query, &commits); err != nil {
			return nil, err
		}

		recent := false
		for _, commit := range commits {
			date := commit.Commit.Committer.Date
			if !date.Before(since) {
				recent = true
			}
			results = append(results, models.RepositoryCommit{SHA: commit.SHA, Date: date})
		}

		if len(commits) < PageSize || !recent {
			break
		}
	}

	return results, nil
}

// GetFeedCommits returns the commits pushed to the repositories of an owner, newest push first.
// The organization feed is used, or the user feed when the owner is not an organization (404).
func (r *giteaRepository) GetFeedCommits(owner string, limit int) ([]models.FeedCommit, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(PageSize))

	var activities []feedActivity
	if _, err := r.get(fmt.Sprintf("/orgs/%s/activities/feeds", url.PathEscape(owner)), query, &activities); err != nil {
		// Only an unknown organization means a user: other errors (timeout, 5xx) are reported as they are
		if !models.IsNotFound(err) {
			return nil, err
		}
		if _, err = r.get(fmt.Sprintf("/users/%s/activities/feeds", url.PathEscape(owner)), query, &activities); err != nil {
			return nil, err
		}
	}

	var results []models.FeedCommit
	for _, activity := range activities {
		if activity.OpType != "commit_repo" || activity.Repo == nil {
			continue
		}

		var content feedPushContent
		if err := json.Unmarshal([]byte(activity.Content), &content); err != nil {
			continue
		}

		for _, commit := range content.Commits {
			author := commit.AuthorName
			if author == "" && activity.ActUser != nil {
				author = defaultString(activity.ActUser.FullName, activity.ActUser.Login)
			}

			results = append(results, models.FeedCommit{
				Repository: activity.Repo.Name,
				Branch:     strings.TrimPrefix(activity.RefName, "refs/heads/"),
				Author:     author,
				Message:    strings.SplitN(strings.TrimSpace(commit.Message), "\n", 2)[0],
				PushedAt:   activity.Created,
			})
			if len(results) == limit {
				return results, nil
			}
		}
	}

	return results, nil
}

// GetOrganizationMembers returns the logins of the public and visible members of an organization.
func (r *giteaRepository) GetOrganizationMembers(organization string) ([]string, error) {
	members, err := getAllPages[user](r, fmt.Sprintf("/orgs/%s/members", url.PathEscape(organization)), url.Values{})
	if err != nil {
		return nil, err
	}

	logins := make([]string, 0, len(members))
	for _, member := range members {
		logins = append(logins, member.Login)
	}

	return logins, nil
}
