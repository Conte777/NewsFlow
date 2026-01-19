-- Restore is_active column for rollback
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Set is_active based on status
UPDATE subscriptions SET is_active = (status = 'active');
