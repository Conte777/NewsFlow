-- Drop trigger
DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Drop indexes
DROP INDEX IF EXISTS idx_subscriptions_is_active;
DROP INDEX IF EXISTS idx_subscriptions_channel_id;
DROP INDEX IF EXISTS idx_subscriptions_user_id;

-- Drop table
DROP TABLE IF EXISTS subscriptions;
