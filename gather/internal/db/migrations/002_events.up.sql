BEGIN;
CREATE TABLE events (
    id            TEXT PRIMARY KEY,
    organiser_id  TEXT        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    location      TEXT        NOT NULL DEFAULT '',
    starts_at     TIMESTAMPTZ NOT NULL,
    capacity      INTEGER     NOT NULL CHECK (capacity > 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_starts_at ON events (starts_at);
COMMIT;
