package monitor

import (
	"context"
	"fmt"
	"uuid"
)

func (r *Repository) SaveMonitorCheck(ctx context.Context, c *Check) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO monitor_checks (id, monitor_id, status_code, response_time_ms, success, error)
	VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.MonitorID, c.StatusCode, c.ResponseTimeMS, c.Success, c.Error)

	if err != nil {
		return fmt.Errorf("save monitor check %w", err)
	}

	return nil
}

func (r *Repository) ListChecksByMonitor(ctx context.Context, monitorID uuid.UUID, limit int) ([]Check, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, monitor_id, status_code, response_time_ms, success, error, checked_at
	FROM monitor_checks
	WHERE monitor_id = $1
	ORDER BY checked_at DESC
	LIMIT $2`, monitorID, limit)

	if err != nil {
		return nil, fmt.Errorf("list monitor checks %w", err)
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		if err := rows.Scan(&c.ID, &c.MonitorID, &c.StatusCode, &c.ResponseTimeMS, &c.Success, &c.Error, &c.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan monitor check %w", err)
		}

		checks = append(checks, c)
	}

	return checks, rows.Err()
}
