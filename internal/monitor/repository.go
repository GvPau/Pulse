package monitor

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

type Repository struct {
	pool *pgxpool.Pool
}

var ErrNotFound = errors.New("monitor not found")

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, m *Monitor) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO monitors (id, user_id, name, url, method, interval_seconds, timeout_seconds, expected_status, active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		m.ID, m.UserID, m.Name, m.URL, m.Method, m.IntervalSeconds, m.TimeoutSeconds, m.ExpectedStatus, m.Active,
	)

	if err != nil {
		return fmt.Errorf("create monitor: %w", err)
	}

	return nil
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Monitor, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, url, method, interval_seconds, timeout_seconds, expected_status, active, created_at, updated_at
		 FROM monitors WHERE user_id = $1 ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	defer rows.Close()

	var monitors []Monitor
	for rows.Next() {
		var m Monitor
		if err := rows.Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.Method, &m.IntervalSeconds, &m.TimeoutSeconds, &m.ExpectedStatus, &m.Active, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan monitor: %w", err)
		}
		monitors = append(monitors, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list monitors rows: %w", err)
	}

	return monitors, nil
}

func (r *Repository) GetByID(ctx context.Context, userID, id uuid.UUID) (*Monitor, error) {
	m := &Monitor{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, url, method, interval_seconds, timeout_seconds, expected_status, active, created_at, updated_at, failure_threshold
		 FROM monitors WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.Method, &m.IntervalSeconds, &m.TimeoutSeconds, &m.ExpectedStatus, &m.Active, &m.CreatedAt, &m.UpdatedAt, &m.FailureThreshold)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get monitor by id: %w", err)
	}

	return m, nil
}

func (r *Repository) Update(ctx context.Context, userID, id uuid.UUID, m *Monitor) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE monitors
		 SET name = $1, url = $2, method = $3,
		     interval_seconds = $4, timeout_seconds = $5,
		     expected_status = $6, active = $7
		 WHERE id = $8 AND user_id = $9`,
		m.Name, m.URL, m.Method, m.IntervalSeconds, m.TimeoutSeconds, m.ExpectedStatus, m.Active, id, userID,
	)

	if err != nil {
		return fmt.Errorf("update monitor: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM monitors WHERE id = $1 AND user_id = $2`,
		id, userID,
	)

	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) ListActive(ctx context.Context) ([]Monitor, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, url, method, interval_seconds, timeout_seconds, expected_status, active, next_run, created_at, updated_at, failure_threshold 
	FROM monitors 
	WHERE active = true`)

	if err != nil {
		return nil, fmt.Errorf("list active monitors %w", err)
	}
	defer rows.Close()

	var monitors []Monitor
	for rows.Next() {
		var m Monitor
		err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.Name,
			&m.URL,
			&m.Method,
			&m.IntervalSeconds,
			&m.TimeoutSeconds,
			&m.ExpectedStatus,
			&m.Active,
			&m.NextRun,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.FailureThreshold,
		)
		if err != nil {
			return nil, fmt.Errorf("scan active monitor: %w", err)
		}
		monitors = append(monitors, m)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("list active monitors rows %w", err)
	}

	return monitors, nil
}

func (r *Repository) UpdateNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE monitors SET next_run = $1 WHERE id = $2`, nextRun, id)

	if err != nil {
		return fmt.Errorf("update next run %w", err)
	}

	return nil
}

func (r *Repository) GetMonitorById(ctx context.Context, id uuid.UUID) (*Monitor, error) {
	m := &Monitor{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, url, method, interval_seconds, timeout_seconds, expected_status, active, next_run, created_at, updated_at, failure_threshold
		 FROM monitors WHERE id = $1`, id).Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.Method,
		&m.IntervalSeconds, &m.TimeoutSeconds, &m.ExpectedStatus, &m.Active,
		&m.NextRun, &m.CreatedAt, &m.UpdatedAt, &m.FailureThreshold)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get monitor by id: %w", err)
	}

	return m, nil
}

// CountConsecutiveFailures returns how many consecutive failing checks the
// monitor has had, counting back from the most recent check.
func (r *Repository) CountConsecutiveFailures(ctx context.Context, monitorID uuid.UUID) (int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT success 
	FROM monitor_checks 
	WHERE monitor_id = $1 ORDER BY checked_at DESC`, monitorID)

	if err != nil {
		return 0, fmt.Errorf("count consecutive failures %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var success bool
		if err := rows.Scan(&success); err != nil {
			return 0, err
		}

		if !success {
			count++
		} else {
			break
		}
	}

	return count, rows.Err()
}

func (r *Repository) UpdateNextRuns(ctx context.Context, patches []NextRunPatch) error {
	if len(patches) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(patches))
	args := make([]interface{}, 0, len(patches)*2)
	for i, p := range patches {
		placeholders = append(placeholders, fmt.Sprintf("($%d::uuid, $%d::timestamptz)", i*2+1, i*2+2))
		args = append(args, p.ID, p.NextRun)
	}

	query := fmt.Sprintf(
		`UPDATE monitors SET next_run = v.next_run
		FROM (VALUES %s) as v(id, next_run)
		WHERE monitors.id = v.id`, strings.Join(placeholders, ", "),
	)

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update next runs batch: %w", err)
	}

	return nil
}
