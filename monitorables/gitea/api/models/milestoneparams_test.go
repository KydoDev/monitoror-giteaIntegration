package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/test"
)

func TestMilestoneParams_Validate(t *testing.T) {
	param := &MilestoneParams{Owner: "test", Repository: "test"}
	test.AssertParams(t, param, 0)

	param = &MilestoneParams{Owner: "test"}
	test.AssertParams(t, param, 1)

	param = &MilestoneParams{}
	test.AssertParams(t, param, 2)

	assert.Equal(t, "MILESTONE-test-repo", fmt.Sprint(&MilestoneParams{Owner: "test", Repository: "repo"}))
}

func TestMilestoneGeneratorParams_Validate(t *testing.T) {
	param := &MilestoneGeneratorParams{Owner: "test"}
	test.AssertParams(t, param, 0)

	param = &MilestoneGeneratorParams{}
	test.AssertParams(t, param, 1)
}
