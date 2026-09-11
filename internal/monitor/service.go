package monitor

import (
	"context"
	"errors"
	"fmt"
	"pulse/internal/httpx"
	"time"
	"uuid"
)

type Service struct {
	repo      *Repository
	incidents incidentLookup
	onCreate  func(ctx context.Context, id uuid.UUID)
	onUpdate  func(ctx context.Context, id uuid.UUID)
	onDelete  func(ctx context.Context, id uuid.UUID)
}

func NewService(repo *Repository, incidents incidentLookup, onCreate, onUpdate, onDelete func(ctx context.Context, id uuid.UUID)) *Service {
	return &Service{
		repo:      repo,
		incidents: incidents,
		onCreate:  onCreate,
		onUpdate:  onUpdate,
		onDelete:  onDelete,
	}
}

type ListParams struct {
	Page   int
	Limit  int
	Active *bool
	Q      string
	Sort   string
	Order  string
	Window string
}

type CheckListParams struct {
	Page    int
	Limit   int
	Success *bool
}

type incidentLookup interface {
	ActiveByMonitorIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error)
}

var monitorSortColumns = map[string]string{
	"name":             "name",
	"created_at":       "created_at",
	"interval_seconds": "interval_seconds",
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

func (s *Service) List(ctx context.Context, userID uuid.UUID, p ListParams) ([]MonitorWithStatus, int, error) {
	monitors, total, err := s.repo.ListByUser(ctx, userID, p)

	if err != nil {
		return nil, 0, err
	}

	if len(monitors) == 0 {
		return []MonitorWithStatus{}, total, nil
	}

	ids := make([]uuid.UUID, 0, len(monitors))
	for _, m := range monitors {
		ids = append(ids, m.ID)
	}

	win := windows[DefaultWindow]
	if w, ok := windows[p.Window]; ok {
		win = w
	}

	statuses, err := s.repo.ListStatusByIDs(ctx, ids, win)
	if err != nil {
		return nil, 0, err
	}

	active, err := s.incidents.ActiveByMonitorIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	byID := make(map[uuid.UUID]MonitorWithStatus, len(statuses))
	for _, st := range statuses {
		byID[st.ID] = st
	}

	result := make([]MonitorWithStatus, 0, len(monitors))
	for _, m := range monitors {
		mws := MonitorWithStatus{Monitor: m}
		if st, ok := byID[m.ID]; ok {
			mws.LastCheckAt = st.LastCheckAt
			mws.LastStatusCode = st.LastStatusCode
			mws.LastSuccess = st.LastSuccess
			mws.Uptime = st.Uptime
			mws.AvgResponseMs = st.AvgResponseMs
			mws.Checks = st.Checks
		}
		mws.Status = computeStatus(active[m.ID], mws.LastCheckAt)
		result = append(result, mws)
	}

	return result, total, nil

}

func computeStatus(hasActiveIncident bool, lastCheckedAt *time.Time) string {
	if hasActiveIncident {
		return "down"
	}

	if lastCheckedAt == nil {
		return "unknown"
	}

	return "operational"
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

func (s *Service) ListChecks(ctx context.Context, userID, monitorID uuid.UUID, p CheckListParams) ([]Check, int, error) {
	if _, err := s.repo.GetByID(ctx, userID, monitorID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListChecksByMonitor(ctx, monitorID, p)
}

func (s *Service) Metrics(ctx context.Context, userID, monitorID uuid.UUID, windowKey string) (*Metrics, error) {
	win, ok := windowByKey(windowKey)
	if !ok {
		return nil, errors.New("invalid window")
	}

	if _, err := s.repo.GetByID(ctx, userID, monitorID); err != nil {
		return nil, err
	}

	summary, err := s.repo.MetricsSummaryByMonitor(ctx, monitorID, win)
	if err != nil {
		return nil, err
	}

	series, err := s.repo.MetricsSeriesByMonitor(ctx, monitorID, win)
	if err != nil {
		return nil, err
	}

	summary.Failures = summary.Checks - summary.Successes

	return &Metrics{
		MonitorID: monitorID,
		Window:    win.Key,
		Summary:   *summary,
		Series:    series,
	}, nil
}
