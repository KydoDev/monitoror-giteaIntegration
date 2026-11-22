package models

import (
	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
)

type ChecksGeneratorParams struct {
	params.Default

	Owner string `json:"owner" query:"owner" validate:"required"`
}
