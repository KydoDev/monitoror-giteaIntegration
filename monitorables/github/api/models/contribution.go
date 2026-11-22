package models

type (
	// Contribution is the number of contributions of a calendar day
	Contribution struct {
		Date  string // YYYY-MM-DD
		Count int
	}
)
