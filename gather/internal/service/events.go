package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FyrmForge/hamr-example/gather/internal/repo"
)

// StartsAtLayout is the format an <input type="datetime-local"> submits.
const StartsAtLayout = "2006-01-02T15:04"

// ErrInvalidEvent wraps every reason NewEvent is rejected.
var ErrInvalidEvent = errors.New("invalid event")

// NewEvent is what an organiser submits. Values arrive as form strings; the
// service owns turning them into an Event.
type NewEvent struct {
	Title       string
	Description string
	Location    string
	StartsAt    string // StartsAtLayout, read as UTC
	Capacity    string
}

// EventService holds the event rules.
type EventService struct {
	store repo.Store
}

// NewEventService creates a new event service.
func NewEventService(store repo.Store) *EventService {
	return &EventService{store: store}
}

// Create checks in and stores it as an event organised by organiserID.
func (s *EventService) Create(ctx context.Context, organiserID string, in NewEvent) (*repo.Event, error) {
	e, err := in.toEvent(organiserID, time.Now())
	if err != nil {
		return nil, err
	}
	if err := s.store.Events().Create(ctx, e); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	return e, nil
}

// Get returns the event with id, or nil if there is none.
func (s *EventService) Get(ctx context.Context, id string) (*repo.Event, error) {
	return s.store.Events().GetByID(ctx, id)
}

// Upcoming lists events that have not started yet, soonest first.
func (s *EventService) Upcoming(ctx context.Context) ([]repo.Event, error) {
	return s.store.Events().ListUpcoming(ctx, time.Now())
}

// toEvent parses and checks the submitted values. The form validates the same
// things for inline errors; this is the backstop for anything that skips it.
func (in NewEvent) toEvent(organiserID string, now time.Time) (*repo.Event, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidEvent)
	}
	// ponytail: times are read and shown as UTC; store the organiser's zone if Gather goes multi-city.
	startsAt, err := time.Parse(StartsAtLayout, in.StartsAt)
	if err != nil {
		return nil, fmt.Errorf("%w: starts_at %q", ErrInvalidEvent, in.StartsAt)
	}
	capacity, err := strconv.Atoi(in.Capacity)
	if err != nil || capacity < 1 {
		return nil, fmt.Errorf("%w: capacity %q", ErrInvalidEvent, in.Capacity)
	}
	return &repo.Event{
		ID:          uuid.New().String(),
		OrganiserID: organiserID,
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Location:    strings.TrimSpace(in.Location),
		StartsAt:    startsAt,
		Capacity:    capacity,
		CreatedAt:   now,
	}, nil
}
