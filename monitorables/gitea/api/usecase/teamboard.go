//go:build !faker

package usecase

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	uiConfigModels "github.com/monitoror/monitoror/api/config/models"
	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	"github.com/AlekSi/pointer"
)

const (
	defaultListLimit = 5
	defaultBugLabel  = "bug"

	// activityWeeks is the period of the commit activity line of a repository
	activityWeeks = 12

	repositoriesTTL = 5 * time.Minute
	aggregatesTTL   = time.Minute
)

// PullRequests lists the open pull requests of every repository of an owner, oldest first.
func (gu *giteaUsecase) PullRequests(params *models.ListParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaPullRequestsTileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = "Pull requests"

	pullRequests, err := gu.repository.SearchIssues(params.Owner, "pulls")
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load pull requests"}
	}
	sortOldestFirst(pullRequests)

	now := gu.now()
	tile.Status = coreModels.SuccessStatus
	tile.Metrics.Values = []string{strconv.Itoa(len(pullRequests))}
	for _, pullRequest := range firstIssues(pullRequests, params.Limit) {
		tile.Items = append(tile.Items, coreModels.TileItem{
			Label:  fmt.Sprintf("#%d %s", pullRequest.Number, pullRequest.Title),
			Detail: fmt.Sprintf("%s · %s", pullRequest.Repository, humanizeAge(now, pullRequest.CreatedAt)),
		})
	}

	return tile, nil
}

// Issues lists the open issues of every repository of an owner: bugs first, then the others, oldest first.
func (gu *giteaUsecase) Issues(params *models.IssuesListParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaIssuesTileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = "Issues"

	issues, err := gu.repository.SearchIssues(params.Owner, "issues")
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load issues"}
	}
	sortOldestFirst(issues)

	bugLabel := defaultOr(params.BugLabel, defaultBugLabel)
	var bugs, others []models.Issue
	for _, issue := range issues {
		if hasLabel(issue.Labels, bugLabel) {
			bugs = append(bugs, issue)
		} else {
			others = append(others, issue)
		}
	}

	now := gu.now()
	tile.Status = coreModels.SuccessStatus
	if len(bugs) > 0 {
		tile.Status = coreModels.WarningStatus
	}
	tile.Message = pluralize(len(bugs), "bug")
	tile.Metrics.Values = []string{strconv.Itoa(len(issues))}
	for _, issue := range firstIssues(append(bugs, others...), params.Limit) {
		item := coreModels.TileItem{
			Label:  fmt.Sprintf("#%d %s", issue.Number, issue.Title),
			Detail: fmt.Sprintf("%s · %s", issue.Repository, humanizeAge(now, issue.CreatedAt)),
		}
		if hasLabel(issue.Labels, bugLabel) {
			item.Tag = "bug"
		}
		tile.Items = append(tile.Items, item)
	}

	return tile, nil
}

// Actions lists the latest run of each workflow of every repository of an owner:
// running runs first with their progress, then queued and failed runs, then the others, newest first.
func (gu *giteaUsecase) Actions(params *models.ListParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaActionsTileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = "Actions"

	runs, err := gu.ownerRuns(params.Owner)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load actions"}
	}

	// Latest run of each workflow of each repository
	sort.SliceStable(runs, func(i, j int) bool { return runs[i].CreatedAt.After(runs[j].CreatedAt) })
	var latest []models.Run
	seen := make(map[string]bool)
	for _, run := range runs {
		key := run.Repository + "/" + run.Workflow
		if !seen[key] {
			seen[key] = true
			latest = append(latest, run)
		}
	}

	rank := func(run models.Run) int {
		switch parseRun(&run) {
		case coreModels.RunningStatus:
			return 0
		case coreModels.QueuedStatus:
			return 1
		case coreModels.FailedStatus:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(latest, func(i, j int) bool { return rank(latest[i]) < rank(latest[j]) })

	now := gu.now()
	running := 0
	tile.Status = coreModels.SuccessStatus
	for i, run := range latest {
		status := parseRun(&run)
		switch status {
		case coreModels.RunningStatus:
			running++
		case coreModels.FailedStatus:
			tile.Status = coreModels.FailedStatus
		}

		if i >= limitOr(params.Limit) {
			continue
		}

		item := coreModels.TileItem{
			Label:  fmt.Sprintf("%s · %s", run.Repository, defaultOr(run.Title, run.Workflow)),
			Status: status,
		}
		switch status {
		case coreModels.RunningStatus:
			elapsed := now.Sub(run.StartedAt)
			item.Detail = humanizeDuration(elapsed)
			if estimate := estimateDurationFromRuns(sameWorkflow(runs, run), []models.Run{run}); estimate != nil && *estimate > 0 {
				item.Progress = pointer.ToInt(int(elapsed * 100 / *estimate))
				item.Detail = fmt.Sprintf("%s / %s", humanizeDuration(elapsed), humanizeDuration(*estimate))
			}
		case coreModels.QueuedStatus:
			item.Detail = "queued"
		default:
			item.Detail = humanizeAge(now, defaultTime(run.CompletedAt, run.CreatedAt))
		}
		tile.Items = append(tile.Items, item)
	}
	tile.Metrics.Values = []string{strconv.Itoa(running)}

	return tile, nil
}

// Milestones lists the open milestones of every repository of an owner, due first.
func (gu *giteaUsecase) Milestones(params *models.ListParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaMilestonesTileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = "Milestones"

	repositories, err := gu.ownerRepositories(params.Owner)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load repositories"}
	}

	type repositoryMilestone struct {
		repository string
		milestone  models.Milestone
	}
	var milestones []repositoryMilestone
	for _, repository := range repositories {
		if repository.Archived {
			continue
		}
		repositoryMilestones, err := gu.repositoryMilestones(params.Owner, repository.Name)
		if err != nil {
			continue
		}
		for _, milestone := range repositoryMilestones {
			milestones = append(milestones, repositoryMilestone{repository.Name, milestone})
		}
	}
	sort.SliceStable(milestones, func(i, j int) bool {
		return milestoneBefore(milestones[i].milestone, milestones[j].milestone)
	})

	now := gu.now()
	tile.Status = coreModels.SuccessStatus
	tile.Metrics.Values = []string{strconv.Itoa(len(milestones))}
	for i, m := range milestones {
		status := milestoneStatus(now, m.milestone)
		tile.Status = worstStatus(tile.Status, status)

		if i < limitOr(params.Limit) {
			tile.Items = append(tile.Items, coreModels.TileItem{
				Label:  fmt.Sprintf("%s · %s", m.repository, m.milestone.Title),
				Detail: milestoneDetail(now, m.milestone),
				Status: status,
			})
		}
	}

	return tile, nil
}

// Commits lists the latest commits pushed to the repositories of an owner.
func (gu *giteaUsecase) Commits(params *models.ListParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaCommitsTileType)
	tile.Label = "Latest commits"

	commits, err := gu.repository.GetFeedCommits(params.Owner, limitOr(params.Limit))
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load commits"}
	}

	now := gu.now()
	tile.Status = coreModels.SuccessStatus
	for _, commit := range commits {
		tile.Items = append(tile.Items, coreModels.TileItem{
			Label:  commit.Message,
			Detail: fmt.Sprintf("%s · %s · %s", commit.Repository, commit.Author, humanizeAge(now, commit.PushedAt)),
		})
	}

	return tile, nil
}

// Repository sums up a repository: build status of the ref (default branch), commit activity of the last weeks,
// open pull requests, issues and bugs, next milestone.
func (gu *giteaUsecase) Repository(params *models.RepositoryParams) (*coreModels.Tile, error) {
	ref := params.Ref
	if ref == "" {
		ref = gu.defaultBranch(params.Owner, params.Repository)
	}

	tile, err := gu.Checks(&models.ChecksParams{Owner: params.Owner, Repository: params.Repository, Ref: ref})
	if err != nil {
		var monitororError *coreModels.MonitororError
		if !errors.As(err, &monitororError) || monitororError.ErrorStatus != coreModels.UnknownStatus {
			return nil, err
		}
		// No CI on this repository: the summary is still useful
		tile = coreModels.NewTile(api.GiteaRepositoryTileType)
		tile.Label = params.Repository
		tile.Status = coreModels.SuccessStatus
	}
	tile.Type = api.GiteaRepositoryTileType
	tile.Label = params.Repository

	now := gu.now()
	summary := gu.repositorySummary(params.Owner, params.Repository, ref, defaultOr(params.BugLabel, defaultBugLabel), now)

	if summary.weeklyCommits != nil {
		tile.WithMetrics(coreModels.NumberUnit)
		tile.Metrics.Values = summary.weeklyCommits
	}
	if summary.pullRequests != nil {
		tile.Items = append(tile.Items, coreModels.TileItem{Label: "PR", Detail: strconv.Itoa(*summary.pullRequests)})
	}
	if summary.issues != nil {
		tile.Items = append(tile.Items, coreModels.TileItem{Label: "Issues", Detail: strconv.Itoa(*summary.issues)})
		if *summary.bugs > 0 {
			tile.Items = append(tile.Items, coreModels.TileItem{Label: "Bugs", Detail: strconv.Itoa(*summary.bugs), Tag: "bug"})
		}
	}
	if summary.milestone != nil {
		tile.Message = fmt.Sprintf("%s · %s", summary.milestone.Title, milestoneDetail(now, *summary.milestone))
	}

	return tile, nil
}

// RepositoriesGenerator generates one repository tile per repository of an owner, archived and empty ones excluded.
func (gu *giteaUsecase) RepositoriesGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error) {
	generatorParams := params.(*models.RepositoryGeneratorParams)

	repositories, err := gu.ownerRepositories(generatorParams.Owner)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Message: "unable to load repositories"}
	}

	var results []uiConfigModels.GeneratedTile
	for _, repository := range repositories {
		if repository.Archived || repository.Empty {
			continue
		}

		results = append(results, uiConfigModels.GeneratedTile{
			Label: repository.Name,
			Params: &models.RepositoryParams{
				Owner:      generatorParams.Owner,
				Repository: repository.Name,
				Ref:        repository.DefaultBranch,
				BugLabel:   generatorParams.BugLabel,
			},
		})
	}

	return results, nil
}

type repositorySummary struct {
	weeklyCommits []string // commits per week, oldest first
	pullRequests  *int
	issues        *int
	bugs          *int
	milestone     *models.Milestone
}

// repositorySummary loads the slow-changing parts of a repository tile, cached for a minute.
// A part that fails to load is left out, the others are still shown.
func (gu *giteaUsecase) repositorySummary(owner, repository, ref, bugLabel string, now time.Time) repositorySummary {
	key := fmt.Sprintf("summary/%s/%s/%s/%s", owner, repository, ref, bugLabel)
	value, _ := gu.aggregates.get(key, aggregatesTTL, now, func() (interface{}, error) {
		var summary repositorySummary

		since := now.AddDate(0, 0, -7*activityWeeks)
		if commits, err := gu.repository.GetCommits(owner, repository, ref, since); err == nil {
			summary.weeklyCommits = weeklyCounts(now, commits)
		}

		if count, err := gu.repository.GetCount(&models.CountParams{Owner: owner, Repository: repository, Type: "pulls", State: "open"}); err == nil {
			summary.pullRequests = pointer.ToInt(count)
		}

		if issues, err := gu.repository.GetRepositoryIssues(owner, repository); err == nil {
			bugs := 0
			for _, issue := range issues {
				if hasLabel(issue.Labels, bugLabel) {
					bugs++
				}
			}
			summary.issues = pointer.ToInt(len(issues))
			summary.bugs = pointer.ToInt(bugs)
		}

		if milestones, err := gu.repository.GetMilestones(owner, repository); err == nil {
			summary.milestone = nextMilestone(milestones)
		}

		return summary, nil
	})

	return value.(repositorySummary)
}

func (gu *giteaUsecase) ownerRepositories(owner string) ([]models.Repository, error) {
	value, err := gu.aggregates.get("repositories/"+owner, repositoriesTTL, gu.now(), func() (interface{}, error) {
		return gu.repository.GetRepositories(owner)
	})
	if err != nil {
		return nil, err
	}
	return value.([]models.Repository), nil
}

func (gu *giteaUsecase) repositoryMilestones(owner, repository string) ([]models.Milestone, error) {
	value, err := gu.aggregates.get("milestones/"+owner+"/"+repository, aggregatesTTL, gu.now(), func() (interface{}, error) {
		return gu.repository.GetMilestones(owner, repository)
	})
	if err != nil {
		return nil, err
	}
	return value.([]models.Milestone), nil
}

func (gu *giteaUsecase) defaultBranch(owner, repository string) string {
	if repositories, err := gu.ownerRepositories(owner); err == nil {
		for _, r := range repositories {
			if r.Name == repository && r.DefaultBranch != "" {
				return r.DefaultBranch
			}
		}
	}
	return "main"
}

// ownerRuns reads the runs of an organization in one call, or of each repository for a user
func (gu *giteaUsecase) ownerRuns(owner string) ([]models.Run, error) {
	runs, err := gu.repository.GetOwnerRuns(owner)
	if err == nil {
		return runs, nil
	}

	repositories, err := gu.ownerRepositories(owner)
	if err != nil {
		return nil, err
	}

	runs = nil
	for _, repository := range repositories {
		if repository.Archived || repository.Empty {
			continue
		}
		repositoryRuns, err := gu.repository.GetRuns(owner, repository.Name, "")
		if err != nil {
			continue
		}
		for i := range repositoryRuns {
			if repositoryRuns[i].Repository == "" {
				repositoryRuns[i].Repository = repository.Name
			}
		}
		runs = append(runs, repositoryRuns...)
	}

	return runs, nil
}

func sameWorkflow(runs []models.Run, run models.Run) []models.Run {
	var results []models.Run
	for _, r := range runs {
		if r.Repository == run.Repository && r.Workflow == run.Workflow {
			results = append(results, r)
		}
	}
	return results
}

// weeklyCounts counts the commits of each of the last activityWeeks weeks (7 days periods ending now), oldest first
func weeklyCounts(now time.Time, commits []models.RepositoryCommit) []string {
	counts := make([]int, activityWeeks)
	for _, commit := range commits {
		age := now.Sub(commit.Date)
		if age < 0 {
			age = 0
		}
		week := int(age / (7 * 24 * time.Hour))
		if week < activityWeeks {
			counts[activityWeeks-1-week]++
		}
	}

	values := make([]string, activityWeeks)
	for i, count := range counts {
		values[i] = strconv.Itoa(count)
	}
	return values
}

func milestoneStatus(now time.Time, milestone models.Milestone) coreModels.TileStatus {
	if milestone.DueOn == nil || milestone.OpenIssues == 0 {
		return coreModels.SuccessStatus
	}
	days := daysUntil(now, *milestone.DueOn)
	switch {
	case days < 0:
		return coreModels.FailedStatus
	case days <= milestoneWarningDays:
		return coreModels.WarningStatus
	}
	return coreModels.SuccessStatus
}

func milestoneDetail(now time.Time, milestone models.Milestone) string {
	var parts []string
	if milestone.DueOn != nil {
		parts = append(parts, humanizeDue(daysUntil(now, *milestone.DueOn)))
	}
	if total := milestone.OpenIssues + milestone.ClosedIssues; total > 0 {
		parts = append(parts, fmt.Sprintf("%d%%", milestone.ClosedIssues*100/total))
	}
	return strings.Join(parts, " · ")
}

func worstStatus(current, candidate coreModels.TileStatus) coreModels.TileStatus {
	order := map[coreModels.TileStatus]int{coreModels.SuccessStatus: 0, coreModels.WarningStatus: 1, coreModels.FailedStatus: 2}
	if order[candidate] > order[current] {
		return candidate
	}
	return current
}

func sortOldestFirst(issues []models.Issue) {
	sort.SliceStable(issues, func(i, j int) bool { return issues[i].CreatedAt.Before(issues[j].CreatedAt) })
}

func firstIssues(issues []models.Issue, limit int) []models.Issue {
	if limit = limitOr(limit); len(issues) > limit {
		return issues[:limit]
	}
	return issues
}

func limitOr(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	return limit
}

func defaultOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}

func hasLabel(labels []string, name string) bool {
	for _, label := range labels {
		if strings.EqualFold(label, name) {
			return true
		}
	}
	return false
}

func pluralize(count int, word string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", count, word)
}

// humanizeAge returns a short age: "now", "12m", "3h", "5d"
func humanizeAge(now, t time.Time) string {
	age := now.Sub(t)
	switch {
	case age < time.Minute:
		return "now"
	case age < time.Hour:
		return fmt.Sprintf("%dm", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh", int(age.Hours()))
	}
	return fmt.Sprintf("%dd", int(age.Hours()/24))
}

// humanizeDuration returns a short duration: "45s", "3m", "1h05"
func humanizeDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%02d", int(d.Hours()), int(d.Minutes())%60)
}
