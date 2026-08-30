package monitor

import (
	"context"
	"fmt"
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
