-- Add status column to subscriptions table for Saga pattern support
-- Possible values: 'pending', 'active', 'removing'

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';

-- Update existing records to have 'active' status
UPDATE subscriptions SET status = 'active' WHERE status IS NULL;

-- Add index for status column
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);

-- Remove is_active column as it's replaced by status
-- Note: We keep is_active for backward compatibility during migration
-- It will be removed in a future migration after all services are updated
