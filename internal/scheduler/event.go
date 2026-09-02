package scheduler

import "uuid"

type Event struct {
	Type      string // add, update, remove
	MonitorId uuid.UUID
}
