ALTER TABLE account_channels ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;
CREATE INDEX idx_account_channels_is_active ON account_channels(is_active);
