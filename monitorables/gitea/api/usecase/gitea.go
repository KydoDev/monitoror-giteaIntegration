//go:build !faker

package usecase

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	uiConfigModels "github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/internal/pkg/monitorable/cache"
	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"
	"github.com/monitoror/monitoror/pkg/git"
	"github.com/monitoror/monitoror/pkg/hash"

	"github.com/AlekSi/pointer"
)

type (
	giteaUsecase struct {
		repository api.Repository

		// builds cache. used for save small history of build for stats
		buildsCache *cache.BuildCache
	}
)

var orderedTileStatus = map[coreModels.TileStatus]int{
	coreModels.RunningStatus:        0,
	coreModels.FailedStatus:         1,
	coreModels.WarningStatus:        2,
	coreModels.CanceledStatus:       3,
	coreModels.ActionRequiredStatus: 4,
	coreModels.QueuedStatus:         5,
	coreModels.SuccessStatus:        6,
	coreModels.DisabledStatus:       7,
	coreModels.UnknownStatus:        8,
}

const buildCacheSize = 5

func NewGiteaUsecase(repository api.Repository) api.Usecase {
	return &giteaUsecase{
		repository,
		cache.NewBuildCache(buildCacheSize),
	}
}

func (gu *giteaUsecase) Count(params *models.CountParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaCountTileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = "Gitea count"

	count, err := gu.repository.GetCount(params)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load count"}
	}

	tile.Status = coreModels.SuccessStatus
	tile.Metrics.Values = append(tile.Metrics.Values, fmt.Sprintf("%d", count))

	return tile, nil
}

func (gu *giteaUsecase) Checks(params *models.ChecksParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaChecksTileType).WithBuild()
	tile.Label = params.Repository
	tile.Build.Branch = pointer.ToString(git.HumanizeBranch(params.Ref))

	// Request checks
	checks, err := gu.repository.GetChecks(params.Owner, params.Repository, params.Ref)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load ref checks"}
	}
	// Request Gitea Actions runs: manual runs (workflow_dispatch) create no commit status.
	// Runs are optional: without them (older Gitea, Actions disabled), commit statuses are used alone.
	runs, err := gu.repository.GetRuns(params.Owner, params.Repository, params.Ref)
	if err != nil {
		runs = nil
	}
	buildRuns := selectBuildRuns(checks, runs)

	if len(checks.Statuses) == 0 && len(buildRuns) == 0 {
		// Unknown because request was correct but there is no build
		return nil, &coreModels.MonitororError{Tile: tile, Message: "no ref checks found", ErrorStatus: coreModels.UnknownStatus}
	}

	// Compute checks into tile
	gu.computeRefChecks(tile, checks, buildRuns, params.String())

	if len(buildRuns) > 0 {
		tile.Build.ID = pointer.ToString(strconv.FormatInt(buildRuns[0].Number, 10))

		// Without history in cache (e.g. after a restart), estimate the duration from previous runs
		if tile.Status == coreModels.RunningStatus && *tile.Build.EstimatedDuration == 0 {
			if estimatedDuration := estimateDurationFromRuns(runs, buildRuns); estimatedDuration != nil {
				tile.Build.EstimatedDuration = pointer.ToInt64(int64(estimatedDuration.Seconds()))
			}
		}
	}

	// Author of last commit
	headCommit := checks.HeadCommit
	if headCommit == nil && len(buildRuns) > 0 {
		headCommit = &buildRuns[0].HeadSHA
	}
	if tile.Status == coreModels.FailedStatus && headCommit != nil {
		commit, err := gu.repository.GetCommit(params.Owner, params.Repository, *headCommit)
		if err == nil {
			tile.Build.Author = &coreModels.Author{
				Name:      commit.Author.Name,
				AvatarURL: commit.Author.AvatarURL,
			}
		}
	}

	return tile, nil
}

func (gu *giteaUsecase) PullRequest(params *models.PullRequestParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaPullRequestTileType).WithBuild()

	// Request pullRequest
	pullRequest, err := gu.repository.GetPullRequest(params.Owner, params.Repository, *params.ID)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load pull request"}
	}

	tile.Label = params.Repository
	tile.Build.Branch = pointer.ToString(git.HumanizeBranch(pullRequest.SourceBranch))
	if pullRequest.SourceOwner != "" && params.Owner != pullRequest.SourceOwner {
		tile.Build.Branch = pointer.ToString(fmt.Sprintf("%s:%s", pullRequest.SourceOwner, *tile.Build.Branch))
	}
	tile.Build.MergeRequest = &coreModels.TileMergeRequest{
		ID:    pullRequest.ID,
		Title: pullRequest.Title,
	}

	// With pull request, we use CommitSHA as Ref on params.Owner/params.Repo to handle correctly Checks from Forks
	checks, err := gu.repository.GetChecks(params.Owner, params.Repository, pullRequest.CommitSHA)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load ref checks"}
	}

	// Compute checks into tile
	gu.computeRefChecks(tile, checks, nil, params.String())

	// Author of pull request
	if tile.Status == coreModels.FailedStatus {
		tile.Build.Author = &pullRequest.Author
	}

	return tile, nil
}

func (gu *giteaUsecase) PullRequestsGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error) {
	prParams := params.(*models.PullRequestGeneratorParams)

	pullRequests, err := gu.repository.GetPullRequests(prParams.Owner, prParams.Repository)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Message: "unable to load pull request"}
	}

	var results []uiConfigModels.GeneratedTile
	for _, pullRequest := range pullRequests {
		p := &models.PullRequestParams{}
		p.Owner = prParams.Owner
		p.Repository = prParams.Repository
		p.ID = pointer.ToInt(pullRequest.ID)

		results = append(results, uiConfigModels.GeneratedTile{
			Params: p,
		})
	}

	return results, nil
}

func (gu *giteaUsecase) computeRefChecks(tile *coreModels.Tile, checks *models.Checks, runs []models.Run, paramsKey string) {
	// convert checks
	statuses, startedAt, finishedAt, id := convertChecks(checks, runs)

	// if statuses is empty, return SuccessStatus by default
	if len(statuses) == 0 {
		tile.Status = coreModels.SuccessStatus
		tile.Build.PreviousStatus = coreModels.UnknownStatus

		return
	}

	tile.Status = statuses[0]

	// Set Previous Status
	previousStatus := gu.buildsCache.GetPreviousStatus(paramsKey, id)
	if previousStatus != nil {
		tile.Build.PreviousStatus = *previousStatus
	} else {
		tile.Build.PreviousStatus = coreModels.UnknownStatus
	}

	// StartedAt / FinishedAt
	tile.Build.StartedAt = startedAt
	if tile.Status != coreModels.RunningStatus && tile.Status != coreModels.QueuedStatus {
		tile.Build.FinishedAt = finishedAt
	}

	// Duration
	if tile.Status == coreModels.RunningStatus {
		tile.Build.Duration = pointer.ToInt64(int64(time.Since(*tile.Build.StartedAt).Seconds()))

		estimatedDuration := gu.buildsCache.GetEstimatedDuration(paramsKey)
		if estimatedDuration != nil {
			tile.Build.EstimatedDuration = pointer.ToInt64(int64(estimatedDuration.Seconds()))
		} else {
			tile.Build.EstimatedDuration = pointer.ToInt64(int64(0))
		}
	}

	// Cache Duration when success / failed
	if tile.Status == coreModels.SuccessStatus || tile.Status == coreModels.FailedStatus || tile.Status == coreModels.WarningStatus {
		gu.buildsCache.Add(paramsKey, id, tile.Status, tile.Build.FinishedAt.Sub(*tile.Build.StartedAt))
	}
}

// convertChecks transform models.Checks and the runs of the build to use them in computeRefChecks
func convertChecks(checks *models.Checks, runs []models.Run) ([]coreModels.TileStatus, *time.Time, *time.Time, string) {
	var statuses []coreModels.TileStatus
	var startedAt *time.Time = nil
	var finishedAt *time.Time = nil
	var ids = ""

	// Sort statues from newest to oldest and save every title to remove duplicate statues
	// Gitea adds a new status each time a status changes. Timestamps have a one second precision,
	// so the highest ID wins when two statuses share the same date.
	sort.Slice(checks.Statuses, func(i, j int) bool {
		if checks.Statuses[i].CreatedAt.Equal(checks.Statuses[j].CreatedAt) {
			return checks.Statuses[i].ID > checks.Statuses[j].ID
		}
		return checks.Statuses[i].CreatedAt.After(checks.Statuses[j].CreatedAt)
	})

	titles := make(map[string]bool)
	for i := range checks.Statuses {
		status := &checks.Statuses[i]
		if _, ok := titles[status.Title]; !ok {
			statuses = append(statuses, parseStatus(status))
			titles[status.Title] = true
		}

		if startedAt == nil || startedAt.After(status.CreatedAt) {
			startedAt = &status.CreatedAt
		}
		if finishedAt == nil || finishedAt.Before(status.UpdatedAt) {
			finishedAt = &status.UpdatedAt
		}
		ids = fmt.Sprintf("%s-%d", ids, status.ID)
	}

	for i := range runs {
		run := &runs[i]
		statuses = append(statuses, parseRun(run))

		runStartedAt := run.CreatedAt
		if !run.StartedAt.IsZero() {
			runStartedAt = run.StartedAt
		}
		runFinishedAt := runStartedAt
		if !run.CompletedAt.IsZero() {
			runFinishedAt = run.CompletedAt
		}

		if startedAt == nil || startedAt.After(runStartedAt) {
			startedAt = &runStartedAt
		}
		if finishedAt == nil || finishedAt.Before(runFinishedAt) {
			finishedAt = &runFinishedAt
		}
		// A re-run keeps the run id and increments the attempt
		ids = fmt.Sprintf("%s-run%d.%d", ids, run.ID, run.Attempt)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return orderedTileStatus[statuses[i]] < orderedTileStatus[statuses[j]]
	})

	return statuses, startedAt, finishedAt, hash.GetMD5Hash(ids)
}

func parseStatus(status *models.Status) coreModels.TileStatus {
	// Based on Gitea CommitStatusState : pending, success, error, failure, warning, skipped
	switch status.State {
	case "success":
		return coreModels.SuccessStatus
	case "failure", "error":
		return coreModels.FailedStatus
	case "warning":
		return coreModels.WarningStatus
	case "pending":
		return coreModels.RunningStatus
	case "skipped":
		return coreModels.DisabledStatus
	}

	return coreModels.UnknownStatus
}

func parseRun(run *models.Run) coreModels.TileStatus {
	switch run.Status {
	case "completed":
		switch run.Conclusion {
		case "success":
			return coreModels.SuccessStatus
		case "failure":
			return coreModels.FailedStatus
		case "cancelled":
			return coreModels.CanceledStatus
		case "skipped":
			return coreModels.DisabledStatus
		}
	case "in_progress":
		return coreModels.RunningStatus
	case "queued", "waiting", "pending", "requested":
		return coreModels.QueuedStatus
	}

	return coreModels.UnknownStatus
}

// selectBuildRuns keeps the runs of the build: the latest run of each workflow on the head commit.
// The head commit comes from the commit statuses, or from the newest run when there is no status.
// Commit statuses written by these runs are removed from checks, so a run is not counted twice.
func selectBuildRuns(checks *models.Checks, runs []models.Run) []models.Run {
	if len(runs) == 0 {
		return nil
	}

	sorted := make([]models.Run, len(runs))
	copy(sorted, runs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID > sorted[j].ID })

	headSHA := sorted[0].HeadSHA
	if checks.HeadCommit != nil {
		headSHA = *checks.HeadCommit
	}

	var buildRuns []models.Run
	headRunIDs := make(map[string]bool)
	workflows := make(map[string]bool)
	for _, run := range sorted {
		if run.HeadSHA != headSHA {
			continue
		}
		headRunIDs[strconv.FormatInt(run.ID, 10)] = true
		if !workflows[run.Workflow] {
			buildRuns = append(buildRuns, run)
			workflows[run.Workflow] = true
		}
	}

	if len(buildRuns) > 0 {
		statuses := []models.Status{}
		for _, status := range checks.Statuses {
			if runID := actionsRunID(status.TargetURL); runID == "" || !headRunIDs[runID] {
				statuses = append(statuses, status)
			}
		}
		checks.Statuses = statuses
	}

	return buildRuns
}

// actionsRunID returns the run id of a commit status written by Gitea Actions
// (target url ".../actions/runs/{id}/jobs/{job}"), or "" for any other status.
func actionsRunID(targetURL string) string {
	const marker = "/actions/runs/"
	i := strings.Index(targetURL, marker)
	if i < 0 {
		return ""
	}
	id := targetURL[i+len(marker):]
	if j := strings.Index(id, "/"); j >= 0 {
		id = id[:j]
	}
	return id
}

// estimateDurationFromRuns averages the previous successful runs of each running workflow of the build
// and returns the longest workflow estimate, or nil without history.
func estimateDurationFromRuns(runs, buildRuns []models.Run) *time.Duration {
	var estimate *time.Duration
	for _, buildRun := range buildRuns {
		if buildRun.Status != "in_progress" {
			continue
		}

		var total time.Duration
		var count int
		for _, run := range runs {
			if count == buildCacheSize {
				break
			}
			if run.ID == buildRun.ID || run.Workflow != buildRun.Workflow || run.Conclusion != "success" ||
				run.StartedAt.IsZero() || run.CompletedAt.IsZero() {
				continue
			}
			total += run.CompletedAt.Sub(run.StartedAt)
			count++
		}

		if count > 0 {
			average := total / time.Duration(count)
			if estimate == nil || average > *estimate {
				estimate = &average
			}
		}
	}

	return estimate
}
