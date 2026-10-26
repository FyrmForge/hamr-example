package repo

import (
	"context"

	"github.com/FyrmForge/hamr/pkg/auth"
)

// Store defines the data access interface for the application.
//
// Data access is grouped into one sub-store per entity, reached through an
// accessor: Store.Users().GetByID(ctx, id). Add a new entity by defining its
// interface next to its model (see UserStore in user.go), adding an accessor
// here, and implementing it in the database packages.
//
// Only internal/service should hold a Store. Handlers call services — except
// health probes and the route wiring in server.go, which reach it directly.
// See "Layers" in AGENTS.md.
type Store interface {
	// Health checks the database connection.
	Health(ctx context.Context) error

	// Session persistence — satisfies auth.SessionStore.
	auth.SessionStore

	// Users returns the user sub-store.
	Users() UserStore

	// Events returns the event sub-store.
	Events() EventStore
}
