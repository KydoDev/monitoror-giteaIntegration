package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeamBoardParams_Validate(t *testing.T) {
	assert.Empty(t, (&ListParams{Owner: "acme-team_1.0"}).Validate())
	assert.Len(t, (&ListParams{Owner: ".."}).Validate(), 1)
	assert.Len(t, (&IssuesListParams{Owner: "a/b"}).Validate(), 1)
	assert.Len(t, (&RepositoryGeneratorParams{Owner: "."}).Validate(), 1)
	assert.Empty(t, (&RepositoryParams{Owner: "acme", Repository: "shop", Ref: "feature/login"}).Validate())
	assert.Len(t, (&RepositoryParams{Owner: "acme", Repository: "..", Ref: "../x"}).Validate(), 2)
	assert.Empty(t, (&RepositoryParams{}).Validate(), "empty names are left to the required tags")
	assert.Len(t, (&ContributionsParams{Organization: "../admin"}).Validate(), 1)
}
