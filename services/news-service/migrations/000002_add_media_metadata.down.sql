-- Remove media_metadata column from news table
ALTER TABLE news DROP COLUMN IF EXISTS media_metadata;
