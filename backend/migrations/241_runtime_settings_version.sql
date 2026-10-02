-- 241_runtime_settings_version.sql
-- 初始化运行时设置版本契约使用的 CAS 元数据行。
-- 首次版本化写入会按现有设置计算真实内容哈希，因此兼容已有任意配置。
INSERT INTO settings (key, value, updated_at)
VALUES ('__sub2api_runtime_settings_version', '{"version":"legacy"}', NOW())
ON CONFLICT (key) DO NOTHING;
