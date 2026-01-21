-- Remove numeric_id column
DROP INDEX IF EXISTS idx_account_channels_numeric_id;
ALTER TABLE account_channels DROP COLUMN IF EXISTS numeric_id;
