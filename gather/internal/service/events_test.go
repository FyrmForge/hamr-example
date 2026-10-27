package service

import (
	"errors"
	"testing"
	"time"
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
