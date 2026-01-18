-- Rollback: Remove status column from subscriptions table

DROP INDEX IF EXISTS idx_subscriptions_status;

ALTER TABLE subscriptions DROP COLUMN IF EXISTS status;
