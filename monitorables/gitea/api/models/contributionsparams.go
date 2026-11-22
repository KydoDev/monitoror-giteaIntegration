package models

import (
	"fmt"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
	"github.com/monitoror/monitoror/internal/pkg/validator"
)

type (
	ContributionsParams struct {
		params.Default

		User         string `json:"user,omitempty" query:"user"`
		Organization string `json:"organization,omitempty" query:"organization"` // sum of the members contributions
		Weeks        int    `json:"weeks,omitempty" query:"weeks" validate:"omitempty,gte=1,lte=52"`
	}
)

// Used by cache as identifier
func (p *ContributionsParams) String() string {
	return fmt.Sprintf("CONTRIBUTIONS-%s-%s-%d", p.User, p.Organization, p.Weeks)
}

// Validate requires a user or an organization
func (p *ContributionsParams) Validate() []validator.Error {
	if (p.User == "") == (p.Organization == "") {
		return []validator.Error{validator.NewDefaultError("User", "set, or organization set (not both)")}
	}
	return validateNames(map[string]string{"User": p.User, "Organization": p.Organization})
}
