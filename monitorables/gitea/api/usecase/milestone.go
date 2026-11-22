//go:build !faker

package usecase

import (
	"fmt"
	"math"
	"sort"
	"time"

	uiConfigModels "github.com/monitoror/monitoror/api/config/models"
	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/gitea/api"
	"github.com/monitoror/monitoror/monitorables/gitea/api/models"
)

// milestoneWarningDays: a milestone due within this many days with open issues is shown as a warning
const milestoneWarningDays = 3

// timeNow is replaced in tests
var timeNow = time.Now

func (gu *giteaUsecase) Milestone(params *models.MilestoneParams) (*coreModels.Tile, error) {
	tile := coreModels.NewTile(api.GiteaMilestoneTileType)
	tile.Label = params.Repository

	milestones, err := gu.repository.GetMilestones(params.Owner, params.Repository)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Tile: tile, Message: "unable to load milestones"}
	}

	milestone := nextMilestone(milestones)
	if milestone == nil {
		return nil, &coreModels.MonitororError{Tile: tile, Message: "no open milestone", ErrorStatus: coreModels.UnknownStatus}
	}

	now := timeNow()
	tile.Status = coreModels.SuccessStatus
	tile.Message = milestone.Title

	if milestone.DueOn != nil {
		days := daysUntil(now, *milestone.DueOn)
		tile.Message = fmt.Sprintf("%s · %s", milestone.Title, humanizeDue(days))

		if milestone.OpenIssues > 0 {
			if days < 0 {
				tile.Status = coreModels.FailedStatus
			} else if days <= milestoneWarningDays {
				tile.Status = coreModels.WarningStatus
			}
		}
	}

	// Progress: closed issues out of all issues of the milestone
	if total := milestone.OpenIssues + milestone.ClosedIssues; total > 0 {
		tile.WithMetrics(coreModels.RawUnit)
		tile.Metrics.Values = []string{fmt.Sprintf("%d%%", milestone.ClosedIssues*100/total)}
	}

	return tile, nil
}

func (gu *giteaUsecase) MilestonesGenerator(params interface{}) ([]uiConfigModels.GeneratedTile, error) {
	generatorParams := params.(*models.MilestoneGeneratorParams)

	repositories, err := gu.repository.GetRepositories(generatorParams.Owner)
	if err != nil {
		return nil, &coreModels.MonitororError{Err: err, Message: "unable to load repositories"}
	}

	var results []uiConfigModels.GeneratedTile
	for _, repository := range repositories {
		if repository.Archived {
			continue
		}

		// Only repositories with an open milestone. Milestones may be unavailable (issues disabled): skip.
		milestones, err := gu.repository.GetMilestones(generatorParams.Owner, repository.Name)
		if err != nil || len(milestones) == 0 {
			continue
		}

		results = append(results, uiConfigModels.GeneratedTile{
			Label: repository.Name,
			Params: &models.MilestoneParams{
				Owner:      generatorParams.Owner,
				Repository: repository.Name,
			},
		})
	}

	return results, nil
}

// nextMilestone returns the open milestone due first; milestones without due date come last, oldest first.
func nextMilestone(milestones []models.Milestone) *models.Milestone {
	if len(milestones) == 0 {
		return nil
	}

	sorted := make([]models.Milestone, len(milestones))
	copy(sorted, milestones)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i].DueOn, sorted[j].DueOn
		switch {
		case a != nil && b != nil && !a.Equal(*b):
			return a.Before(*b)
		case a != nil && b == nil:
			return true
		case a == nil && b != nil:
			return false
		}
		return sorted[i].ID < sorted[j].ID
	})

	return &sorted[0]
}

// daysUntil counts calendar days from now to due, in the location of now (TZ)
func daysUntil(now, due time.Time) int {
	due = due.In(now.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, now.Location())

	// Round: a day can last 23 or 25 hours on DST changes
	return int(math.Round(dueDay.Sub(today).Hours() / 24))
}

func humanizeDue(days int) string {
	switch {
	case days == 0:
		return "due today"
	case days == 1:
		return "due tomorrow"
	case days > 1:
		return fmt.Sprintf("due in %d days", days)
	case days == -1:
		return "1 day overdue"
	default:
		return fmt.Sprintf("%d days overdue", -days)
	}
}
