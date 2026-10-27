//go:build e2e

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/FyrmForge/hamr/pkg/e2e"
	"github.com/go-rod/rod/lib/input"
	"github.com/stretchr/testify/require"
)

// TestGatherFlow walks the main path a new user takes: sign up, create an
// event (tripping an inline validation error on the way), RSVP without a
// page reload, then cancel.
func TestGatherFlow(t *testing.T) {
	browser := e2e.SetupBrowser(t)
	htmxWait := 5 * time.Second

	// Sign up.
	email := fmt.Sprintf("e2e-%d@example.com", time.Now().UnixNano())
	page := openPage(t, browser, "/register")
	e2e.Input(t, page, "#name", "E2E Organiser")
	e2e.Input(t, page, "#email", email)
	e2e.Input(t, page, "#password", "Gather123!")
	e2e.Click(t, page, "button[type=submit]")
	e2e.WaitForElement(t, page, "[data-testid=nav-user]", 10*time.Second)
	e2e.AssertElementContainsText(t, page, "[data-testid=nav-user]", email)

	// A too-short title shows its error when the field loses focus.
	page = openPage(t, browser, "/events/new")
	e2e.Input(t, page, "#title", "Go")
	page.Keyboard.MustType(input.Tab)
	e2e.WaitForHTMXIdle(t, page, htmxWait)
	e2e.AssertElementContainsText(t, page, "#error-title", "Use at least 3 characters")

	// A valid event lands on its own page.
	e2e.Input(t, page, "#title", "E2E meetup")
	e2e.Input(t, page, "#location", "Test lab")
	starts := time.Now().AddDate(0, 1, 0).Truncate(time.Minute)
	require.NoError(t, page.MustElement("#starts_at").InputTime(starts))
	e2e.Input(t, page, "#capacity", "25")
	e2e.Click(t, page, "button[type=submit]")
	e2e.WaitForElement(t, page, "[data-testid=rsvp]", 10*time.Second)
	e2e.AssertURLContains(t, page, "/events/")
	e2e.AssertElementContainsText(t, page, "h1", "E2E meetup")
	e2e.AssertElementContainsText(t, page, "[data-testid=rsvp-count]", "0 of 25 going")

	// RSVP swaps the button and the count in place; the page is not reloaded.
	page.MustEval(`() => { window.__noReload = true }`)
	e2e.ClickAndWaitHTMX(t, page, "[data-testid=rsvp]", htmxWait)
	e2e.AssertElementContainsText(t, page, "[data-testid=rsvp-count]", "1 of 25 going")
	e2e.AssertElementContainsText(t, page, "[data-testid=rsvp]", "Cancel")
	require.True(t, page.MustEval(`() => window.__noReload === true`).Bool(), "RSVP reloaded the page")

	// And again to cancel.
	e2e.ClickAndWaitHTMX(t, page, "[data-testid=rsvp]", htmxWait)
	e2e.AssertElementContainsText(t, page, "[data-testid=rsvp-count]", "0 of 25 going")

	// The event is listed on the home page.
	page = openPage(t, browser, "/")
	e2e.AssertElementContainsText(t, page, "[data-testid=event-list]", "E2E meetup")
}

// TestNewEvent_RequiresLogin sends signed-out visitors to the login page.
func TestNewEvent_RequiresLogin(t *testing.T) {
	browser := e2e.SetupBrowser(t)
	page := openPage(t, browser, "/events/new")
	e2e.AssertURLContains(t, page, "/login")
}
