-- Initial custodian schema: the sole source of truth for all content.

CREATE TABLE log (
    slug         TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    subtitle     TEXT,
    description  TEXT,
    cover_image  TEXT,
    reading_time INTEGER,
    tags         TEXT NOT NULL DEFAULT '[]',
    body         TEXT NOT NULL DEFAULT '',
    state        TEXT NOT NULL DEFAULT 'unlisted' CHECK (state IN ('listed', 'unlisted')),
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE INDEX idx_log_state_updated ON log (state, updated_at);

CREATE TABLE profile (
    key  TEXT PRIMARY KEY,
    body TEXT NOT NULL
);

CREATE TABLE media (
    key          TEXT PRIMARY KEY,
    state        TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'available')),
    content_type TEXT NOT NULL,
    url          TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    expires_at   TEXT
);

-- Append-on-change timeseries: one row per distinct polled state, timestamped.
-- The read surface serves the latest row per source; idle polls insert nothing.
CREATE TABLE integration (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    source     TEXT NOT NULL,
    data       TEXT,
    etag       TEXT,
    fetched_at TEXT NOT NULL
);

CREATE INDEX idx_integration_source_id ON integration (source, id);
