package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/test"
)

func TestContributionsParams_Validate(t *testing.T) {
	param := &ContributionsParams{User: "test"}
	test.AssertParams(t, param, 0)

	param = &ContributionsParams{User: "test", Weeks: 52}
	test.AssertParams(t, param, 0)

	param = &ContributionsParams{User: "test", Weeks: 53}
	test.AssertParams(t, param, 1)

	param = &ContributionsParams{User: "test", Weeks: -1}
	test.AssertParams(t, param, 1)

	param = &ContributionsParams{}
	test.AssertParams(t, param, 1)
}

func TestContributionsParams_String(t *testing.T) {
	param := &ContributionsParams{User: "test", Weeks: 12}
	assert.Equal(t, "CONTRIBUTIONS-test-12", fmt.Sprint(param))
}
