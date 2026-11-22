package models

import (
	"errors"
	"fmt"
	"net/http"
)

// StatusError is returned when Gitea answers with an unexpected HTTP status
type StatusError struct {
	StatusCode int
	Body       string // truncated
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("gitea returned status %d: %s", e.StatusCode, e.Body)
}

// IsNotFound tells whether err is a 404 answer of Gitea (unknown owner, or endpoint missing on this version)
func IsNotFound(err error) bool {
	var statusError *StatusError
	return errors.As(err, &statusError) && statusError.StatusCode == http.StatusNotFound
}
