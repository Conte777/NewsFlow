-- Drop triggers
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
DROP TRIGGER IF EXISTS update_accounts_updated_at ON accounts;

-- Drop tables
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS accounts;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();
