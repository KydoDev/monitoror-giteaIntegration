package models

type Repository struct {
	Name          string
	DefaultBranch string
	Archived      bool
	Empty         bool
}
