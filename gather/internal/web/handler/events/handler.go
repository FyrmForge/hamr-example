package events

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/FyrmForge/hamr/pkg/middleware"
	"github.com/FyrmForge/hamr/pkg/respond"
	"github.com/FyrmForge/hamr/pkg/validate"
	"github.com/labstack/echo/v4"

	"github.com/FyrmForge/hamr-example/gather/internal/repo"
	"github.com/FyrmForge/hamr-example/gather/internal/service"
	"github.com/FyrmForge/hamr-example/gather/internal/web/components/form"
)

// EventForm holds the new-event form values.
type EventForm struct {
	Title       string `form:"title"`
	Location    string `form:"location"`
	StartsAt    string `form:"starts_at"`
	Capacity    string `form:"capacity"`
	Description string `form:"description"`
}

type handler struct {
	events *service.EventService

	FormRules validate.Form
}

// NewHandler creates the events handler. The rules live here so the same
// FormRules check one field on blur and the whole form on submit.
func NewHandler(events *service.EventService) *handler {
	return &handler{
		events: events,
		FormRules: validate.NewForm(
			validate.WithOOBRenderer(form.OOBValidator),
			validate.WithTrim(true),
			validate.Field("title", validate.Required,
				validate.WithMsg(validate.MinLen(3), "Use at least 3 characters"), validate.MaxLen(120)),
			validate.Field("location", validate.MaxLen(120)),
			validate.Field("starts_at", validate.Required, inTheFuture),
			validate.Field("capacity", validate.Required, positiveInt),
			validate.Field("description", validate.MaxLen(2000)),
		),
	}
}

// GET /
func (h *handler) Index(c echo.Context) error {
	list, err := h.events.Upcoming(c.Request().Context())
	if err != nil {
		return err
	}
	return respond.HTML(c, http.StatusOK, indexPage(c, list))
}

// GET /events/:id
func (h *handler) Show(c echo.Context) error {
	e, err := h.events.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	if e == nil {
		return echo.NewHTTPError(http.StatusNotFound, "event not found")
	}
	return respond.HTML(c, http.StatusOK, showPage(c, e))
}

// GET /events/new
func (h *handler) New(c echo.Context) error {
	return respond.HTML(c, http.StatusOK, newPage(c, EventForm{}, nil))
}

// POST /events
func (h *handler) Create(c echo.Context) error {
	var f EventForm
	if err := c.Bind(&f); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid form data")
	}
	if errs := h.FormRules.Validate(c); errs != nil {
		return respond.HTML(c, http.StatusUnprocessableEntity, eventForm(c, f, errs))
	}
	in := service.NewEvent{
		Title:       f.Title,
		Location:    f.Location,
		StartsAt:    f.StartsAt,
		Capacity:    f.Capacity,
		Description: f.Description,
	}
	e, err := h.events.Create(c.Request().Context(), middleware.GetSubjectID(c), in)
	if err != nil {
		if errors.Is(err, service.ErrInvalidEvent) {
			return respond.HTML(c, http.StatusUnprocessableEntity, eventForm(c, f, map[string]string{
				"general": "Check the details and try again.",
			}))
		}
		return err
	}
	middleware.SetFlash(c, "Event created.", middleware.FlashSuccess)
	return respond.Redirect(c, "/events/"+e.ID)
}

// positiveInt is a custom rule: any func(string) string is a validate.Rule.
func positiveInt(v string) string {
	if n, err := strconv.Atoi(v); err != nil || n < 1 {
		return "Enter a number of places above 0"
	}
	return ""
}

// inTheFuture accepts a datetime-local value that has not passed yet.
func inTheFuture(v string) string {
	t, err := time.Parse(service.StartsAtLayout, v)
	if err != nil {
		return "Pick a date and time"
	}
	if !t.After(time.Now()) {
		return "Pick a time in the future"
	}
	return ""
}

// when formats an event's start, e.g. "Sat 14 Nov, 19:00".
func when(e *repo.Event) string {
	return e.StartsAt.Format("Mon 2 Jan, 15:04")
}
