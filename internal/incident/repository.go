package incident

import (
	"context"
	"errors"
	"fmt"
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

// Resolve marks an incient as resolved
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
