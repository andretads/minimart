package response

import (
	"errors"
	"net/http"
)

// ErrorStatus pairs a sentinel error with the HTTP status, and optional client
// message, used when it matches. An empty Message answers with err.Error().
type ErrorStatus struct {
	Err     error
	Status  int
	Message string
}

// DomainError writes the standard envelope for err using the first entry of
// statuses that matches (compared with errors.Is, in order). Unmatched errors
// answer 500 with fallbackMessage so internal details never leak to clients.
// It reports whether err matched, so callers can log unexpected errors.
func DomainError(w http.ResponseWriter, err error, statuses []ErrorStatus, fallbackMessage string) bool {
	for _, s := range statuses {
		if errors.Is(err, s.Err) {
			msg := s.Message
			if msg == "" {
				msg = err.Error()
			}
			WriteJSON(w, s.Status, Fail(s.Status, msg))
			return true
		}
	}
	WriteJSON(w, http.StatusInternalServerError, Fail(http.StatusInternalServerError, fallbackMessage))
	return false
}

// KnownDomainError reports whether err matches any sentinel in statuses.
func KnownDomainError(err error, statuses []ErrorStatus) bool {
	for _, s := range statuses {
		if errors.Is(err, s.Err) {
			return true
		}
	}
	return false
}
