-- Drop trigger first
DROP TRIGGER IF EXISTS update_account_channels_updated_at ON account_channels;

-- Drop table
DROP TABLE IF EXISTS account_channels;
