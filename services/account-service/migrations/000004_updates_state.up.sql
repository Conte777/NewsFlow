-- Create table for storing Telegram updates state (pts, qts, date, seq)
CREATE TABLE IF NOT EXISTS telegram_updates_state (
    user_id BIGINT PRIMARY KEY,
    pts INT NOT NULL DEFAULT 0,
    qts INT NOT NULL DEFAULT 0,
    date INT NOT NULL DEFAULT 0,
    seq INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create table for storing channel-specific pts
CREATE TABLE IF NOT EXISTS telegram_channel_state (
    user_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    pts INT NOT NULL DEFAULT 0,
    access_hash BIGINT,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, channel_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_telegram_updates_state_updated_at ON telegram_updates_state(updated_at);
CREATE INDEX idx_telegram_channel_state_user_id ON telegram_channel_state(user_id);
CREATE INDEX idx_telegram_channel_state_channel_id ON telegram_channel_state(channel_id);

-- Create triggers for automatic updated_at updates
CREATE TRIGGER update_telegram_updates_state_updated_at
    BEFORE UPDATE ON telegram_updates_state
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_telegram_channel_state_updated_at
    BEFORE UPDATE ON telegram_channel_state
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
