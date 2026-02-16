CREATE TABLE IF NOT EXISTS user_sync_state (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_bookmark_cursor TEXT,
    last_seen_tweet_id TEXT,
    last_sync_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_error TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_sync_state_last_success_at
    ON user_sync_state (last_success_at);
