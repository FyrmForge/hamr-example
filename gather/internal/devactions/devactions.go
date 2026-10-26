// Package devactions registers the dev/staging actions hamr shows in its
// widget (wipe and reseed, create account, log everyone out) plus the seeded
// test logins. The endpoint is mounted by internal/web and only exists when
// HAMR_ACTIONS_TOKEN is set and HAMR_ACTIONS_ENV is development or staging.
package devactions

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	hamrdev "github.com/FyrmForge/hamr/pkg/devactions"
	"github.com/FyrmForge/hamr/pkg/auth"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/FyrmForge/hamr-example/gather/internal/repo"
)

// seedSQL is the dev seed for this database driver. It also creates the
// hamr_disposable marker table that unlocks the dangerous actions.
//
//go:embed seed.sql
var seedSQL string

// defaultPassword is the password of the seeded accounts and the default for
// accounts created through the create-account action.
const defaultPassword = "Test1234!"

// accounts are the logins seeded by seed.sql, offered by the widget's login
// helper. Keep in sync with seed.sql (the generator tests check this).
var accounts = []hamrdev.Account{
	{Email: "admin@test.com", Password: defaultPassword, Label: "admin"},
	{Email: "user@test.com", Password: defaultPassword, Label: "user"},
}

// New builds the registry of dev actions.
func New(db *sqlx.DB, store repo.Store, sessions *auth.SessionManager) *hamrdev.Registry {
	r := &hamrdev.Registry{
		Disposable: func(ctx context.Context) (bool, error) { return disposable(ctx, db) },
		Accounts: func(context.Context) ([]hamrdev.Account, error) { return accounts, nil },
		// The widget's "Logged in as" section and per-user inbox links.
		WhoAmI: hamrdev.SessionWhoAmI(sessions, func(ctx context.Context, id string) (*hamrdev.Account, error) {
			u, err := store.Users().GetByID(ctx, id)
			if err != nil || u == nil {
				return nil, err
			}
			return &hamrdev.Account{Email: u.Email, Label: u.Name}, nil
		}),
	}

	r.Register(hamrdev.Action{
		Name:        "reseed",
		Label:       "Wipe & reseed",
		Description: "Deletes all users and sessions, then reloads the dev seed",
		Dangerous:   true,
		Run: func(ctx context.Context, _ map[string]string) (string, error) {
			// One transaction: a failed seed must not leave the database half wiped.
			err := inTx(ctx, db, func(tx execer) error {
				if err := exec(ctx, tx, "DELETE FROM sessions"); err != nil {
					return fmt.Errorf("delete sessions: %w", err)
				}
				if err := exec(ctx, tx, "DELETE FROM users"); err != nil {
					return fmt.Errorf("delete users: %w", err)
				}
				return seed(ctx, tx)
			})
			if err != nil {
				return "", err
			}
			return "Database wiped and reseeded", nil
		},
	})

	r.Register(hamrdev.Action{
		Name:        "create-account",
		Label:       "Create account",
		Description: "Creates a user with the given role",
		Inputs: []hamrdev.Input{
			{Name: "email", Label: "Email", Type: hamrdev.InputText, Required: true},
			{Name: "name", Label: "Name", Type: hamrdev.InputText},
			{Name: "role", Label: "Role", Type: hamrdev.InputSelect, Options: []string{"admin", "user"}, Default: "user"},
			{Name: "password", Label: "Password", Type: hamrdev.InputText, Default: defaultPassword},
		},
		Run: func(ctx context.Context, in map[string]string) (string, error) {
			email := strings.ToLower(strings.TrimSpace(in["email"]))
			if email == "" || !strings.Contains(email, "@") {
				return "", fmt.Errorf("invalid email %q", email)
			}
			role := in["role"]
			if role == "" {
				role = "user"
			}
			if role != "admin" && role != "user" {
				return "", fmt.Errorf("invalid role %q", role)
			}
			password := in["password"]
			if password == "" {
				password = defaultPassword
			}

			existing, err := store.Users().GetByEmail(ctx, email)
			if err != nil {
				return "", fmt.Errorf("check existing user: %w", err)
			}
			if existing != nil {
				return "", fmt.Errorf("%s already exists", email)
			}
			hash, err := auth.HashPassword(password)
			if err != nil {
				return "", fmt.Errorf("hash password: %w", err)
			}
			now := time.Now()
			err = store.Users().Create(ctx, &repo.User{
				ID:           uuid.New().String(),
				Email:        email,
				PasswordHash: hash,
				Name:         strings.TrimSpace(in["name"]),
				Role:         role,
				Active:       true,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
			if err != nil {
				return "", fmt.Errorf("create user: %w", err)
			}
			return "Created " + email, nil
		},
	})

	r.Register(hamrdev.Action{
		Name:        "logout-everyone",
		Label:       "Log everyone out",
		Description: "Deletes every session",
		Run: func(ctx context.Context, _ map[string]string) (string, error) {
			if err := exec(ctx, db, "DELETE FROM sessions"); err != nil {
				return "", fmt.Errorf("delete sessions: %w", err)
			}
			return "All sessions deleted", nil
		},
	})

	return r
}

// Seed runs the statements of seed.sql (also exposed as `make seed`). The
// file holds plain statements separated by semicolons, with only whole-line
// "--" comments. Safe to re-run; runs in one transaction.
func Seed(ctx context.Context, db *sqlx.DB) error {
	return inTx(ctx, db, func(tx execer) error { return seed(ctx, tx) })
}

func seed(ctx context.Context, db execer) error {
	var sb strings.Builder
	for _, line := range strings.Split(seedSQL, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}
	for _, stmt := range strings.Split(sb.String(), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if err := exec(ctx, db, stmt); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
	}
	return nil
}

// execer is the database or an open transaction.
type execer = sqlx.ExecerContext

func exec(ctx context.Context, db execer, query string) error {
	_, err := db.ExecContext(ctx, query)
	return err
}

func inTx(ctx context.Context, db *sqlx.DB, fn func(tx execer) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// disposable reports whether the hamr_disposable marker table exists.
func disposable(ctx context.Context, db *sqlx.DB) (bool, error) {
	var ok bool
	err := db.QueryRowContext(ctx, disposableQuery).Scan(&ok)
	return ok, err
}

const disposableQuery = `SELECT to_regclass('hamr_disposable') IS NOT NULL`
