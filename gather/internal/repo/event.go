package repo

import (
	"context"
	"time"
)

// EventStore is the event sub-store, reached via Store.Events().
type EventStore interface {
	// Create inserts a new event.
	Create(ctx context.Context, event *Event) error

	// GetByID returns an event by ID, or nil if not found.
	GetByID(ctx context.Context, id string) (*Event, error)

	// ListUpcoming returns events starting at or after from, soonest first.
	ListUpcoming(ctx context.Context, from time.Time) ([]Event, error)

	// AddRSVP records that userID is going. It returns false, and records
	// nothing, when the event is already full.
	AddRSVP(ctx context.Context, eventID, userID string) (bool, error)

	// RemoveRSVP deletes userID's RSVP, if any.
	RemoveRSVP(ctx context.Context, eventID, userID string) error

	// RSVPs returns how many people are going and whether userID is one of them.
	RSVPs(ctx context.Context, eventID, userID string) (count int, going bool, err error)
}

// Event is something people can sign up for.
type Event struct {
	ID          string    `db:"id"`
	OrganiserID string    `db:"organiser_id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	Location    string    `db:"location"`
	StartsAt    time.Time `db:"starts_at"`
	Capacity    int       `db:"capacity"`
	PriceCents  int       `db:"price_cents"` // 0: free, no tickets
	CreatedAt   time.Time `db:"created_at"`
}
