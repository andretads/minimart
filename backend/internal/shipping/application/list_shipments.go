package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
	"sort"
)

func (s *Service) ListShipments(ctx context.Context, userID string, page, pageSize int) ([]domain.Shipment, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	shipments, total, err := s.repo.ListShipments(ctx, userID, page, pageSize)
	if err != nil {
		s.m.IncShipmentList(metrics.StatusError)
		return nil, 0, err
	}

	// Sort by created date descending (newest first)
	sort.Slice(shipments, func(i, j int) bool {
		return shipments[i].CreatedAt.After(shipments[j].CreatedAt)
	})

	s.m.IncShipmentList(metrics.StatusSuccess)
	return shipments, total, nil
}
