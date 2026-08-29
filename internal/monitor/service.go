package monitor

import (
	"context"
	"errors"
	"fmt"
	"uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, m *Monitor) (*Monitor, error) {
	if m.Name == "" || m.URL == "" {
		return nil, errors.New("name and url are required")
	}

	m.ID = uuid.New()
	m.UserID = userID

	if err := s.repo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("update monitor: %w", err)
	}

	return m, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Monitor, error) {
	m, err := s.repo.ListByUser(ctx, userID)

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Monitor, error) {
	m, err := s.repo.GetByID(ctx, userID, id)

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, m *Monitor) error {
	if err := s.repo.Update(ctx, userID, id, m); err != nil {
		return fmt.Errorf("update monitor: %w", err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}
