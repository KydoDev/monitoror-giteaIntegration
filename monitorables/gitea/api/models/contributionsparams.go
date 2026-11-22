package models

import (
	"fmt"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
)

type (
	ContributionsParams struct {
		params.Default

		User  string `json:"user" query:"user" validate:"required"`
		Weeks int    `json:"weeks,omitempty" query:"weeks" validate:"omitempty,gte=1,lte=52"`
	}
)

// Used by cache as identifier
func (p *ContributionsParams) String() string {
	return fmt.Sprintf("CONTRIBUTIONS-%s-%d", p.User, p.Weeks)
}
