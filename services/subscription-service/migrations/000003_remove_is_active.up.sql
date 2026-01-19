-- Remove deprecated is_active column (replaced by status column in migration 000002)
ALTER TABLE subscriptions DROP COLUMN IF EXISTS is_active;
