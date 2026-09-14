package incident

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("incident not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, inc *Incident) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO incidents (id, monitor_id, started_at, resolved_at, status, failure_count, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		inc.ID, inc.MonitorID, inc.StartedAt, inc.ResolvedAt,
		inc.Status, inc.FailureCount, inc.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create incident %w", err)
	}

	return nil
}

// List returns all incidents for a monitor owned by the user.
// If monitorID is nil, it returns all incidents for the user.
func (r *Repository) List(ctx context.Context, userID uuid.UUID, p ListParams) ([]Incident, int, error) {
	where := []string{"m.user_id = $1"}
	args := []any{userID}

	if p.MonitorID != nil {
		args = append(args, *p.MonitorID)
		where = append(where, fmt.Sprintf("inc.monitor_id = $%d", len(args)))
	}

	if p.Status != "" {
		args = append(args, p.Status)
		where = append(where, fmt.Sprintf("inc.status = $%d", len(args)))
	}

	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM incidents inc
	JOIN monitors m ON m.id = inc.monitor_id
	WHERE `+whereSQL, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count incidents: %w", err)
	}

	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(
			`SELECT inc.id, inc.monitor_id, inc.started_at, inc.resolved_at, inc.status, inc.failure_count, inc.created_at
		 FROM incidents inc
		 JOIN monitors m ON m.id = inc.monitor_id
		 WHERE %s
		 ORDER BY inc.started_at DESC
		 LIMIT $%d OFFSET $%d`,
			whereSQL, len(args)-1, len(args),
		), args...,
	)

	if err != nil {
		return nil, 0, fmt.Errorf("list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		inc := Incident{}
		if err := rows.Scan(&inc.ID, &inc.MonitorID, &inc.StartedAt, &inc.ResolvedAt,
			&inc.Status, &inc.FailureCount, &inc.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan incident %w", err)
		}
		incidents = append(incidents, inc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list incidents rows: %w", err)
	}

	return incidents, total, nil
}

// GetByID returns a single incident scoped to the user's monitors.
func (r *Repository) GetByID(ctx context.Context, userID, incidentID uuid.UUID) (*Incident, error) {
	inc := &Incident{}
	err := r.pool.QueryRow(ctx,
		`SELECT inc.id, inc.monitor_id, inc.started_at, inc.resolved_at, inc.status, inc.failure_count, inc.created_at
	FROM incidents inc
	JOIN monitors m ON m.id = inc.monitor_id
	WHERE m.user_id = $1 AND inc.id = $2`,
		userID, incidentID,
	).Scan(
		&inc.ID, &inc.MonitorID, &inc.StartedAt, &inc.ResolvedAt,
		&inc.Status, &inc.FailureCount, &inc.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get incident by ID %w", err)
	}

	return inc, nil
}

// FindActiveByMonitor returns the active unresolved incident for a monitor, if any
func (r *Repository) FindActiveByMonitor(ctx context.Context, monitorID uuid.UUID) (*Incident, error) {
	inc := &Incident{}
	err := r.pool.QueryRow(ctx, `SELECT id, monitor_id, started_at, resolved_at, status, failure_count, created_at
		 FROM incidents
		 WHERE monitor_id = $1 AND resolved_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1`,
		monitorID,
	).Scan(
		&inc.ID, &inc.MonitorID, &inc.StartedAt, &inc.ResolvedAt,
		&inc.Status, &inc.FailureCount, &inc.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("find active incident %w", err)
	}

	return inc, nil
}

// Resolve marks an incident as resolved
func (r *Repository) Resolve(ctx context.Context, id uuid.UUID, resolvedAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE incidents SET resolved_at = $2, status = 'resolved' WHERE id = $1`,
		id, resolvedAt,
	)

	if err != nil {
		return fmt.Errorf("resolve incident %w", err)
	}

	return nil
}

// ActiveByMonitorIDs returns the monitor that have an active incident
func (r *Repository) ActiveByMonitorIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT monitor_id FROM incidents
	WHERE monitor_id = ANY($1::uuid[]) AND resolved_at IS NULL`, ids)

	if err != nil {
		return nil, fmt.Errorf("active indicents %w", err)
	}
	defer rows.Close()

	active := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan active incident %w", err)
		}

		active[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("active incidents rows: %w", err)
	}

	return active, nil
}
