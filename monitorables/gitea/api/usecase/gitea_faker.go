//go:build faker

package usecase

import (
	"fmt"
	"math/rand"
	"time"

	uiConfigModels "github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/internal/pkg/monitorable/faker"
	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"
	"github.com/monitoror/monitoror/pkg/git"
	"github.com/monitoror/monitoror/pkg/nonempty"

	"github.com/AlekSi/pointer"
	cmap "github.com/orcaman/concurrent-map"
)

type (
	giteaUsecase struct {
		timeRefByProject cmap.ConcurrentMap
	}
)

var availableBuildStatus = faker.Statuses{
	{coreModels.SuccessStatus, time.Second * 30},
	{coreModels.FailedStatus, time.Second * 30},
	{coreModels.CanceledStatus, time.Second * 20},
	{coreModels.RunningStatus, time.Second * 60},
	{coreModels.QueuedStatus, time.Second * 30},
	{coreModels.WarningStatus, time.Second * 20},
	{coreModels.DisabledStatus, time.Second * 20},
	{coreModels.ActionRequiredStatus, time.Second * 20},
}

func NewGiteaUsecase() api.Usecase {
	return &giteaUsecase{cmap.New()}
}

func (gu *giteaUsecase) Count(params *models.CountParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaCountTileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = "Gitea count"

	tile.Status = coreModels.SuccessStatus

	if len(params.ValueValues) != 0 {
		tile.Metrics.Values = params.ValueValues
	} else {
		tile.Metrics.Values = []string{"42"}
	}

	return tile, nil
}

func (gu *giteaUsecase) Checks(params *models.ChecksParams) (tile *coreModels.Tile, err error) {
	tile = coreModels.NewTile(api.GiteaChecksTileType).WithBuild()
	tile.Label = params.Repository

	projectID := fmt.Sprintf("%s-%s-%s", params.Owner, params.Repository, params.Ref)
	tile.Status = nonempty.Struct(params.Status, gu.computeStatus(projectID)).(coreModels.TileStatus)

	if tile.Status == coreModels.DisabledStatus {
		return
	}

	if tile.Status == coreModels.WarningStatus {
		// Warning can be Unstable Build
		if rand.Intn(2) == 0 {
			tile.Message = "Fake error message"
			return
		}
	}

	tile.Build.Branch = pointer.ToString(git.HumanizeBranch(params.Ref))
	tile.Build.PreviousStatus = nonempty.Struct(params.PreviousStatus, coreModels.SuccessStatus).(coreModels.TileStatus)

	// Author
	if tile.Status == coreModels.FailedStatus {
		tile.Build.Author = &coreModels.Author{}
		tile.Build.Author.Name = nonempty.String(params.AuthorName, "John Doe")
		tile.Build.Author.AvatarURL = nonempty.String(params.AuthorAvatarURL, "https://monitoror.com/assets/images/avatar.png")
	}

	// Duration / EstimatedDuration
	if tile.Status == coreModels.RunningStatus {
		estimatedDuration := nonempty.Duration(time.Duration(params.EstimatedDuration), time.Second*300)
		tile.Build.Duration = pointer.ToInt64(nonempty.Int64(params.Duration, int64(gu.computeDuration(projectID, estimatedDuration).Seconds())))

		if tile.Build.PreviousStatus != coreModels.UnknownStatus {
			tile.Build.EstimatedDuration = pointer.ToInt64(int64(estimatedDuration.Seconds()))
		} else {
			tile.Build.EstimatedDuration = pointer.ToInt64(0)
		}
	}

	// StartedAt / FinishedAt
	if tile.Build.Duration == nil {
		tile.Build.StartedAt = pointer.ToTime(nonempty.Time(params.StartedAt, time.Now().Add(-time.Minute*10)))
	} else {
		tile.Build.StartedAt = pointer.ToTime(nonempty.Time(params.StartedAt, time.Now().Add(-time.Second*time.Duration(*tile.Build.Duration))))
	}

	if tile.Status != coreModels.QueuedStatus && tile.Status != coreModels.RunningStatus {
		tile.Build.FinishedAt = pointer.ToTime(nonempty.Time(params.FinishedAt, tile.Build.StartedAt.Add(time.Minute*5)))
	}

	return tile, nil
}

func (gu *giteaUsecase) PullRequest(params *models.PullRequestParams) (tile *coreModels.Tile, err error) {
	tile = coreModels.NewTile(api.GiteaPullRequestTileType).WithBuild()
	tile.Label = params.Repository

	projectID := fmt.Sprintf("%s-%s-%d", params.Owner, params.Repository, *params.ID)
	tile.Status = nonempty.Struct(params.Status, gu.computeStatus(projectID)).(coreModels.TileStatus)

	if tile.Status == coreModels.DisabledStatus {
		return
	}

	if tile.Status == coreModels.WarningStatus {
		// Warning can be Unstable Build
		if rand.Intn(2) == 0 {
			tile.Message = "Fake error message"
			return
		}
	}

	tile.Build.Branch = pointer.ToString(nonempty.String(git.HumanizeBranch(params.Branch), "feature-branch"))
	tile.Build.PreviousStatus = nonempty.Struct(params.PreviousStatus, coreModels.SuccessStatus).(coreModels.TileStatus)
	tile.Build.MergeRequest = &coreModels.TileMergeRequest{
		ID:    *params.ID,
		Title: nonempty.String(params.MergeRequestTitle, "Feature branch title"),
	}

	// Author
	if tile.Status == coreModels.FailedStatus {
		tile.Build.Author = &coreModels.Author{}
		tile.Build.Author.Name = nonempty.String(params.AuthorName, "John Doe")
		tile.Build.Author.AvatarURL = nonempty.String(params.AuthorAvatarURL, "https://monitoror.com/assets/images/avatar.png")
	}

	// Duration / EstimatedDuration
	if tile.Status == coreModels.RunningStatus {
		estimatedDuration := nonempty.Duration(time.Duration(params.EstimatedDuration), time.Second*300)
		tile.Build.Duration = pointer.ToInt64(nonempty.Int64(params.Duration, int64(gu.computeDuration(projectID, estimatedDuration).Seconds())))

		if tile.Build.PreviousStatus != coreModels.UnknownStatus {
			tile.Build.EstimatedDuration = pointer.ToInt64(int64(estimatedDuration.Seconds()))
		} else {
			tile.Build.EstimatedDuration = pointer.ToInt64(0)
		}
	}

	// StartedAt / FinishedAt
	if tile.Build.Duration == nil {
		tile.Build.StartedAt = pointer.ToTime(nonempty.Time(params.StartedAt, time.Now().Add(-time.Minute*10)))
	} else {
		tile.Build.StartedAt = pointer.ToTime(nonempty.Time(params.StartedAt, time.Now().Add(-time.Second*time.Duration(*tile.Build.Duration))))
	}

	if tile.Status != coreModels.QueuedStatus && tile.Status != coreModels.RunningStatus {
		tile.Build.FinishedAt = pointer.ToTime(nonempty.Time(params.FinishedAt, tile.Build.StartedAt.Add(time.Minute*5)))
	}

	return tile, nil
}

func (gu *giteaUsecase) Contributions(params *models.ContributionsParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaContributionsTileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = params.User
	tile.Status = coreModels.SuccessStatus
	tile.Metrics.Values = faker.Contributions(params.User+params.Organization, params.Weeks)

	return tile, nil
}

func (gu *giteaUsecase) ChecksGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error) {
	panic("unimplemented")
}

func (gu *giteaUsecase) PullRequestsGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error) {
	panic("unimplemented")
}

func (gu *giteaUsecase) Milestone(params *models.MilestoneParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaMilestoneTileType).WithMetrics(coreModels.RawUnit)
	tile.Label = params.Repository
	tile.Status = coreModels.SuccessStatus
	tile.Message = "v2.0 · due in 5 days"
	tile.Metrics.Values = []string{"60%"}

	return tile, nil
}

func (gu *giteaUsecase) MilestonesGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error) {
	panic("unimplemented")
}

func (gu *giteaUsecase) computeStatus(projectID string) coreModels.TileStatus {
	value, ok := gu.timeRefByProject.Get(projectID)
	if !ok || value == nil {
		value = faker.GetRefTime()
		gu.timeRefByProject.Set(projectID, value)
	}

	return faker.ComputeStatus(value.(time.Time), availableBuildStatus)
}

func (gu *giteaUsecase) computeDuration(projectID string, duration time.Duration) time.Duration {
	value, ok := gu.timeRefByProject.Get(projectID)
	if !ok || value == nil {
		value = faker.GetRefTime()
		gu.timeRefByProject.Set(projectID, value)
	}

	return faker.ComputeDuration(value.(time.Time), duration)
}

func fakeListTile(tileType coreModels.TileType, label string, items ...coreModels.TileItem) *coreModels.Tile {
	tile := coreModels.NewTile(tileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = label
	tile.Status = coreModels.SuccessStatus
	tile.Metrics.Values = []string{fmt.Sprint(len(items))}
	tile.Items = items
	return tile
}

func (gu *giteaUsecase) PullRequests(params *models.ListParams) (*coreModels.Tile, error) {
	return fakeListTile(api.GiteaPullRequestsTileType, "Pull requests",
		coreModels.TileItem{Label: "#41 Add login form", Detail: "api · 5d"},
		coreModels.TileItem{Label: "#39 Faster search", Detail: "web · 3d"}), nil
}

func (gu *giteaUsecase) Issues(params *models.IssuesListParams) (*coreModels.Tile, error) {
	return fakeListTile(api.GiteaIssuesTileType, "Issues",
		coreModels.TileItem{Label: "#88 Crash on login", Detail: "mobile · 4d", Tag: "bug"},
		coreModels.TileItem{Label: "#90 Dark mode", Detail: "web · 2d"}), nil
}

func (gu *giteaUsecase) Actions(params *models.ListParams) (*coreModels.Tile, error) {
	return fakeListTile(api.GiteaActionsTileType, "Actions",
		coreModels.TileItem{Label: "api · build", Detail: "3m / 5m", Status: coreModels.RunningStatus, Progress: pointer.ToInt(60)},
		coreModels.TileItem{Label: "web · test", Detail: "2h", Status: coreModels.FailedStatus}), nil
}

func (gu *giteaUsecase) Milestones(params *models.ListParams) (*coreModels.Tile, error) {
	return fakeListTile(api.GiteaMilestonesTileType, "Milestones",
		coreModels.TileItem{Label: "api · v2.0", Detail: "due in 5 days · 60%", Status: coreModels.SuccessStatus}), nil
}

func (gu *giteaUsecase) Commits(params *models.ListParams) (*coreModels.Tile, error) {
	return fakeListTile(api.GiteaCommitsTileType, "Latest commits",
		coreModels.TileItem{Label: "feat: add login form", Detail: "api · alice · 2h"}), nil
}

func (gu *giteaUsecase) Repository(params *models.RepositoryParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaRepositoryTileType).WithMetrics(coreModels.NumberUnit)
	tile.Label = params.Repository
	tile.Status = coreModels.SuccessStatus
	tile.Metrics.Values = []string{"3", "5", "2", "8", "6", "9", "4", "7", "10", "6", "8", "5"}
	tile.Items = []coreModels.TileItem{{Label: "PR", Detail: "3"}, {Label: "Issues", Detail: "5"}, {Label: "Bugs", Detail: "1", Tag: "bug"}}
	tile.Message = "v2.0 · due in 5 days · 60%"
	return tile, nil
}

func (gu *giteaUsecase) RepositoriesGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error) {
	panic("unimplemented")
}
