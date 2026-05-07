CREATE TABLE incidents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id       UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at       TIMESTAMP NOT NULL,
    resolved_at      TIMESTAMP,
    duration_seconds INT
);
