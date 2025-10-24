-- Drop triggers
DROP TRIGGER IF EXISTS update_delivered_news_updated_at ON delivered_news;
DROP TRIGGER IF EXISTS update_news_updated_at ON news;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Drop indexes
DROP INDEX IF EXISTS idx_delivered_news_delivered_at;
DROP INDEX IF EXISTS idx_delivered_news_news_id;
DROP INDEX IF EXISTS idx_delivered_news_user_id;
DROP INDEX IF EXISTS idx_news_created_at;
DROP INDEX IF EXISTS idx_news_channel_id;

-- Drop tables
DROP TABLE IF EXISTS delivered_news;
DROP TABLE IF EXISTS news;
