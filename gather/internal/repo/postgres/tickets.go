package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/FyrmForge/hamr-example/gather/internal/repo"
)

// Tickets returns the ticket sub-store.
func (s *Store) Tickets() repo.TicketStore { return &ticketStore{db: s.db} }

type ticketStore struct {
	db *sqlx.DB
}

func (s *ticketStore) Create(ctx context.Context, t *repo.Ticket) error {
	_, err := s.db.NamedExecContext(ctx,
		`INSERT INTO tickets (id, event_id, user_id, checkout_session_id, amount_cents, status, created_at)
		 VALUES (:id, :event_id, :user_id, :checkout_session_id, :amount_cents, :status, :created_at)`, t)
	return err
}

func (s *ticketStore) SetStatus(ctx context.Context, checkoutSessionID, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tickets SET status = $2, paid_at = CASE WHEN $2 = 'paid' THEN NOW() ELSE paid_at END
		 WHERE checkout_session_id = $1 AND status <> 'paid'`, checkoutSessionID, status)
	return err
}

func (s *ticketStore) Paid(ctx context.Context, eventID, userID string) (*repo.Ticket, error) {
	var t repo.Ticket
	err := s.db.GetContext(ctx, &t,
		`SELECT id, event_id, user_id, checkout_session_id, amount_cents, status, created_at, paid_at
		 FROM tickets WHERE event_id = $1 AND user_id = $2 AND status = 'paid' LIMIT 1`, eventID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
