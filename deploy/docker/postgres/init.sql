-- FixPhone bootstrap for the official postgres docker image.
-- Runs once on an empty volume, as the POSTGRES_USER superuser.
--
-- The application schema itself is owned by golang-migrate
-- (backend/migrations). This file only prepares extensions and the
-- disposable test database.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

SELECT 'CREATE DATABASE fixphone_test OWNER ' || quote_ident(current_user)
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'fixphone_test')\gexec

COMMENT ON DATABASE postgres IS 'FixPhone bootstrap: schema via backend/cmd/cli migrate up';
