package monitor

import (
	"context"
	"fmt"
	"strings"
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

func (r *Repository) ListChecksByMonitor(ctx context.Context, monitorID uuid.UUID, p CheckListParams) ([]Check, int, error) {
	where := []string{"monitor_id = $1"}
	args := []any{monitorID}

	if p.Success != nil {
		args = append(args, *p.Success)
		where = append(where, fmt.Sprintf("success = $%d", len(args)))
	}

	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM monitor_checks WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count monitor checks: %w", err)
	}

	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(`SELECT id, monitor_id, status_code, response_time_ms, success, error, checked_at
		 FROM monitor_checks WHERE %s ORDER BY checked_at DESC LIMIT $%d OFFSET $%d`,
			whereSQL, len(args)-1, len(args)),
		args...)

	if err != nil {
		return nil, 0, fmt.Errorf("list monitor checks %w", err)
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		if err := rows.Scan(&c.ID, &c.MonitorID, &c.StatusCode, &c.ResponseTimeMS, &c.Success, &c.Error, &c.CheckedAt); err != nil {
			return nil, 0, fmt.Errorf("scan monitor check %w", err)
		}
		checks = append(checks, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list monitor checks rows %w", err)
	}

	return checks, total, nil
}
