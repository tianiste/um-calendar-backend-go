ALTER TABLE calendars
ADD COLUMN IF NOT EXISTS etag TEXT,
ADD COLUMN IF NOT EXISTS last_modified TEXT,
ADD COLUMN IF NOT EXISTS content_hash TEXT,
ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_calendars_last_checked_at ON calendars (last_checked_at);
