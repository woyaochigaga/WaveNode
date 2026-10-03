-- 操作审计补充请求/响应类型与脱敏后的响应正文。
-- 正文仅用于单条详情，列表查询不会读取这些大字段。
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS request_content_type VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS response_content_type VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS response_body TEXT NOT NULL DEFAULT '';
