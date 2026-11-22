// Package contributions builds the daily series displayed by contributions calendar tiles.
//
// A series covers whole weeks: the first value is a Sunday and the last value is today, oldest first.
// The UI lays the values out by index (one column per week, one row per weekday) and needs no dates.
package contributions

import (
	"strconv"
	"time"
)

const (
	// DefaultWeeks is used when the tile does not set weeks.
	DefaultWeeks = 26
	// MaxWeeks matches the one year limit of the GitHub contributions API.
	MaxWeeks = 52

	dayKeyLayout = "2006-01-02"
)

// Days returns the days of a calendar of `weeks` weeks ending today: from the Sunday of the week
// `weeks-1` weeks before now, to the day of now. Days are at midnight in the location of now.
// Days are built with calendar arithmetic, so daylight saving changes never skip or repeat a day.
func Days(now time.Time, weeks int) []time.Time {
	if weeks <= 0 {
		weeks = DefaultWeeks
	}
	if weeks > MaxWeeks {
		weeks = MaxWeeks
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today.AddDate(0, 0, -int(today.Weekday())-7*(weeks-1))

	days := make([]time.Time, 0, 7*weeks)
	for day := start; !day.After(today); day = day.AddDate(0, 0, 1) {
		days = append(days, day)
	}

	return days
}

// DayKey identifies the calendar day of t, in the location of t.
func DayKey(t time.Time) string {
	return t.Format(dayKeyLayout)
}

// Values maps the counts per day (keyed by DayKey) onto days, as tile metric values.
// Days without contributions are 0.
func Values(days []time.Time, counts map[string]int) []string {
	values := make([]string, len(days))
	for i, day := range days {
		values[i] = strconv.Itoa(counts[DayKey(day)])
	}

	return values
}
