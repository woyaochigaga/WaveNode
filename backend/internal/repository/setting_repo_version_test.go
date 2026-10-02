package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

type settingVersionQueryMatcher struct {
	queries *[]string
}

func (m settingVersionQueryMatcher) Match(_, actual string) error {
	*m.queries = append(*m.queries, actual)
	return nil
}

func TestSettingRepositoryVersionedWriteLocksVersionRow(t *testing.T) {
	var queries []string
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(settingVersionQueryMatcher{queries: &queries}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))
	t.Cleanup(func() { _ = client.Close() })
	repo := NewSettingRepository(client).(*settingRepository)
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("version row").WillReturnRows(
		sqlmock.NewRows(setting.Columns).AddRow(int64(1), "__sub2api_runtime_settings_version", `{"version":"legacy"}`, now),
	)
	mock.ExpectQuery("all settings").WillReturnError(errors.New("stop after lock assertion"))
	mock.ExpectRollback()

	_, err = repo.SetMultipleWithVersion(context.Background(), map[string]string{"feature": "on"}, "expected", "next", `{}`)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.NotEmpty(t, queries)
	require.Contains(t, strings.ToUpper(normalizeSQLWhitespace(queries[0])), "FOR UPDATE")
}
