package monitor

import (
	"pulse/internal/shared"
	"time"
	"uuid"
)

type Monitor struct {
	shared.Model
	UserID           uuid.UUID  `json:"user_id"`
	Name             string     `json:"name"`
	URL              string     `json:"url"`
	Method           string     `json:"method"`
	IntervalSeconds  int        `json:"interval_seconds"`
	TimeoutSeconds   int        `json:"timeout_seconds"`
	ExpectedStatus   int        `json:"expected_status"`
	Active           bool       `json:"active"`
	NextRun          *time.Time `json:"next_run,omitempty"`
	FailureThreshold int        `json:"failure_threshold"`
}

type Check struct {
	ID             uuid.UUID `json:"id"`
	MonitorID      uuid.UUID `json:"monitor_id"`
	StatusCode     *int      `json:"status_code,omitempty"`
	ResponseTimeMS int       `json:"response_time_ms"`
	Success        bool      `json:"success"`
	Error          *string   `json:"error,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

type NextRunPatch struct {
	ID      uuid.UUID
	NextRun time.Time
}

type MonitorWithStatus struct {
	Monitor
	Status         string     `json:"status"` // operational | down | unknown
	LastCheckAt    *time.Time `json:"last_check_at,omitempty"`
	LastStatusCode *int       `json:"last_status_code,omitempty"`
	LastSuccess    *bool      `json:"last_success,omitempty"`
	Uptime         *float64   `json:"uptime,omitempty"`
	AvgResponseMs  *float64   `json:"avg_response_ms,omitempty"`
	Checks         int        `json:"checks"`
}

type Window struct {
	Key      string        // Api key e.g. "24h"
	Duration time.Duration // How far the window looks
	Step     string        //PostgreSQL step interval for series buckets, e.g "1 hour"
	Trunc    string        // date_trunc unit used to align b uckets e.g "hour"
}

var windows = map[string]Window{
	"24h": {Key: "24h", Duration: 24 * time.Hour, Step: "1 hour", Trunc: "hour"},
	"7d":  {Key: "7d", Duration: 7 * 24 * time.Hour, Step: "1 hour", Trunc: "hour"},
	"30d": {Key: "30d", Duration: 30 * 24 * time.Hour, Step: "1 day", Trunc: "day"},
	"90d": {Key: "90d", Duration: 90 * 24 * time.Hour, Step: "1 day", Trunc: "day"},
}

const DefaultWindow = "24h"

func windowByKey(key string) (Window, bool) {
	w, ok := windows[key]
	return w, ok
}

type Metrics struct {
	MonitorID uuid.UUID      `json:"monitor_id"`
	Window    string         `json:"window"`
	Summary   MetricsSummary `json:"summary"`
	Series    []MetricsPoint `json:"series"`
}

type MetricsSummary struct {
	Checks        int      `json:"checks"`
	Successes     int      `json:"successes"`
	Failures      int      `json:"failures"`
	Uptime        float64  `json:"uptime"`          // % sobre la ventana
	AvgResponseMs *float64 `json:"avg_response_ms"` // null si no hay data
	P95ResponseMs *float64 `json:"p95_response_ms"` // null si no hay data
}

type MetricsPoint struct {
	Bucket        time.Time `json:"bucket"`
	Checks        int       `json:"checks"`
	Uptime        float64   `json:"uptime"`
	AvgResponseMs *float64  `json:"avg_response_ms"`
}
