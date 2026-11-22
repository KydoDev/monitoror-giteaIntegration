package models

import "time"

type (
	// Contribution is a number of contributions made at a point in time
	Contribution struct {
		Time  time.Time
		Count int
	}
)
