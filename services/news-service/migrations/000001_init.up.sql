-- Create news table
CREATE TABLE IF NOT EXISTS news (
    id SERIAL PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL,
    channel_name VARCHAR(255) NOT NULL,
    message_id INTEGER NOT NULL,
    content TEXT,
    media_urls TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    -- Ensure same message from same channel is not duplicated
    CONSTRAINT uq_channel_message UNIQUE (channel_id, message_id)
);

-- Create delivered_news table
CREATE TABLE IF NOT EXISTS delivered_news (
    id SERIAL PRIMARY KEY,
    news_id INTEGER NOT NULL,
    user_id BIGINT NOT NULL,
    delivered_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    -- Foreign key constraint
    CONSTRAINT fk_delivered_news_news FOREIGN KEY (news_id) REFERENCES news(id) ON DELETE CASCADE,

    -- Ensure same news is not delivered to same user twice
    CONSTRAINT uq_news_user UNIQUE (news_id, user_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_news_channel_id ON news(channel_id);
CREATE INDEX idx_news_created_at ON news(created_at);
CREATE INDEX idx_delivered_news_user_id ON delivered_news(user_id);
CREATE INDEX idx_delivered_news_news_id ON delivered_news(news_id);
CREATE INDEX idx_delivered_news_delivered_at ON delivered_news(delivered_at);

-- Create updated_at trigger for news
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_news_updated_at
    BEFORE UPDATE ON news
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_delivered_news_updated_at
    BEFORE UPDATE ON delivered_news
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
