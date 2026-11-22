//go:build faker

package gitea

import (
	"github.com/monitoror/monitoror/api/config/versions"
	"github.com/monitoror/monitoror/internal/pkg/monitorable"
	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api"
	giteaDelivery "github.com/monitoror/monitoror/monitorables/gitea/api/delivery/http"
	giteaModels "github.com/monitoror/monitoror/monitorables/gitea/api/models"
	giteaUsecase "github.com/monitoror/monitoror/monitorables/gitea/api/usecase"
	"github.com/monitoror/monitoror/registry"
	"github.com/monitoror/monitoror/store"
)

type Monitorable struct {
	monitorable.DefaultMonitorableFaker

	store *store.Store

	// Config tile settings
	countTileEnabler         registry.TileEnabler
	checksTileEnabler        registry.TileEnabler
	pullRequestTileEnabler   registry.TileEnabler
	contributionsTileEnabler registry.TileEnabler
	milestoneTileEnabler     registry.TileEnabler

	// Team board tiles
	pullRequestsTileEnabler registry.TileEnabler
	issuesTileEnabler       registry.TileEnabler
	actionsTileEnabler      registry.TileEnabler
	milestonesTileEnabler   registry.TileEnabler
	commitsTileEnabler      registry.TileEnabler
	repositoryTileEnabler   registry.TileEnabler
}

func NewMonitorable(store *store.Store) *Monitorable {
	m := &Monitorable{}
	m.store = store

	// Register Monitorable Tile in config manager
	m.countTileEnabler = store.Registry.RegisterTile(api.GiteaCountTileType, versions.Version2001, m.GetVariantsNames())
	m.checksTileEnabler = store.Registry.RegisterTile(api.GiteaChecksTileType, versions.Version2001, m.GetVariantsNames())
	m.pullRequestTileEnabler = store.Registry.RegisterTile(api.GiteaPullRequestTileType, versions.Version2001, m.GetVariantsNames())
	m.contributionsTileEnabler = store.Registry.RegisterTile(api.GiteaContributionsTileType, versions.Version2001, m.GetVariantsNames())
	m.milestoneTileEnabler = store.Registry.RegisterTile(api.GiteaMilestoneTileType, versions.Version2001, m.GetVariantsNames())
	m.pullRequestsTileEnabler = store.Registry.RegisterTile(api.GiteaPullRequestsTileType, versions.Version2001, m.GetVariantsNames())
	m.issuesTileEnabler = store.Registry.RegisterTile(api.GiteaIssuesTileType, versions.Version2001, m.GetVariantsNames())
	m.actionsTileEnabler = store.Registry.RegisterTile(api.GiteaActionsTileType, versions.Version2001, m.GetVariantsNames())
	m.milestonesTileEnabler = store.Registry.RegisterTile(api.GiteaMilestonesTileType, versions.Version2001, m.GetVariantsNames())
	m.commitsTileEnabler = store.Registry.RegisterTile(api.GiteaCommitsTileType, versions.Version2001, m.GetVariantsNames())
	m.repositoryTileEnabler = store.Registry.RegisterTile(api.GiteaRepositoryTileType, versions.Version2001, m.GetVariantsNames())

	return m
}

func (m *Monitorable) GetDisplayName() string {
	return "Gitea"
}

func (m *Monitorable) Enable(variantName coreModels.VariantName) {
	usecase := giteaUsecase.NewGiteaUsecase()
	delivery := giteaDelivery.NewGiteaDelivery(usecase)

	// EnableTile route to echo
	routeGroup := m.store.MonitorableRouter.Group("/gitea", variantName)
	routeCount := routeGroup.GET("/count", delivery.GetCount)
	routeChecks := routeGroup.GET("/checks", delivery.GetChecks)
	routePullRequest := routeGroup.GET("/pullrequest", delivery.GetPullRequest)
	routeContributions := routeGroup.GET("/contributions", delivery.GetContributions)
	routeMilestone := routeGroup.GET("/milestone", delivery.GetMilestone)
	routePullRequests := routeGroup.GET("/pullrequests", delivery.GetPullRequests)
	routeIssues := routeGroup.GET("/issues", delivery.GetIssues)
	routeActions := routeGroup.GET("/actions", delivery.GetActions)
	routeMilestones := routeGroup.GET("/milestones", delivery.GetMilestones)
	routeCommits := routeGroup.GET("/commits", delivery.GetCommits)
	routeRepository := routeGroup.GET("/repository", delivery.GetRepository)

	// EnableTile data for config hydration
	m.countTileEnabler.Enable(variantName, &giteaModels.CountParams{}, routeCount.Path)
	m.checksTileEnabler.Enable(variantName, &giteaModels.ChecksParams{}, routeChecks.Path)
	m.pullRequestTileEnabler.Enable(variantName, &giteaModels.PullRequestParams{}, routePullRequest.Path)
	m.contributionsTileEnabler.Enable(variantName, &giteaModels.ContributionsParams{}, routeContributions.Path)
	m.milestoneTileEnabler.Enable(variantName, &giteaModels.MilestoneParams{}, routeMilestone.Path)
	m.pullRequestsTileEnabler.Enable(variantName, &giteaModels.ListParams{}, routePullRequests.Path)
	m.issuesTileEnabler.Enable(variantName, &giteaModels.IssuesListParams{}, routeIssues.Path)
	m.actionsTileEnabler.Enable(variantName, &giteaModels.ListParams{}, routeActions.Path)
	m.milestonesTileEnabler.Enable(variantName, &giteaModels.ListParams{}, routeMilestones.Path)
	m.commitsTileEnabler.Enable(variantName, &giteaModels.ListParams{}, routeCommits.Path)
	m.repositoryTileEnabler.Enable(variantName, &giteaModels.RepositoryParams{}, routeRepository.Path)
}
