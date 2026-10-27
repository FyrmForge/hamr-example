package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/FyrmForge/hamr-example/gather/internal/repo"
)

func TestNewEventToEvent(t *testing.T) {
	ok := NewEvent{Title: " Go and HTMX meetup ", StartsAt: "2026-11-14T19:00", Capacity: "40"}
	e, err := ok.toEvent("u1", time.Now())
	if err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	if e.Title != "Go and HTMX meetup" || e.Capacity != 40 || e.OrganiserID != "u1" ||
		!e.StartsAt.Equal(time.Date(2026, 11, 14, 19, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected event: %+v", e)
	}

	for name, in := range map[string]NewEvent{
		"no title":      {Title: " ", StartsAt: ok.StartsAt, Capacity: ok.Capacity},
		"bad time":      {Title: ok.Title, StartsAt: "next friday", Capacity: ok.Capacity},
		"zero capacity": {Title: ok.Title, StartsAt: ok.StartsAt, Capacity: "0"},
		"text capacity": {Title: ok.Title, StartsAt: ok.StartsAt, Capacity: "lots"},
	} {
		if _, err := in.toEvent("u1", time.Now()); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("%s: want ErrInvalidEvent, got %v", name, err)
		}
	}
}

func TestAttendanceFull(t *testing.T) {
	if (Attendance{Going: 39, Capacity: 40}).Full() || !(Attendance{Going: 40, Capacity: 40}).Full() {
		t.Fatal("Full() wrong at the capacity boundary")
	}
}

func TestRSVPEmail(t *testing.T) {
	e := &repo.Event{Title: "Go and HTMX meetup", StartsAt: time.Date(2026, 11, 14, 19, 0, 0, 0, time.UTC)}
	org := &repo.User{Name: "Ana", Email: "ana@example.com"}
	guest := &repo.User{Email: "ben@example.com"} // no name: falls back to the email
	m := rsvpEmail(e, org, guest, Attendance{Going: 23, Capacity: 40})
	if m.To[0].Email != "ana@example.com" || m.Subject != "ben@example.com is going to Go and HTMX meetup" ||
		!strings.Contains(m.Text, "23 of 40") {
		t.Fatalf("unexpected message: %+v", m)
	}
}
