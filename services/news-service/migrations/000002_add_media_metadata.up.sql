-- Add media_metadata column to news table for storing media type and attributes
ALTER TABLE news ADD COLUMN IF NOT EXISTS media_metadata TEXT;

-- Add comment explaining the field
COMMENT ON COLUMN news.media_metadata IS 'JSON-encoded array of media metadata (type, width, height, duration)';
