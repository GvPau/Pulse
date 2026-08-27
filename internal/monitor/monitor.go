package monitor

import (
	"pulse/internal/shared"
	"time"
	"uuid"
)

type Monitor struct {
	shared.Model
	UserID          uuid.UUID `json:"user_id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	Method          string    `json:"method"`
	IntervalSeconds int       `json:"interval_seconds"`
	TimeoutSeconds  int       `json:"timeout_seconds"`
	ExpectedStatus  int       `json:"expected_status"`
	Active          bool      `json:"active"`
}

type Check struct {
	ID             uuid.UUID `json:"id"`
	MonitorID      uuid.UUID `json:"monitor_id"`
	StatusCode     *int      `json:"status_code,omitempty"`
	ResponseTimeMS int       `json:"reponse_time_ms"`
	Success        bool      `json:"success"`
	Error          *string   `json:"error,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}
