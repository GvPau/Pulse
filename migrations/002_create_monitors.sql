CREATE TABLE monitors (
    id               UUID PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    url              TEXT NOT NULL,
    method           TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL,
    timeout_seconds  INTEGER NOT NULL,
    expected_status  INTEGER NOT NULL,
    active           BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_monitors_user_id ON monitors(user_id);