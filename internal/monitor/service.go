package monitor

import (
	"context"
	"errors"
	"fmt"
	"pulse/internal/httpx"
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

var monitorSortColumns = map[string]string{
	"name":             "name",
	"created_at":       "created_at",
	"interval_seconds": "interval_seconds",
}

type ListParams struct {
	Page   int
	Limit  int
	Active *bool
	Q      string
	Sort   string
	Order  string
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, m *Monitor) (*Monitor, error) {
	fields := map[string]string{}
	if m.Name == "" {
		fields["name"] = "is required"
	}

	if m.URL == "" {
		fields["url"] = "is required"
	}

	if len(fields) > 0 {
		return nil, &httpx.ValidationError{Message: "validation failed", Fields: fields}
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

func (s *Service) List(ctx context.Context, userID uuid.UUID, p ListParams) ([]Monitor, int, error) {
	monitors, total, err := s.repo.ListByUser(ctx, userID, p)

	if err != nil {
		return nil, 0, err
	}

	return monitors, total, nil
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Monitor, error) {
	m, err := s.repo.GetByID(ctx, userID, id)

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, m *Monitor) error {
	fields := map[string]string{}
	if m.Name == "" {
		fields["name"] = "is required"
	}
	if m.URL == "" {
		fields["url"] = "is required"
	}
	if len(fields) > 0 {
		return &httpx.ValidationError{Message: "validation failed", Fields: fields}
	}

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

func (s *Service) ListChecks(ctx context.Context, userID, monitorID uuid.UUID, limit int) ([]Check, error) {
	if _, err := s.repo.GetByID(ctx, userID, monitorID); err != nil {
		return nil, fmt.Errorf("monitor not found: %w", err)
	}
	return s.repo.ListChecksByMonitor(ctx, monitorID, limit)
}
