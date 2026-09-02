package monitor

import (
	"context"
	"errors"
	"fmt"
	"uuid"
)

type Service struct {
	repo     *Repository
	onCreate func(ctx context.Context, id uuid.UUID)
	onUpdate func(ctx context.Context, id uuid.UUID)
	onDelete func(ctx context.Context, id uuid.UUID)
}

func NewService(repo *Repository, onCreate, onUpdate, onDelete func(ctx context.Context, id uuid.UUID)) *Service {
	return &Service{
		repo:     repo,
		onCreate: onCreate,
		onUpdate: onUpdate,
		onDelete: onDelete,
	}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, m *Monitor) (*Monitor, error) {
	if m.Name == "" || m.URL == "" {
		return nil, errors.New("name and url are required")
	}

	// Default failure threshold to 3 if not set
	if m.FailureThreshold == 0 {
		m.FailureThreshold = 3
	}
	if m.FailureThreshold < 1 {
		return nil, errors.New("failure_threshold must be at least 1")
	}

	m.ID = uuid.New()
	m.UserID = userID

	if err := s.repo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("update monitor: %w", err)
	}

	if s.onCreate != nil {
		s.onCreate(ctx, m.ID)
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

	if s.onUpdate != nil {
		s.onUpdate(ctx, id)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {

	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}

	if s.onDelete != nil {
		s.onDelete(ctx, id)
	}

	return nil
}
