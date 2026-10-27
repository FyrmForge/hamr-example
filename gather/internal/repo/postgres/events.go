package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/FyrmForge/hamr-example/gather/internal/repo"
)

// Events returns the event sub-store.
func (s *Store) Events() repo.EventStore { return &eventStore{db: s.db} }

type eventStore struct {
	db *sqlx.DB
}

const eventColumns = `id, organiser_id, title, description, location, starts_at, capacity, created_at`

func (s *eventStore) Create(ctx context.Context, e *repo.Event) error {
	_, err := s.db.NamedExecContext(ctx,
		`INSERT INTO events (`+eventColumns+`)
		 VALUES (:id, :organiser_id, :title, :description, :location, :starts_at, :capacity, :created_at)`, e)
	return err
}

func (s *eventStore) GetByID(ctx context.Context, id string) (*repo.Event, error) {
	var e repo.Event
	err := s.db.GetContext(ctx, &e, `SELECT `+eventColumns+` FROM events WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *eventStore) ListUpcoming(ctx context.Context, from time.Time) ([]repo.Event, error) {
	var events []repo.Event
	err := s.db.SelectContext(ctx, &events,
		`SELECT `+eventColumns+` FROM events WHERE starts_at >= $1 ORDER BY starts_at`, from)
	return events, err
}

func (s *eventStore) AddRSVP(ctx context.Context, eventID, userID string) (bool, error) {
	// ponytail: the capacity check and insert are one statement but not
	// serialised; two people taking the last place at the same instant can
	// overbook by one. Lock the event row (SELECT ... FOR UPDATE) if that matters.
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO rsvps (event_id, user_id)
		 SELECT e.id, $2 FROM events e
		 WHERE e.id = $1 AND (SELECT COUNT(*) FROM rsvps r WHERE r.event_id = e.id) < e.capacity
		 ON CONFLICT DO NOTHING`, eventID, userID)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return true, nil
	}
	// Nothing inserted: either already going (fine) or full.
	_, going, err := s.RSVPs(ctx, eventID, userID)
	return going, err
}

func (s *eventStore) RemoveRSVP(ctx context.Context, eventID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM rsvps WHERE event_id = $1 AND user_id = $2`, eventID, userID)
	return err
}

func (s *eventStore) RSVPs(ctx context.Context, eventID, userID string) (int, bool, error) {
	var r struct {
		Count int  `db:"count"`
		Going bool `db:"going"`
	}
	err := s.db.GetContext(ctx, &r,
		`SELECT COUNT(*) AS count, COALESCE(BOOL_OR(user_id = $2), false) AS going
		 FROM rsvps WHERE event_id = $1`, eventID, userID)
	return r.Count, r.Going, err
}
