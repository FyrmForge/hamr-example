package web

import (
	"context"
	"net/http"

	"github.com/FyrmForge/hamr/pkg/auth"
	"github.com/FyrmForge/hamr/pkg/devactions"
	"github.com/FyrmForge/hamr/pkg/email"
	hamrmw "github.com/FyrmForge/hamr/pkg/middleware"
	"github.com/FyrmForge/hamr/pkg/server"
	"github.com/FyrmForge/hamr/pkg/storage"
	"github.com/labstack/echo/v4"

	"github.com/FyrmForge/hamr-example/gather/internal/middleware"
	"github.com/FyrmForge/hamr-example/gather/internal/repo"
	"github.com/FyrmForge/hamr-example/gather/internal/service"
	"github.com/FyrmForge/hamr-example/gather/internal/web/components"
	"github.com/FyrmForge/hamr-example/gather/internal/web/handler/about"
	"github.com/FyrmForge/hamr-example/gather/internal/web/handler/auth/login"
	"github.com/FyrmForge/hamr-example/gather/internal/web/handler/auth/register"
	"github.com/FyrmForge/hamr-example/gather/internal/web/handler/devemail"
	"github.com/FyrmForge/hamr-example/gather/internal/web/handler/events"
	"github.com/FyrmForge/hamr-example/gather/internal/web/handler/home"
)

// Deps holds the dependencies for route registration.
type Deps struct {
	Store         repo.Store
	BaseURL       string
	StaticBaseURL string
	DevMode       bool

	// DevActions is the hamr dev/staging actions endpoint (see
	// internal/devactions). Nil unless HAMR_ACTIONS_TOKEN and
	// HAMR_ACTIONS_ENV enable it.
	DevActions http.Handler

	SessionManager *auth.SessionManager

	AuthService *service.AuthService

	EventService *service.EventService

	FileStorage storage.FileStorage

	// EmailSender is the outbound mail transport. In dev it points at the
	// hamr email inbox at /__hamr/mail (via pkg/emailmock); in production
	// swap in a real provider adapter that satisfies email.Sender.
	EmailSender email.Sender
}

// RegisterRoutes registers all web route handlers on the server.
func RegisterRoutes(srv *server.Server, deps *Deps) {
	e := srv.Echo()

	// Dev actions endpoint for the hamr widget. Registered on the root echo
	// instance, not the site group, so the site CSRF, session and auth
	// middleware never run for it; it authenticates with its own token.
	if deps.DevActions != nil {
		h := echo.WrapHandler(deps.DevActions)
		e.Any(devactions.Prefix, h)
		e.Any(devactions.Prefix+"/*", h)
	}

	// Content Security Policy.
	csp := "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"

	// Site routes.
	site := e.Group("")
	site.Use(middleware.Logging())
	site.Use(hamrmw.ErrorPages(components.ErrorPage))
	site.Use(hamrmw.SecureWithConfig(hamrmw.SecureConfig{
		ContentSecurityPolicy: csp,
	}))
	site.Use(hamrmw.FlashWithConfig(hamrmw.FlashConfig{Secure: !deps.DevMode}))
	site.Use(hamrmw.CSRFWithConfig(hamrmw.CSRFConfig{Secure: !deps.DevMode}))

	auth := hamrmw.NewBrowserAuth(deps.SessionManager,
		hamrmw.WithSubjectLoader(func(reqCtx context.Context, id string) (any, error) {
			return deps.Store.Users().GetByID(reqCtx, id)
		}),
		hamrmw.WithLoginRedirect("/login"),
		hamrmw.WithHomeRedirect("/"),
	)
	site.Use(auth.Load())

	homeHandler := home.NewHandler()
	site.GET("/", homeHandler.Index)

	// Dev-only sample email endpoint. Sends a test message through the
	// configured email.Sender so you can smoke-test the /__hamr/mail inbox.
	// Gated on DevMode; no-op in production regardless of route registration.
	if deps.EmailSender != nil {
		devmailHandler := devemail.NewHandler(deps.EmailSender, deps.DevMode)
		site.GET("/dev/send-test-email", devmailHandler.SendTest)
	}

	// Auth routes — one page-package per page (login owns logout as its inverse action).
	loginHandler := login.NewHandler(deps.AuthService, deps.SessionManager)
	site.GET("/login", loginHandler.Page, auth.RequireNotAuth())
	site.POST("/login", loginHandler.Submit, auth.RequireNotAuth())
	site.POST("/login/validate/:field", loginHandler.FormRules.ValidationHandler("field"), auth.RequireNotAuth())
	site.POST("/logout", loginHandler.Logout, auth.RequireAuth())

	registerHandler := register.NewHandler(deps.AuthService, deps.SessionManager)
	site.GET("/register", registerHandler.Page, auth.RequireNotAuth())
	site.POST("/register", registerHandler.Submit, auth.RequireNotAuth())
	site.POST("/register/validate/:field", registerHandler.FormRules.ValidationHandler("field"), auth.RequireNotAuth())

	eventsHandler := events.NewHandler(deps.EventService)
	site.GET("/events/new", eventsHandler.New, auth.RequireAuth())
	site.POST("/events", eventsHandler.Create, auth.RequireAuth())
	site.POST("/events/validate/:field", eventsHandler.FormRules.ValidationHandler("field"), auth.RequireAuth())
}

// RegisterStaticPages registers handlers for static generation and runtime
// serving. Each call to StaticPage registers both a generation entry and a
// GET route. These handlers must not depend on database or session state.
func RegisterStaticPages(srv *server.Server) {
	h := about.NewHandler()
	srv.StaticPage("/about", h.About)
}
