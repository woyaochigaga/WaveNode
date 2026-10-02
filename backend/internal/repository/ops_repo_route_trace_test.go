package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryGetRouteTraceBilling(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\), COALESCE\(SUM\(actual_cost\), 0\).*FROM usage_logs.*request_id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count", "actual_cost"}).AddRow(2, 1.25))

	summary, err := repo.GetRouteTraceBilling(context.Background(), "req-1", "client-1")
	require.NoError(t, err)
	require.Equal(t, "charged", summary.Status)
	require.Equal(t, 1.25, summary.Amount)
	require.EqualValues(t, 2, summary.RecordCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetRouteTraceBillingDistinguishesZeroCostRecord(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\), COALESCE\(SUM\(actual_cost\), 0\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count", "actual_cost"}).AddRow(1, 0))

	summary, err := repo.GetRouteTraceBilling(context.Background(), "req-1", "req-1")
	require.NoError(t, err)
	require.Equal(t, "recorded_zero_cost", summary.Status)
	require.EqualValues(t, 1, summary.RecordCount)
	require.NoError(t, mock.ExpectationsWereMet())
}
