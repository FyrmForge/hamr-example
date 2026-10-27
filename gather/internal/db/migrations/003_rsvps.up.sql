BEGIN;
CREATE TABLE rsvps (
    event_id    TEXT        NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id     TEXT        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id)
);
COMMIT;
