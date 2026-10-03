-- 使用记录详情与计费主表分离，避免请求/响应正文拖慢列表与统计查询。
-- request_id + api_key_id 与 usage_logs 的幂等键一致；异步计费写入前后均可安全 upsert。
CREATE TABLE IF NOT EXISTS usage_log_details (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(128) NOT NULL,
    api_key_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    method VARCHAR(16) NOT NULL DEFAULT '',
    path VARCHAR(512) NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    request_content_type VARCHAR(128) NOT NULL DEFAULT '',
    response_content_type VARCHAR(128) NOT NULL DEFAULT '',
    request_body TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
    request_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    response_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    CONSTRAINT usage_log_details_request_key UNIQUE (request_id, api_key_id)
);

CREATE INDEX IF NOT EXISTS idx_usage_log_details_expires_at
    ON usage_log_details (expires_at);

CREATE INDEX IF NOT EXISTS idx_usage_log_details_user_created
    ON usage_log_details (user_id, created_at DESC);
