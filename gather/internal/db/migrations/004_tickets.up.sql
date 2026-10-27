BEGIN;
ALTER TABLE events ADD COLUMN price_cents INTEGER NOT NULL DEFAULT 0 CHECK (price_cents >= 0);

CREATE TABLE tickets (
    id                   TEXT PRIMARY KEY,
    event_id             TEXT        NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id              TEXT        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    checkout_session_id  TEXT        NOT NULL UNIQUE,
    amount_cents         INTEGER     NOT NULL,
    status               TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'expired')),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at              TIMESTAMPTZ
);

CREATE INDEX idx_tickets_event_user ON tickets (event_id, user_id);
COMMIT;
