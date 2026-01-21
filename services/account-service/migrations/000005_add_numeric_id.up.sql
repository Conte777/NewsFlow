-- Add numeric_id column to store Telegram's numeric channel ID
-- This is used for fallback lookup when channel username is not available
ALTER TABLE account_channels ADD COLUMN numeric_id BIGINT NOT NULL DEFAULT 0;

-- Create index for fast lookup by numeric_id
CREATE INDEX idx_account_channels_numeric_id ON account_channels(numeric_id);
