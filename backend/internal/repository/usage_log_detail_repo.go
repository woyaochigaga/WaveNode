package repository

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var usageDetailWriteCount atomic.Uint64

// UpsertUsageLogDetail 独立写入正文详情；计费任务可异步完成，不要求 usage_logs 已先存在。
func (r *usageLogRepository) UpsertUsageLogDetail(ctx context.Context, detail *service.UsageLogDetail) error {
	if detail == nil {
		return nil
	}
	now := detail.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	expiresAt := detail.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = now.Add(service.UsageDetailRetention)
	}
	query := `
		INSERT INTO usage_log_details (
			request_id, api_key_id, user_id, method, path, status_code,
			request_content_type, response_content_type, request_body, response_body,
			request_truncated, response_truncated, created_at, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (request_id, api_key_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			method = EXCLUDED.method,
			path = EXCLUDED.path,
			status_code = EXCLUDED.status_code,
			request_content_type = EXCLUDED.request_content_type,
			response_content_type = EXCLUDED.response_content_type,
			request_body = EXCLUDED.request_body,
			response_body = EXCLUDED.response_body,
			request_truncated = EXCLUDED.request_truncated,
			response_truncated = EXCLUDED.response_truncated,
			created_at = EXCLUDED.created_at,
			expires_at = EXCLUDED.expires_at
	`
	_, err := r.sql.ExecContext(ctx, query,
		detail.RequestID, detail.APIKeyID, detail.UserID, detail.Method, detail.Path, detail.StatusCode,
		detail.RequestContentType, detail.ResponseContentType, detail.RequestBody, detail.ResponseBody,
		detail.RequestTruncated, detail.ResponseTruncated, now, expiresAt,
	)
	if err != nil {
		return err
	}

	// 每 100 次写入顺带分批清理过期正文，避免为短保留期另起常驻 worker。
	if usageDetailWriteCount.Add(1)%100 == 0 {
		_, _ = r.sql.ExecContext(ctx, `
			WITH expired AS (
				SELECT id FROM usage_log_details WHERE expires_at < NOW() ORDER BY expires_at LIMIT 5000
			)
			DELETE FROM usage_log_details WHERE id IN (SELECT id FROM expired)
		`)
	}
	return nil
}

// GetUsageLogDetail 读取未过期详情；历史记录或已过期正文返回 nil 而不是 404。
func (r *usageLogRepository) GetUsageLogDetail(ctx context.Context, requestID string, apiKeyID int64) (*service.UsageLogDetail, error) {
	query := `
		SELECT id, request_id, api_key_id, user_id, method, path, status_code,
			request_content_type, response_content_type, request_body, response_body,
			request_truncated, response_truncated, created_at, expires_at
		FROM usage_log_details
		WHERE request_id = $1 AND api_key_id = $2 AND expires_at > NOW()
	`
	var detail service.UsageLogDetail
	err := scanSingleRow(ctx, r.sql, query, []any{requestID, apiKeyID},
		&detail.ID, &detail.RequestID, &detail.APIKeyID, &detail.UserID, &detail.Method, &detail.Path, &detail.StatusCode,
		&detail.RequestContentType, &detail.ResponseContentType, &detail.RequestBody, &detail.ResponseBody,
		&detail.RequestTruncated, &detail.ResponseTruncated, &detail.CreatedAt, &detail.ExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &detail, nil
}
