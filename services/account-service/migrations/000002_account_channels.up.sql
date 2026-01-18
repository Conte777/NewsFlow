-- Create account_channels table for storing account-channel subscriptions
CREATE TABLE IF NOT EXISTS account_channels (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    channel_id VARCHAR(255) NOT NULL,
    channel_name VARCHAR(255) NOT NULL DEFAULT '',
    last_processed_message_id INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_account_channel UNIQUE (account_id, channel_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_account_channels_account_id ON account_channels(account_id);
CREATE INDEX idx_account_channels_channel_id ON account_channels(channel_id);
CREATE INDEX idx_account_channels_is_active ON account_channels(is_active);

-- Create trigger for automatic updated_at updates
CREATE TRIGGER update_account_channels_updated_at
    BEFORE UPDATE ON account_channels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
