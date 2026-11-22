package models

import "time"

// Run represents a Gitea Actions workflow run.
// Runs triggered manually (workflow_dispatch) create no commit status, so runs are read too.
// See : https://docs.gitea.com/api (ActionWorkflowRun)
type Run struct {
	ID          int64
	Attempt     int64
	Number      int64
	Workflow    string // workflow file, e.g. "ci.yml"
	Title       string // commit message or pull request title
	Repository  string
	HeadSHA     string
	Status      string // queued, waiting, in_progress, completed ...
	Conclusion  string // success, failure, cancelled, skipped ... (when completed)
	CreatedAt   time.Time
	StartedAt   time.Time // zero when the run has not started
	CompletedAt time.Time // zero when the run has not completed
}
