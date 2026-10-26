package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/FyrmForge/hamr/pkg/auth"
	"github.com/FyrmForge/hamr/pkg/config"
	"github.com/FyrmForge/hamr/pkg/db"
	"github.com/FyrmForge/hamr/pkg/logging"
	"github.com/FyrmForge/hamr/pkg/middleware"
	"github.com/FyrmForge/hamr/pkg/server"
	"github.com/FyrmForge/hamr/pkg/storage"
	stripe "github.com/stripe/stripe-go/v86"
	"github.com/FyrmForge/hamr/pkg/email"
	"github.com/FyrmForge/hamr/pkg/emailmock"
	appdb "github.com/FyrmForge/hamr-example/gather/internal/db"
	"github.com/FyrmForge/hamr-example/gather/internal/repo/postgres"
	"github.com/FyrmForge/hamr-example/gather/internal/service"
	"github.com/FyrmForge/hamr-example/gather/internal/api"
	"github.com/FyrmForge/hamr-example/gather/internal/devactions"
	"github.com/FyrmForge/hamr-example/gather/internal/web"
	"github.com/FyrmForge/hamr-example/gather/internal/web/components"
)

// version is set at build time via ldflags.
var version = "dev"

var (
	envPort          = config.GetEnvOrDefaultInt("PORT", 8080)
	// DEV_MODE defaults to false (fail closed in prod). Local dev sets
	// DEV_MODE=true via .env so the scaffolded `.env` ships with it set
	// explicitly. This makes the STRIPE_MOCK production guard actually
	// guard — a leftover STRIPE_MOCK=true in a prod deploy without
	// DEV_MODE explicitly set would otherwise slip through.
	envDevMode       = config.GetEnvOrDefaultBool("DEV_MODE", false)
	envBaseURL       = config.GetEnvOrDefault("BASE_URL", "")
	envDatabaseURL   = config.GetEnvOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5439/gather?sslmode=disable")
	envStaticBaseURL = config.GetEnvOrDefault("STATIC_BASE_URL", "/static")
	envStoragePath   = config.GetEnvOrDefault("STORAGE_PATH", "./uploads")
	envStripeKey           = config.GetEnvOrDefault("STRIPE_KEY", "")
	envStripeWebhookSecret = config.GetEnvOrDefault("STRIPE_WEBHOOK_SECRET", "")
	// STRIPE_WEBHOOK_SECRET_V2 signs v2 thin events. Under hamr dev and
	// `stripe listen` it equals STRIPE_WEBHOOK_SECRET; a deployed Stripe
	// event destination has its own.
	envStripeWebhookSecretV2 = config.GetEnvOrDefault("STRIPE_WEBHOOK_SECRET_V2", "")
	envStripeMock          = config.GetEnvOrDefaultBool("STRIPE_MOCK", false)
	// HAMR_STRIPE_MOCK_URL is auto-injected by `hamr dev` (derived from
	// hamr.toml [proxy].listen) so the mock URL tracks proxy port changes
	// without manual edits. The fallback is for app instances run outside
	// hamr dev — where STRIPE_MOCK shouldn't be true anyway.
	envHamrStripeMockURL   = config.GetEnvOrDefault("HAMR_STRIPE_MOCK_URL", "http://localhost:3000")
	envEmailMock   = config.GetEnvOrDefaultBool("EMAIL_MOCK", false)
	envHamrDevURL  = config.GetEnvOrDefault("HAMR_DEV_URL", "http://localhost:3000")
	envTrustedProxies = config.GetEnvCSV("TRUSTED_PROXIES")
)

func main() {
	generateFlag := flag.Bool("generate", false, "generate static pages and exit")
	seedFlag := flag.Bool("seed", false, "load the dev seed (internal/devactions/seed.sql) and exit")
	flag.Parse()

	log := logging.New(!envDevMode)
	slog.SetDefault(log)

	components.StaticBaseURL = envStaticBaseURL

	// Base URL (cookie domain & CORS).
	baseOrigin, baseDomain, err := config.ParseBaseURL(envBaseURL)
	if err != nil {
		log.Error("invalid BASE_URL", "error", err)
		os.Exit(1)
	}
	components.BaseURL = baseOrigin

	// Server.
	srv, err := server.New(
		server.WithPort(envPort),
		server.WithDevMode(envDevMode),
		server.WithStaticDir("frontend/static"),
		server.WithStaticDistDir("frontend/dist"),
		server.WithGeneratedDir("generated"),
		// TRUSTED_PROXIES: comma-separated CIDRs of upstream proxies/load
		// balancers allowed to set X-Forwarded-For (drives client-IP detection
		// and the rate-limit key). Empty/unset ignores X-Forwarded-For so a
		// direct client can't spoof its IP; set it to your LB ranges behind one.
		// "cloudflare" trusts Cloudflare's edge ranges, refreshed every 24h.
		server.WithTrustedProxies(envTrustedProxies...),
	)
	if err != nil {
		log.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if baseOrigin != "" {
		srv.Echo().Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{baseOrigin},
			AllowCredentials: true,
		}))
	}

	// Static page generation — no heavy deps needed.
	web.RegisterStaticPages(srv)
	if *generateFlag {
		if err := srv.GenerateStatic("generated"); err != nil {
			log.Error("generate static pages failed", "error", err)
			os.Exit(1)
		}
		return
	}

	// Database.
	connectCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	database, err := db.ConnectContext(connectCtx, envDatabaseURL)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Run migrations at startup.
	if err := db.Migrate(database, appdb.MigrateConfig()); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}
	log.Info("migrations completed")
	store := postgres.NewStore(database)

	if *seedFlag {
		if err := devactions.Seed(context.Background(), database); err != nil {
			log.Error("seed failed", "error", err)
			os.Exit(1)
		}
		log.Info("dev seed loaded")
		return
	}

	// Sessions.
	sessionManager := auth.NewSessionManager(store,
		auth.WithCookieSecure(!envDevMode),
		auth.WithCookieDomain(baseDomain),
	)

	// Auth service.
	authService := service.NewAuthService(store)

	// File storage (local).
	fileStorage, err := storage.NewLocalStorage(envStoragePath)
	if err != nil {
		log.Error("failed to init storage", "error", err)
		os.Exit(1)
	}

	// Stripe SDK setup. With STRIPE_MOCK=true (dev-only), real stripe-go
	// calls are routed to hamr's local mock served on the dev proxy — no
	// abstraction layer in app code. In production leave STRIPE_MOCK unset
	// and stripe-go talks to api.stripe.com directly.
	//
	// Production safety: STRIPE_MOCK=true with DEV_MODE=false is refused
	// at startup. A leftover STRIPE_MOCK=true in a production .env would
	// otherwise route real payment calls to a non-existent localhost mock,
	// silently breaking checkouts. Failing closed loud is the only
	// acceptable behavior. Staging environments that want fake payments
	// should also set DEV_MODE=true.
	//
	// The mock URL is auto-injected by `hamr dev` as HAMR_STRIPE_MOCK_URL,
	// derived from hamr.toml [proxy].listen. The default fallback only
	// applies when running outside hamr dev — where STRIPE_MOCK shouldn't
	// be true anyway.
	stripe.Key = envStripeKey
	if envStripeMock {
		if !envDevMode {
			log.Error("STRIPE_MOCK=true requires DEV_MODE=true (refusing to route Stripe calls to a local mock without dev mode)")
			os.Exit(1)
		}
		stripe.SetBackend(stripe.APIBackend, stripe.GetBackendWithConfig(
			stripe.APIBackend,
			&stripe.BackendConfig{URL: stripe.String(envHamrStripeMockURL)},
		))
		log.Info("stripe SDK pointed at hamr mock", "url", envHamrStripeMockURL)
	}
	// One client for everything, built after SetBackend so it inherits the
	// mock backend. Use its namespaces (V1Transfers, V2CoreAccounts, ...).
	stripeClient := stripe.NewClient(envStripeKey)

	// Email sender. In dev (EMAIL_MOCK=true), ships messages to the hamr dev
	// inbox at HAMR_DEV_URL/__hamr/mail. Swap for a real provider adapter in
	// production.
	var emailSender email.Sender
	if envEmailMock {
		emailSender = emailmock.New(envHamrDevURL)
		log.Info("email mock enabled", "inbox", envHamrDevURL+"/__hamr/mail")
	}

	api.RegisterRoutes(srv, &api.Deps{
		Store: store,
		WebhookSecret:   envStripeWebhookSecret,
		WebhookSecretV2: envStripeWebhookSecretV2,
		StripeClient:    stripeClient,
	})

	web.RegisterRoutes(srv, &web.Deps{
		Store:         store,
		BaseURL:       baseOrigin,
		StaticBaseURL: envStaticBaseURL,
		DevMode:       envDevMode,
		// Nil unless HAMR_ACTIONS_TOKEN and HAMR_ACTIONS_ENV enable it.
		DevActions: devactions.New(database, store, sessionManager).Handler(),
		SessionManager: sessionManager,
		AuthService: authService,
		FileStorage: fileStorage,
		EmailSender: emailSender,
	})

	log.Info("starting server", "port", envPort, "devMode", envDevMode)
	if err := srv.Start(); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
