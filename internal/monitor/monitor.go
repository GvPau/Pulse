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
