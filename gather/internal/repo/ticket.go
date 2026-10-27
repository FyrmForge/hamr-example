package repo

import (
	"context"
	"time"
)

// TicketStore is the ticket sub-store, reached via Store.Tickets().
type TicketStore interface {
	// Create inserts a pending ticket.
	Create(ctx context.Context, t *Ticket) error

	// SetStatus moves the ticket paid for through checkoutSessionID to status
	// ("paid" or "expired"). Unknown sessions are ignored.
	SetStatus(ctx context.Context, checkoutSessionID, status string) error

	// Paid returns userID's paid ticket for eventID, or nil.
	Paid(ctx context.Context, eventID, userID string) (*Ticket, error)
}

// Ticket is one paid place at an event. It starts "pending" when checkout
// opens and becomes "paid" when Stripe's webhook says so.
type Ticket struct {
	ID                string     `db:"id"`
	EventID           string     `db:"event_id"`
	UserID            string     `db:"user_id"`
	CheckoutSessionID string     `db:"checkout_session_id"`
	AmountCents       int        `db:"amount_cents"`
	Status            string     `db:"status"`
	CreatedAt         time.Time  `db:"created_at"`
	PaidAt            *time.Time `db:"paid_at"`
}
