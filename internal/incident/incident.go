package incident

import (
	"time"
	"uuid"
)

type Incident struct {
	ID           uuid.UUID  `json:"id"`
	MonitorID    uuid.UUID  `json:"monitor_id"`
	StartedAt    time.Time  `json:"started_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
	Status       string     `json:"status"`
	FailureCount int        `json:"failure_count"`
	CreatedAt    time.Time  `json:"created_at"`
}
