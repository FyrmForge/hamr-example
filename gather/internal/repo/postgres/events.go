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
