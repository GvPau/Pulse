package incident

import (
	"context"
	"uuid"
)

type Service struct {
	repo *Repository
}

type ListParams struct {
	Page      int
	Limit     int
	MonitorID *uuid.UUID
	Status    string
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, p ListParams) ([]Incident, int, error) {
	return s.repo.List(ctx, userID, p)
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID, incidentID uuid.UUID) (*Incident, error) {
	return s.repo.GetByID(ctx, userID, incidentID)
}
