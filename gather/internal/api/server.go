package api

import (
	"github.com/FyrmForge/hamr/pkg/server"
	"github.com/stripe/stripe-go/v86"

	"github.com/FyrmForge/hamr-example/gather/internal/middleware"
	"github.com/FyrmForge/hamr-example/gather/internal/repo"
	"github.com/FyrmForge/hamr-example/gather/internal/api/handler/health"
	stripehandler "github.com/FyrmForge/hamr-example/gather/internal/api/handler/stripe"
)

// Deps holds the dependencies for API route registration.
type Deps struct {
	Store repo.Store
	WebhookSecret   string // signs v1 snapshot events
	WebhookSecretV2 string // signs v2 thin event notifications
	StripeClient    *stripe.Client
}

// RegisterRoutes registers all API route handlers on the server.
func RegisterRoutes(srv *server.Server, deps *Deps) {
	api := srv.Echo().Group("/api")
	api.Use(middleware.Logging())

	healthHandler := health.NewHandler(deps.Store)
	api.GET("/health", healthHandler.Health)

	stripeHandler := stripehandler.NewHandler(deps.WebhookSecret, deps.WebhookSecretV2, deps.StripeClient)
	api.POST("/webhooks/stripe", stripeHandler.HandleWebhook)
	api.POST("/webhooks/stripe/v2", stripeHandler.HandleThinWebhook)
}
