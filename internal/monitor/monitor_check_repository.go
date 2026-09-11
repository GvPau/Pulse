package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"
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

func (r *Repository) MetricsSummaryByMonitor(ctx context.Context, monitorID uuid.UUID, win Window) (*MetricsSummary, error) {
	start := time.Now().Add(-win.Duration)

	var s MetricsSummary
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) AS checks,
			COUNT(*) FILTER (WHERE success) AS successes,
			COALESCE(AVG(CASE WHEN success THEN 100.0 ELSE 0 END), 0) AS uptime,
			AVG(response_time_ms) AS avg_response_ms,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY response_time_ms) AS p95
	 FROM monitor_checks
	 WHERE monitor_id = $1 AND checked_at >= $2`, monitorID, start).
		Scan(&s.Checks, &s.Successes, &s.Uptime, &s.AvgResponseMs, &s.P95ResponseMs)

	if err != nil {
		return nil, fmt.Errorf("metrics summary %w", err)
	}

	return &s, nil
}

func (r *Repository) MetricsSeriesByMonitor(ctx context.Context, monitorID uuid.UUID, win Window) ([]MetricsPoint, error) {
	start := time.Now().Add(-win.Duration)

	rows, err := r.pool.Query(ctx,
		`WITH buckets as (
		SELECT generate_series(
			date_trunc($4::text, $2::timestamptz),
			now(),
			$3::interval
		) AS bucket	
	),
	agg AS (
		SELECT date_bin($3::interval, checked_at, date_trunc($4::text, $2::timestamptz))
		AS bucket,
			COUNT(*) AS checks,
			AVG(CASE WHEN success THEN 100.0 ELSE 0 END) AS uptime,
			AVG(response_time_ms) AS avg_response_ms
		FROM monitor_checks
		WHERE monitor_id = $1 AND checked_at >= $2
		GROUP BY 1
	)
	SELECT b.bucket,
		COALESCE(a.checks, 0),
		COALESCE(a.uptime, 0),
		a.avg_response_ms
	FROM buckets b
	LEFT JOIN agg a ON a.bucket = b.bucket
	ORDER BY b.bucket`,
		monitorID, start, win.Step, win.Trunc)

	if err != nil {
		return nil, fmt.Errorf("metrics series %w", err)
	}
	defer rows.Close()

	var points []MetricsPoint
	for rows.Next() {
		var p MetricsPoint
		if err := rows.Scan(&p.Bucket, &p.Checks, &p.Uptime, &p.AvgResponseMs); err != nil {
			return nil, fmt.Errorf("scan metrics point: %w", err)
		}
		points = append(points, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("metrics series rows %w", err)
	}

	return points, nil
}
