CREATE TABLE monitor_checks (
    id               UUID PRIMARY KEY,
    monitor_id       UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status_code      INTEGER,
    response_time_ms INTEGER NOT NULL,
    success          BOOLEAN NOT NULL,
    error            TEXT,
    checked_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_monitor_checks_monitor_id ON monitor_checks(monitor_id);
CREATE INDEX idx_monitor_checks_checked_at ON monitor_checks(checked_at);