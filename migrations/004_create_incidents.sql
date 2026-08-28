CREATE TABLE incidents (
    id            UUID PRIMARY KEY,
    monitor_id    UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at    TIMESTAMPTZ NOT NULL,
    resolved_at   TIMESTAMPTZ,
    status        TEXT NOT NULL,
    failure_count INTEGER NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_incidents_monitor_id ON incidents(monitor_id);