package incident

import (
	"context"
	"uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, monitorID *uuid.UUID) ([]Incident, error) {
	return s.repo.ListByMonitor(ctx, userID, monitorID)
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID, incidentID uuid.UUID) (*Incident, error) {
	return s.repo.GetByID(ctx, userID, incidentID)
}
