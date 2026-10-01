package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type adminStatsUsageRepo struct {
	service.UsageLogRepository
}

func (r *adminStatsUsageRepo) GetGroupStatsWithFilters(context.Context, time.Time, time.Time, int64, int64, int64, int64, *int16, *bool, *int8) ([]usagestats.GroupStat, error) {
	return []usagestats.GroupStat{{
		GroupID:  2,
		Requests: 7,
		Cost:     1.25,
	}}, nil
}

type adminStatsRedeemRepo struct {
	service.RedeemCodeRepository
}

func (r *adminStatsRedeemRepo) GetStats(context.Context) (service.RedeemCodeStats, error) {
	return service.RedeemCodeStats{
		TotalCodes:            4,
		ActiveCodes:           2,
		UsedCodes:             1,
		ExpiredCodes:          1,
		TotalValueDistributed: 10,
		ByType: map[string]int64{
			service.RedeemTypeBalance:    3,
			service.RedeemTypeInvitation: 1,
		},
	}, nil
}

func newAdminStatsDashboardService() *service.DashboardService {
	return service.NewDashboardService(&adminStatsUsageRepo{}, nil, nil, nil)
}

func getAdminJSON(t *testing.T, router http.Handler, path string) map[string]any {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestStatsEndpointsReturnRepositoryData(t *testing.T) {
	router, _ := setupAdminRouter()

	groupBody := getAdminJSON(t, router, "/api/v1/admin/groups/2/stats")
	groupData := groupBody["data"].(map[string]any)
	require.EqualValues(t, 7, groupData["total_requests"])
	require.Equal(t, 1.25, groupData["total_cost"])
	require.Equal(t, "group_api_keys+usage_logs", groupData["data_source"])
	require.Equal(t, false, groupData["partial"])

	redeemBody := getAdminJSON(t, router, "/api/v1/admin/redeem-codes/stats")
	redeemData := redeemBody["data"].(map[string]any)
	require.EqualValues(t, 4, redeemData["total_codes"])
	require.EqualValues(t, 2, redeemData["active_codes"])
	require.EqualValues(t, 1, redeemData["used_codes"])
	require.Equal(t, 10.0, redeemData["total_value_distributed"])
	require.Equal(t, "all_time", redeemData["window"])
	require.Equal(t, "redeem_codes", redeemData["data_source"])
	require.Equal(t, false, redeemData["partial"])
}
