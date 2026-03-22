DROP INDEX IF EXISTS idx_calendars_last_checked_at;

ALTER TABLE calendars
DROP COLUMN IF EXISTS last_checked_at,
DROP COLUMN IF EXISTS content_hash,
DROP COLUMN IF EXISTS last_modified,
DROP COLUMN IF EXISTS etag;
