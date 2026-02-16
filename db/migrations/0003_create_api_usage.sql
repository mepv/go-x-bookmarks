CREATE TABLE IF NOT EXISTS api_usage (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    usage_day DATE NOT NULL,
    x_calls_count INTEGER NOT NULL DEFAULT 0 CHECK (x_calls_count >= 0),
    x_estimated_cost_units NUMERIC(12,4) NOT NULL DEFAULT 0 CHECK (x_estimated_cost_units >= 0),
    notion_writes_count INTEGER NOT NULL DEFAULT 0 CHECK (notion_writes_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, usage_day)
);

CREATE INDEX IF NOT EXISTS idx_api_usage_usage_day
    ON api_usage (usage_day);
