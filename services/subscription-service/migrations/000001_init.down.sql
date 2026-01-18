DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;
DROP TRIGGER IF EXISTS update_channels_updated_at ON channels;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS update_updated_at_column;
