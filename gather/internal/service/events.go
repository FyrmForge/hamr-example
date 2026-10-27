package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FyrmForge/hamr/pkg/email"
	"github.com/FyrmForge/hamr/pkg/logging"
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
	Price       string // euros, e.g. "5" or "7.50"; empty means free
}

// mailFrom is the sender of Gather's notifications.
// ponytail: fixed address; read it from the environment when Gather sends from a real domain.
var mailFrom = email.Addr("Gather", "no-reply@gather.example")

// EventService holds the event rules.
type EventService struct {
	store repo.Store
	mail  email.Sender // nil: notifications are skipped
}

// NewEventService creates a new event service. mail may be nil (no email
// provider configured); RSVPs then work without notifying anyone.
func NewEventService(store repo.Store, mail email.Sender) *EventService {
	return &EventService{store: store, mail: mail}
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

// ErrEventFull is returned when someone RSVPs to an event with no places left.
var ErrEventFull = errors.New("event is full")

// Attendance is the RSVP state of an event as one viewer sees it.
type Attendance struct {
	EventID  string
	Going    int  // how many people are going
	Capacity int  // places in total
	IsGoing  bool // whether the viewer is one of them
}

// Full reports whether every place is taken.
func (a Attendance) Full() bool { return a.Going >= a.Capacity }

// EventView is an event plus its attendance for one viewer.
type EventView struct {
	Event      *repo.Event
	Attendance Attendance
	Ticket     *repo.Ticket // the viewer's paid ticket, if any
}

// View returns the event with id as viewerID sees it (viewerID may be
// empty for a signed-out visitor), or nil if there is no such event.
func (s *EventService) View(ctx context.Context, id, viewerID string) (*EventView, error) {
	e, err := s.store.Events().GetByID(ctx, id)
	if err != nil || e == nil {
		return nil, err
	}
	a, err := s.attendance(ctx, e, viewerID)
	if err != nil {
		return nil, err
	}
	v := &EventView{Event: e, Attendance: a}
	if viewerID != "" && e.PriceCents > 0 {
		if v.Ticket, err = s.store.Tickets().Paid(ctx, e.ID, viewerID); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// ToggleRSVP marks userID as going, or not going if they already were.
// It returns ErrEventFull, with the current attendance, when there is no
// place left, and (nil, nil) when the event doesn't exist.
func (s *EventService) ToggleRSVP(ctx context.Context, eventID, userID string) (*Attendance, error) {
	events := s.store.Events()
	e, err := events.GetByID(ctx, eventID)
	if err != nil || e == nil {
		return nil, err
	}
	_, going, err := events.RSVPs(ctx, eventID, userID)
	if err != nil {
		return nil, err
	}
	var full bool
	if going {
		err = events.RemoveRSVP(ctx, eventID, userID)
	} else {
		var added bool
		added, err = events.AddRSVP(ctx, eventID, userID)
		full = !added
	}
	if err != nil {
		return nil, fmt.Errorf("rsvp: %w", err)
	}
	a, err := s.attendance(ctx, e, userID)
	if err != nil {
		return nil, err
	}
	if full {
		return &a, ErrEventFull
	}
	if !going {
		s.notifyOrganiser(ctx, e, userID, a)
	}
	return &a, nil
}

// notifyOrganiser emails the organiser that userID is going. A failed email
// is logged, not returned: the RSVP itself has already succeeded.
func (s *EventService) notifyOrganiser(ctx context.Context, e *repo.Event, userID string, a Attendance) {
	if s.mail == nil || e.OrganiserID == userID {
		return
	}
	log := logging.FromContext(ctx)
	organiser, err := s.store.Users().GetByID(ctx, e.OrganiserID)
	if err != nil || organiser == nil {
		log.Error("rsvp email: load organiser", "event", e.ID, "error", err)
		return
	}
	guest, err := s.store.Users().GetByID(ctx, userID)
	if err != nil || guest == nil {
		log.Error("rsvp email: load guest", "event", e.ID, "error", err)
		return
	}
	if _, err := s.mail.Send(ctx, rsvpEmail(e, organiser, guest, a)); err != nil {
		log.Error("rsvp email: send", "event", e.ID, "error", err)
	}
}

// rsvpEmail is the message an organiser gets when someone says they're going.
func rsvpEmail(e *repo.Event, organiser, guest *repo.User, a Attendance) email.Message {
	who := guest.Name
	if who == "" {
		who = guest.Email
	}
	return email.Message{
		From:    mailFrom,
		To:      []email.Address{email.Addr(organiser.Name, organiser.Email)},
		Subject: fmt.Sprintf("%s is going to %s", who, e.Title),
		Text: fmt.Sprintf("%s (%s) just RSVPed to %s on %s.\n\n%d of %d places are taken.\n",
			who, guest.Email, e.Title, e.StartsAt.Format("Mon 2 Jan, 15:04"), a.Going, a.Capacity),
		Tags: map[string]string{"kind": "rsvp"},
	}
}

func (s *EventService) attendance(ctx context.Context, e *repo.Event, viewerID string) (Attendance, error) {
	count, going, err := s.store.Events().RSVPs(ctx, e.ID, viewerID)
	return Attendance{EventID: e.ID, Going: count, Capacity: e.Capacity, IsGoing: going}, err
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
	price, err := ParsePrice(in.Price)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}
	return &repo.Event{
		ID:          uuid.New().String(),
		OrganiserID: organiserID,
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Location:    strings.TrimSpace(in.Location),
		StartsAt:    startsAt,
		Capacity:    capacity,
		PriceCents:  price,
		CreatedAt:   now,
	}, nil
}
