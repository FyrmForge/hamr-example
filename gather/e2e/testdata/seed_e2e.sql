-- E2E test seed data.
-- All test accounts use password: Test1234!
-- Argon2id hash (m=65536,t=3,p=2) with deterministic salt "e2e_test_salt!!!"

INSERT INTO users (id, email, password_hash, name, role, active, created_at, updated_at)
VALUES
    ('e2e-admin-001', 'admin@test.com', '$argon2id$v=19$m=65536,t=3,p=2$ZTJlX3Rlc3Rfc2FsdCEhIQ$bomHV69Vpgt6LTl83QWN02LhEdsWsb9+b6ZC2VYML9w', 'Test Admin', 'admin', true, NOW(), NOW()),
    ('e2e-user-001', 'user@test.com', '$argon2id$v=19$m=65536,t=3,p=2$ZTJlX3Rlc3Rfc2FsdCEhIQ$bomHV69Vpgt6LTl83QWN02LhEdsWsb9+b6ZC2VYML9w', 'Test User', 'user', true, NOW(), NOW())
-- Any conflict, not just (id): the dev seed (internal/devactions/seed.sql) creates the
-- same emails under other ids, and `make e2e-local` runs against that database.
ON CONFLICT DO NOTHING;
