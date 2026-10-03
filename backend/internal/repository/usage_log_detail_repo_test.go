package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryUpsertUsageLogDetail(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	usageDetailWriteCount.Store(0)

	createdAt := time.Date(2026, 10, 4, 4, 0, 0, 0, time.UTC)
	detail := &service.UsageLogDetail{
		RequestID:           "client:req-1",
		APIKeyID:            2,
		UserID:              3,
		Method:              "POST",
		Path:                "/v1/responses",
		StatusCode:          200,
		RequestContentType:  "application/json",
		ResponseContentType: "text/event-stream",
		RequestBody:         `{"input":"hello"}`,
		ResponseBody:        "data:[DONE]",
		CreatedAt:           createdAt,
	}
	mock.ExpectExec("INSERT INTO usage_log_details").
		WithArgs(
			detail.RequestID, detail.APIKeyID, detail.UserID, detail.Method, detail.Path, detail.StatusCode,
			detail.RequestContentType, detail.ResponseContentType, detail.RequestBody, detail.ResponseBody,
			false, false, createdAt, createdAt.Add(service.UsageDetailRetention),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.UpsertUsageLogDetail(context.Background(), detail))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetUsageLogDetailReturnsNilWhenUnavailable(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	mock.ExpectQuery("FROM usage_log_details").
		WithArgs("client:req-2", int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	detail, err := repo.GetUsageLogDetail(context.Background(), "client:req-2", 4)
	require.NoError(t, err)
	require.Nil(t, detail)
	require.NoError(t, mock.ExpectationsWereMet())
}
