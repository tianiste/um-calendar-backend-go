CREATE TABLE IF NOT EXISTS "calendars" (
    "id" SERIAL PRIMARY KEY,
    "code" TEXT NOT NULL UNIQUE,
    "name" TEXT NOT NULL,
    "ics_url" TEXT NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_calendars_ics_url ON calendars (ics_url);
CREATE INDEX IF NOT EXISTS idx_calendars_name ON calendars (name);