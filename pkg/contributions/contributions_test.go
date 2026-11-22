package contributions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDays(t *testing.T) {
	// Wednesday
	now := time.Date(2026, time.November, 18, 15, 4, 5, 0, time.UTC)

	days := Days(now, 2)

	// Last week from Sunday 8 + current week from Sunday 15 to Wednesday 18
	assert.Len(t, days, 7+4)
	assert.Equal(t, time.Date(2026, time.November, 8, 0, 0, 0, 0, time.UTC), days[0])
	assert.Equal(t, time.Sunday, days[0].Weekday())
	assert.Equal(t, time.Date(2026, time.November, 18, 0, 0, 0, 0, time.UTC), days[len(days)-1])
}

func TestDays_Sunday(t *testing.T) {
	now := time.Date(2026, time.November, 22, 0, 0, 0, 0, time.UTC)

	days := Days(now, 1)

	assert.Len(t, days, 1)
	assert.Equal(t, "2026-11-22", DayKey(days[0]))
}

func TestDays_Bounds(t *testing.T) {
	now := time.Date(2026, time.November, 21, 12, 0, 0, 0, time.UTC) // Saturday

	assert.Len(t, Days(now, 0), 7*DefaultWeeks)
	assert.Len(t, Days(now, -3), 7*DefaultWeeks)
	assert.Len(t, Days(now, 100), 7*MaxWeeks)
}

func TestDays_DaylightSaving(t *testing.T) {
	location, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Skip("time zone database not available")
	}

	// Clocks go back on Sunday 25 October 2026 (a 25 hours day)
	now := time.Date(2026, time.October, 28, 9, 0, 0, 0, location)

	days := Days(now, 2)

	assert.Len(t, days, 7+4)
	for i := 1; i < len(days); i++ {
		assert.Equal(t, 0, days[i].Hour())
		assert.Equal(t, days[i-1].AddDate(0, 0, 1), days[i])
	}
	assert.Equal(t, "2026-10-28", DayKey(days[len(days)-1]))
}

func TestValues(t *testing.T) {
	now := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC) // Tuesday
	days := Days(now, 1)

	values := Values(days, map[string]int{"2026-11-15": 3, "2026-11-17": 1, "2026-11-01": 9})

	assert.Equal(t, []string{"3", "0", "1"}, values)
}
