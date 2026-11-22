//go:build !faker

package usecase

import (
	"errors"
	"testing"
	"time"

	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api"
	"github.com/monitoror/monitoror/monitorables/gitea/api/mocks"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"

	. "github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	. "github.com/stretchr/testify/mock"
)

var (
	boardNow = time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC)
	notFound = &models.StatusError{StatusCode: 404, Body: "Not Found"}
)

func newBoardUsecase(repository *mocks.Repository) *giteaUsecase {
	gu := NewGiteaUsecase(repository).(*giteaUsecase)
	gu.now = func() time.Time { return boardNow }
	return gu
}

func TestPullRequests_OldestFirst(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("SearchIssues", "acme", "pulls").Return([]models.Issue{
		{Number: 2, Title: "new", Repository: "shop", CreatedAt: boardNow.Add(-2 * time.Hour)},
		{Number: 1, Title: "old", Repository: "mobile", CreatedAt: boardNow.Add(-72 * time.Hour)},
		{Number: 3, Title: "newest", Repository: "shop", CreatedAt: boardNow.Add(-10 * time.Minute)},
	}, nil)

	tile, err := newBoardUsecase(mockRepository).PullRequests(&models.ListParams{Owner: "acme", Limit: 2})
	if assert.NoError(t, err) {
		assert.Equal(t, api.GiteaPullRequestsTileType, tile.Type)
		assert.Equal(t, coreModels.SuccessStatus, tile.Status)
		assert.Equal(t, []string{"3"}, tile.Metrics.Values)
		assert.Equal(t, []coreModels.TileItem{
			{Label: "#1 old", Detail: "mobile · 3d"},
			{Label: "#2 new", Detail: "shop · 2h"},
		}, tile.Items)
	}
}

func TestPullRequests_Error(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("SearchIssues", "acme", "pulls").Return(nil, errors.New("boom"))

	_, err := newBoardUsecase(mockRepository).PullRequests(&models.ListParams{Owner: "acme"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load pull requests", err.Error())
	}
}

func TestIssues_BugsFirst(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("SearchIssues", "acme", "issues").Return([]models.Issue{
		{Number: 1, Title: "old feature", Repository: "shop", CreatedAt: boardNow.Add(-48 * time.Hour)},
		{Number: 2, Title: "crash", Repository: "mobile", Labels: []string{"Bug"}, CreatedAt: boardNow.Add(-time.Hour)},
	}, nil)

	tile, err := newBoardUsecase(mockRepository).Issues(&models.IssuesListParams{Owner: "acme"})
	if assert.NoError(t, err) {
		assert.Equal(t, coreModels.WarningStatus, tile.Status)
		assert.Equal(t, "1 bug", tile.Message)
		assert.Equal(t, []string{"2"}, tile.Metrics.Values)
		assert.Equal(t, []coreModels.TileItem{
			{Label: "#2 crash", Detail: "mobile · 1h", Tag: "bug"},
			{Label: "#1 old feature", Detail: "shop · 2d"},
		}, tile.Items)
	}
}

func TestIssues_CustomBugLabelAndError(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("SearchIssues", "acme", "issues").Return([]models.Issue{
		{Number: 1, Title: "a", Labels: []string{"bug"}},
	}, nil).Once()
	mockRepository.On("SearchIssues", "acme", "issues").Return(nil, errors.New("boom"))

	tile, err := newBoardUsecase(mockRepository).Issues(&models.IssuesListParams{Owner: "acme", BugLabel: "kind/bug"})
	if assert.NoError(t, err) {
		assert.Equal(t, coreModels.SuccessStatus, tile.Status)
		assert.Equal(t, "0 bugs", tile.Message)
	}

	// A new usecase has an empty cache
	_, err = newBoardUsecase(mockRepository).Issues(&models.IssuesListParams{Owner: "acme"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load issues", err.Error())
	}
}

func TestActions_LatestPerWorkflow(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOwnerRuns", "acme").Return([]models.Run{
		{ID: 2, Repository: "shop", Workflow: "ci.yml", Title: "feat: cart", Status: "in_progress",
			CreatedAt: boardNow.Add(-2 * time.Minute), StartedAt: boardNow.Add(-2 * time.Minute)},
		{ID: 1, Repository: "shop", Workflow: "ci.yml", Status: "completed", Conclusion: "success",
			CreatedAt: boardNow.Add(-time.Hour), StartedAt: boardNow.Add(-time.Hour), CompletedAt: boardNow.Add(-56 * time.Minute)},
		{Repository: "infra", Workflow: "deploy.yml", Status: "completed", Conclusion: "failure",
			CreatedAt: boardNow.Add(-3 * time.Hour), CompletedAt: boardNow.Add(-3 * time.Hour)},
		{Repository: "docs", Workflow: "pages.yml", Status: "completed", Conclusion: "success",
			CreatedAt: boardNow.Add(-5 * time.Hour), CompletedAt: boardNow.Add(-5 * time.Hour)},
		{Repository: "mobile", Workflow: "ci.yml", Status: "queued", CreatedAt: boardNow.Add(-time.Minute)},
	}, nil)

	tile, err := newBoardUsecase(mockRepository).Actions(&models.ListParams{Owner: "acme", Limit: 3})
	if assert.NoError(t, err) {
		assert.Equal(t, coreModels.FailedStatus, tile.Status)
		assert.Equal(t, []string{"1"}, tile.Metrics.Values)
		assert.Equal(t, []coreModels.TileItem{
			{Label: "shop · feat: cart", Detail: "2m / 4m", Status: coreModels.RunningStatus, Progress: ToInt(50)},
			{Label: "mobile · ci.yml", Detail: "queued", Status: coreModels.QueuedStatus},
			{Label: "infra · deploy.yml", Detail: "3h", Status: coreModels.FailedStatus},
		}, tile.Items)
	}
}

func TestActions_UserFallsBackToRepositories(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOwnerRuns", "alice").Return(nil, notFound)
	mockRepository.On("GetRepositories", "alice").Return([]models.Repository{
		{Name: "blog"}, {Name: "old", Archived: true}, {Name: "broken"},
	}, nil)
	mockRepository.On("GetRuns", "alice", "blog", "").Return([]models.Run{
		{Workflow: "ci.yml", Status: "in_progress", CreatedAt: boardNow, StartedAt: boardNow.Add(-30 * time.Second)},
	}, nil)
	mockRepository.On("GetRuns", "alice", "broken", "").Return(nil, errors.New("boom"))

	gu := newBoardUsecase(mockRepository)
	tile, err := gu.Actions(&models.ListParams{Owner: "alice"})
	if assert.NoError(t, err) {
		assert.Equal(t, coreModels.SuccessStatus, tile.Status)
		assert.Equal(t, []coreModels.TileItem{
			{Label: "blog · ci.yml", Detail: "30s", Status: coreModels.RunningStatus},
		}, tile.Items)
		mockRepository.AssertNotCalled(t, "GetRuns", "alice", "old", "")
	}

	// The runs are cached; once expired, the owner-wide endpoint is not tried again for a while
	_, _ = gu.Actions(&models.ListParams{Owner: "alice"})
	mockRepository.AssertNumberOfCalls(t, "GetRuns", 2)
	gu.now = func() time.Time { return boardNow.Add(runsTTL) }
	_, _ = gu.Actions(&models.ListParams{Owner: "alice"})
	mockRepository.AssertNumberOfCalls(t, "GetOwnerRuns", 1)
	mockRepository.AssertNumberOfCalls(t, "GetRuns", 4)
}

func TestActions_Error(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOwnerRuns", "acme").Return(nil, notFound)
	mockRepository.On("GetRepositories", "acme").Return(nil, errors.New("boom"))

	_, err := newBoardUsecase(mockRepository).Actions(&models.ListParams{Owner: "acme"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load actions", err.Error())
	}
}

func TestMilestones_DueFirst(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetRepositories", "acme").Return([]models.Repository{
		{Name: "shop"}, {Name: "infra"}, {Name: "old", Archived: true}, {Name: "broken"},
	}, nil)
	mockRepository.On("GetMilestones", "acme", "shop").Return([]models.Milestone{
		{Title: "v2.0", OpenIssues: 1, ClosedIssues: 3, DueOn: ToTime(boardNow.AddDate(0, 0, 10))},
		{Title: "Someday"},
	}, nil)
	mockRepository.On("GetMilestones", "acme", "infra").Return([]models.Milestone{
		{Title: "Cluster", OpenIssues: 2, DueOn: ToTime(boardNow.AddDate(0, 0, -2))},
	}, nil)
	mockRepository.On("GetMilestones", "acme", "broken").Return(nil, errors.New("boom"))

	tile, err := newBoardUsecase(mockRepository).Milestones(&models.ListParams{Owner: "acme"})
	if assert.NoError(t, err) {
		assert.Equal(t, coreModels.FailedStatus, tile.Status)
		assert.Equal(t, []string{"3"}, tile.Metrics.Values)
		if assert.Len(t, tile.Items, 3) {
			assert.Equal(t, "infra · Cluster", tile.Items[0].Label)
			assert.Equal(t, coreModels.FailedStatus, tile.Items[0].Status)
			assert.Equal(t, "shop · v2.0", tile.Items[1].Label)
			assert.Equal(t, "shop · Someday", tile.Items[2].Label)
			assert.Equal(t, "", tile.Items[2].Detail)
		}
		mockRepository.AssertNotCalled(t, "GetMilestones", "acme", "old")
	}
}

func TestMilestones_Error(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetRepositories", "acme").Return(nil, errors.New("boom"))

	_, err := newBoardUsecase(mockRepository).Milestones(&models.ListParams{Owner: "acme"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load repositories", err.Error())
	}
}

func TestCommits(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetFeedCommits", "acme", defaultListLimit).Return([]models.FeedCommit{
		{Repository: "shop", Author: "Alice", Message: "feat: cart", PushedAt: boardNow.Add(-5 * time.Minute)},
	}, nil).Once()
	mockRepository.On("GetFeedCommits", "acme", defaultListLimit).Return(nil, errors.New("boom"))
	gu := newBoardUsecase(mockRepository)

	tile, err := gu.Commits(&models.ListParams{Owner: "acme"})
	if assert.NoError(t, err) {
		assert.Equal(t, []coreModels.TileItem{{Label: "feat: cart", Detail: "shop · Alice · 5m"}}, tile.Items)
	}

	_, err = gu.Commits(&models.ListParams{Owner: "acme"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load commits", err.Error())
	}
}

func TestRepository_WithoutCI(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetRepositories", "acme").Return([]models.Repository{{Name: "mobile", DefaultBranch: "develop"}}, nil)
	mockRepository.On("GetChecks", "acme", "mobile", "develop").Return(&models.Checks{}, nil)
	mockRepository.On("GetRuns", "acme", "mobile", "develop").Return(nil, nil)
	mockRepository.On("GetCommits", "acme", "mobile", "develop", boardNow.AddDate(0, 0, -7*activityWeeks)).
		Return([]models.RepositoryCommit{
			{Date: boardNow.Add(-time.Hour)},
			{Date: boardNow.Add(-26 * time.Hour)},
			{Date: boardNow.AddDate(0, 0, -8)},
			{Date: boardNow.AddDate(0, 0, -100)},
		}, nil)
	mockRepository.On("GetCount", &models.CountParams{Owner: "acme", Repository: "mobile", Type: "pulls", State: "open"}).Return(2, nil)
	mockRepository.On("GetRepositoryIssues", "acme", "mobile").Return([]models.Issue{
		{Labels: []string{"bug"}}, {Labels: []string{"ui"}}, {},
	}, nil)
	mockRepository.On("GetMilestones", "acme", "mobile").Return([]models.Milestone{
		{Title: "Sprint 47", OpenIssues: 1, ClosedIssues: 1, DueOn: ToTime(boardNow.AddDate(0, 0, 3))},
	}, nil)
	gu := newBoardUsecase(mockRepository)

	tile, err := gu.Repository(&models.RepositoryParams{Owner: "acme", Repository: "mobile"})
	if assert.NoError(t, err) {
		assert.Equal(t, api.GiteaRepositoryTileType, tile.Type)
		assert.Equal(t, "mobile", tile.Label)
		assert.Equal(t, coreModels.SuccessStatus, tile.Status)
		assert.Equal(t, []string{"0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "1", "2"}, tile.Metrics.Values)
		assert.Equal(t, []coreModels.TileItem{
			{Label: "PR", Detail: "2"},
			{Label: "Issues", Detail: "3"},
			{Label: "Bugs", Detail: "1", Tag: "bug"},
		}, tile.Items)
		assert.Equal(t, "Sprint 47 · "+humanizeDue(3)+" · 50%", tile.Message)
	}

	// The summary is cached
	_, err = gu.Repository(&models.RepositoryParams{Owner: "acme", Repository: "mobile"})
	if assert.NoError(t, err) {
		mockRepository.AssertNumberOfCalls(t, "GetCommits", 1)
		mockRepository.AssertNumberOfCalls(t, "GetRepositories", 1)
	}
}

func TestRepository_PartialSummary(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetChecks", "acme", "shop", "main").Return(&models.Checks{}, nil)
	mockRepository.On("GetRuns", "acme", "shop", "main").Return(nil, errors.New("no actions"))
	mockRepository.On("GetCommits", "acme", "shop", "main", AnythingOfType("time.Time")).Return(nil, errors.New("boom"))
	mockRepository.On("GetCount", AnythingOfType("*models.CountParams")).Return(0, errors.New("boom"))
	mockRepository.On("GetRepositoryIssues", "acme", "shop").Return(nil, errors.New("boom"))
	mockRepository.On("GetMilestones", "acme", "shop").Return(nil, errors.New("boom"))

	tile, err := newBoardUsecase(mockRepository).Repository(&models.RepositoryParams{Owner: "acme", Repository: "shop", Ref: "main"})
	if assert.NoError(t, err) {
		assert.Nil(t, tile.Metrics)
		assert.Empty(t, tile.Items)
		assert.Empty(t, tile.Message)
		mockRepository.AssertNotCalled(t, "GetRepositories", AnythingOfType("string"))
	}
}

func TestRepository_ChecksError(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetRepositories", "acme").Return(nil, errors.New("boom"))
	mockRepository.On("GetChecks", "acme", "shop", "main").Return(nil, errors.New("boom"))

	_, err := newBoardUsecase(mockRepository).Repository(&models.RepositoryParams{Owner: "acme", Repository: "shop"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load ref checks", err.Error())
	}
}

func TestRepositoriesGenerator(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetRepositories", "acme").Return([]models.Repository{
		{Name: "shop", DefaultBranch: "main"}, {Name: "old", Archived: true}, {Name: "new", Empty: true},
	}, nil).Once()
	mockRepository.On("GetRepositories", "other").Return(nil, errors.New("boom"))
	gu := newBoardUsecase(mockRepository)

	tiles, err := gu.RepositoriesGenerator(&models.RepositoryGeneratorParams{Owner: "acme", BugLabel: "defect"})
	if assert.NoError(t, err) && assert.Len(t, tiles, 1) {
		assert.Equal(t, "shop", tiles[0].Label)
		assert.Equal(t, &models.RepositoryParams{Owner: "acme", Repository: "shop", Ref: "main", BugLabel: "defect"}, tiles[0].Params)
	}

	_, err = gu.RepositoriesGenerator(&models.RepositoryGeneratorParams{Owner: "other"})
	assert.Error(t, err)
}

func TestContributions_Organization(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOrganizationMembers", "acme").Return([]string{"alice", "bob"}, nil)
	mockRepository.On("GetContributions", "alice").Return([]models.Contribution{{Time: boardNow, Count: 2}}, nil)
	mockRepository.On("GetContributions", "bob").Return([]models.Contribution{{Time: boardNow, Count: 3}}, nil)

	tile, err := newBoardUsecase(mockRepository).Contributions(&models.ContributionsParams{Organization: "acme", Weeks: 1})
	if assert.NoError(t, err) {
		assert.Equal(t, "acme", tile.Label)
		assert.Contains(t, tile.Metrics.Values, "5")
	}
}

func TestContributions_OrganizationError(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOrganizationMembers", "acme").Return(nil, errors.New("boom"))

	_, err := newBoardUsecase(mockRepository).Contributions(&models.ContributionsParams{Organization: "acme"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load organization members", err.Error())
	}
}

func TestTTLCache(t *testing.T) {
	cache := newTTLCache()
	loads := 0
	load := func() (interface{}, error) { loads++; return loads, nil }

	value, _ := cache.get("k", time.Minute, boardNow, load)
	assert.Equal(t, 1, value)
	value, _ = cache.get("k", time.Minute, boardNow.Add(30*time.Second), load)
	assert.Equal(t, 1, value)
	value, _ = cache.get("k", time.Minute, boardNow.Add(time.Minute), load)
	assert.Equal(t, 2, value)

	_, err := cache.get("e", time.Minute, boardNow, func() (interface{}, error) { return nil, errors.New("boom") })
	assert.Error(t, err)
	value, _ = cache.get("e", time.Minute, boardNow, load)
	assert.Equal(t, 3, value, "errors are not cached")

	var nilCache *ttlCache
	value, _ = nilCache.get("k", time.Minute, boardNow, load)
	assert.Equal(t, 4, value)
}

func TestTeamBoardHelpers(t *testing.T) {
	assert.Equal(t, "now", humanizeAge(boardNow, boardNow.Add(-30*time.Second)))
	assert.Equal(t, "12m", humanizeAge(boardNow, boardNow.Add(-12*time.Minute)))
	assert.Equal(t, "45s", humanizeDuration(45*time.Second))
	assert.Equal(t, "1h05", humanizeDuration(65*time.Minute))
	assert.Equal(t, "2 bugs", pluralize(2, "bug"))
	assert.Equal(t, coreModels.WarningStatus, worstStatus(coreModels.SuccessStatus, coreModels.WarningStatus))
	assert.Equal(t, coreModels.FailedStatus, worstStatus(coreModels.FailedStatus, coreModels.WarningStatus))
	assert.Equal(t, coreModels.SuccessStatus, milestoneStatus(boardNow, models.Milestone{DueOn: ToTime(boardNow.AddDate(0, 0, -1))}))
	assert.Equal(t, coreModels.WarningStatus, milestoneStatus(boardNow, models.Milestone{OpenIssues: 1, DueOn: ToTime(boardNow.AddDate(0, 0, 2))}))
	assert.Equal(t, coreModels.SuccessStatus, milestoneStatus(boardNow, models.Milestone{OpenIssues: 1, DueOn: ToTime(boardNow.AddDate(0, 0, 20))}))
	assert.Equal(t, "main", newBoardUsecase(func() *mocks.Repository {
		m := new(mocks.Repository)
		m.On("GetRepositories", "x").Return([]models.Repository{{Name: "y"}}, nil)
		return m
	}()).defaultBranch("x", "y"))
}

func TestActions_OwnerErrorIsNotAFallback(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOwnerRuns", "acme").Return(nil, &models.StatusError{StatusCode: 502})

	_, err := newBoardUsecase(mockRepository).Actions(&models.ListParams{Owner: "acme"})
	if assert.Error(t, err) {
		mockRepository.AssertNotCalled(t, "GetRepositories", AnythingOfType("string"))
	}
}

func TestActions_RunningWithoutStart(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOwnerRuns", "acme").Return([]models.Run{
		{ID: 1, Repository: "shop", Workflow: "ci.yml", Status: "in_progress", CreatedAt: boardNow},
		{ID: 2, Repository: "api", Workflow: "ci.yml", Status: "in_progress", CreatedAt: boardNow, StartedAt: boardNow.Add(time.Minute)},
	}, nil)

	tile, err := newBoardUsecase(mockRepository).Actions(&models.ListParams{Owner: "acme"})
	if assert.NoError(t, err) && assert.Len(t, tile.Items, 2) {
		assert.Equal(t, "running", tile.Items[0].Detail)
		assert.Nil(t, tile.Items[0].Progress)
		assert.Equal(t, "0s", tile.Items[1].Detail)
	}
}

func TestPullRequests_CachedSearchIsNotMutated(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("SearchIssues", "acme", "pulls").Return([]models.Issue{
		{Number: 2, CreatedAt: boardNow},
		{Number: 1, CreatedAt: boardNow.Add(-time.Hour)},
	}, nil)
	gu := newBoardUsecase(mockRepository)

	_, _ = gu.PullRequests(&models.ListParams{Owner: "acme"})
	issues, err := gu.searchIssues("acme", "pulls")
	if assert.NoError(t, err) {
		assert.Equal(t, 2, issues[0].Number, "the cached slice keeps the order of Gitea")
		mockRepository.AssertNumberOfCalls(t, "SearchIssues", 1)
	}
}

func TestRepository_BugLabelIsNotInTheCacheKey(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetChecks", "acme", "shop", "main").Return(&models.Checks{}, nil)
	mockRepository.On("GetRuns", "acme", "shop", "main").Return(nil, nil)
	mockRepository.On("GetCommits", "acme", "shop", "main", AnythingOfType("time.Time")).Return(nil, nil)
	mockRepository.On("GetCount", AnythingOfType("*models.CountParams")).Return(0, nil)
	mockRepository.On("GetRepositoryIssues", "acme", "shop").Return([]models.Issue{{Labels: []string{"defect"}}}, nil)
	mockRepository.On("GetMilestones", "acme", "shop").Return(nil, nil)
	gu := newBoardUsecase(mockRepository)

	tile, _ := gu.Repository(&models.RepositoryParams{Owner: "acme", Repository: "shop", Ref: "main"})
	assert.Len(t, tile.Items, 2)
	tile, _ = gu.Repository(&models.RepositoryParams{Owner: "acme", Repository: "shop", Ref: "main", BugLabel: "defect"})
	assert.Equal(t, coreModels.TileItem{Label: "Bugs", Detail: "1", Tag: "bug"}, tile.Items[2])
	mockRepository.AssertNumberOfCalls(t, "GetRepositoryIssues", 1)
}

func TestRepository_ErrorTileType(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetChecks", "acme", "shop", "main").Return(nil, errors.New("boom"))

	_, err := newBoardUsecase(mockRepository).Repository(&models.RepositoryParams{Owner: "acme", Repository: "shop", Ref: "main"})
	var monitororError *coreModels.MonitororError
	if assert.ErrorAs(t, err, &monitororError) {
		assert.Equal(t, api.GiteaRepositoryTileType, monitororError.Tile.Type)
	}
}

func TestContributions_OrganizationSkipsFailingMembers(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOrganizationMembers", "acme").Return([]string{"alice", "ghost"}, nil)
	mockRepository.On("GetContributions", "alice").Return([]models.Contribution{{Time: boardNow, Count: 2}}, nil)
	mockRepository.On("GetContributions", "ghost").Return(nil, notFound)
	gu := newBoardUsecase(mockRepository)

	tile, err := gu.Contributions(&models.ContributionsParams{Organization: "acme", Weeks: 1})
	if assert.NoError(t, err) {
		assert.Contains(t, tile.Metrics.Values, "2")
	}

	// Cached
	_, _ = gu.Contributions(&models.ContributionsParams{Organization: "acme", Weeks: 1})
	mockRepository.AssertNumberOfCalls(t, "GetOrganizationMembers", 1)
}

func TestContributions_OrganizationAllMembersFail(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetOrganizationMembers", "acme").Return([]string{"ghost"}, nil)
	mockRepository.On("GetContributions", "ghost").Return(nil, notFound)

	_, err := newBoardUsecase(mockRepository).Contributions(&models.ContributionsParams{Organization: "acme"})
	if assert.Error(t, err) {
		assert.Equal(t, "unable to load contributions", err.Error())
	}
}

func TestTTLCache_Bounded(t *testing.T) {
	cache := newTTLCache()
	load := func() (interface{}, error) { return 1, nil }

	for i := 0; i < maxEntries; i++ {
		_, _ = cache.get(cacheKey("k", string(rune('a'+i%26)), time.Duration(i).String()), time.Minute, boardNow, load)
	}
	assert.Len(t, cache.entries, maxEntries)

	// Expired entries are removed first
	_, _ = cache.get("new", time.Minute, boardNow.Add(time.Hour), load)
	assert.Len(t, cache.entries, 1)

	// Without expired entries, the cache starts over
	for i := 0; i < maxEntries; i++ {
		_, _ = cache.get(cacheKey("x", time.Duration(i).String()), time.Hour, boardNow.Add(time.Hour), load)
	}
	assert.LessOrEqual(t, len(cache.entries), maxEntries)
}
