package models

import (
	"testing"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/test"
)

func TestChecksGeneratorParams_Validate(t *testing.T) {
	param := &ChecksGeneratorParams{Owner: "test"}
	test.AssertParams(t, param, 0)

	param = &ChecksGeneratorParams{}
	test.AssertParams(t, param, 1)
}
