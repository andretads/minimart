package application

// RecordTokenRefresh records the outcome of a token refresh attempt.
func (s *Service) RecordTokenRefresh(status string) {
	s.m.IncTokenRefreshed(status)
}
