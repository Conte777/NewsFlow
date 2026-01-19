-- Drop triggers
DROP TRIGGER IF EXISTS update_telegram_channel_state_updated_at ON telegram_channel_state;
DROP TRIGGER IF EXISTS update_telegram_updates_state_updated_at ON telegram_updates_state;

-- Drop indexes
DROP INDEX IF EXISTS idx_telegram_channel_state_channel_id;
DROP INDEX IF EXISTS idx_telegram_channel_state_user_id;
DROP INDEX IF EXISTS idx_telegram_updates_state_updated_at;

-- Drop tables
DROP TABLE IF EXISTS telegram_channel_state;
DROP TABLE IF EXISTS telegram_updates_state;
