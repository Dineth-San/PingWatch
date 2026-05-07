CREATE TABLE checks (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id       UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    checked_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    status_code      INT,
    response_time_ms INT,
    is_up            BOOLEAN NOT NULL,
    error_message    VARCHAR(500)
);

CREATE INDEX checks_monitor_checked_at ON checks (monitor_id, checked_at DESC);
