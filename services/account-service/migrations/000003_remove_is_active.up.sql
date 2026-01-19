-- Удалить неактивные записи перед удалением колонки
DELETE FROM account_channels WHERE is_active = false;

-- Удалить индекс и колонку
DROP INDEX IF EXISTS idx_account_channels_is_active;
ALTER TABLE account_channels DROP COLUMN IF EXISTS is_active;
