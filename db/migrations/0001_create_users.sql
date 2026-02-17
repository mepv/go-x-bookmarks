CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    x_user_id TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT,
    notion_workspace_id TEXT,
    metadata_fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata_expires_at TIMESTAMPTZ NOT NULL,
    last_successful_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_metadata_expires_at
    ON users (metadata_expires_at);
