package models

import (
	"fmt"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
)

type MilestoneParams struct {
	params.Default

	Owner      string `json:"owner" query:"owner" validate:"required"`
	Repository string `json:"repository" query:"repository" validate:"required"`
}

// Used by cache as identifier
func (p *MilestoneParams) String() string {
	return fmt.Sprintf("MILESTONE-%s-%s", p.Owner, p.Repository)
}
