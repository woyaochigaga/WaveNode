package repository

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// GetRouteTraceBilling 按服务端或客户端请求 ID 汇总用量记录。
// 这里只返回记录数和实际扣费，不读取提示词、响应正文或用户凭据。
func (r *opsRepository) GetRouteTraceBilling(ctx context.Context, requestID, clientRequestID string) (service.RouteTraceBillingSummary, error) {
	ids := make([]string, 0, 2)
	for _, id := range []string{requestID, clientRequestID} {
		id = strings.TrimSpace(id)
		if id != "" && (len(ids) == 0 || ids[0] != id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return service.RouteTraceBillingSummary{Status: "not_charged"}, nil
	}

	var summary service.RouteTraceBillingSummary
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*), COALESCE(SUM(actual_cost), 0)
FROM usage_logs
WHERE request_id = ANY($1)
`, pq.Array(ids)).Scan(&summary.RecordCount, &summary.Amount)
	if err != nil {
		return service.RouteTraceBillingSummary{}, err
	}
	summary.Status = "not_charged"
	if summary.RecordCount > 0 {
		summary.Status = "recorded_zero_cost"
	}
	if summary.Amount > 0 {
		summary.Status = "charged"
	}
	return summary, nil
}
