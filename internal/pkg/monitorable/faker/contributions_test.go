package faker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContributions(t *testing.T) {
	values := Contributions("john", 4)

	assert.GreaterOrEqual(t, len(values), 3*7+1)
	assert.LessOrEqual(t, len(values), 4*7)
	assert.Equal(t, values, Contributions("john", 4))
}
