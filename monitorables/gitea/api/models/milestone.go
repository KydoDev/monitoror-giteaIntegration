package models

import "time"

// Milestone is an open milestone of a repository.
type Milestone struct {
	ID           int64
	Title        string
	OpenIssues   int
	ClosedIssues int
	DueOn        *time.Time // nil without due date
}
