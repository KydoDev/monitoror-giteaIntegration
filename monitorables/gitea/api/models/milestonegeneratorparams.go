package models

import (
	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
)

type MilestoneGeneratorParams struct {
	params.Default

	Owner string `json:"owner" query:"owner" validate:"required"`
}
