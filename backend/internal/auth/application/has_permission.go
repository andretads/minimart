package application

import "slices"

func (s *Service) HasPermission(permissions []string, required string) bool {
	return slices.Contains(permissions, required)
}

func (s *Service) HasAnyPermission(permissions []string, required ...string) bool {
	for _, r := range required {
		if slices.Contains(permissions, r) {
			return true
		}
	}
	return false
}
