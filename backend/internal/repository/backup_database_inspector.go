package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

var backupCriticalTables = []string{
	"users",
	"accounts",
	"api_keys",
	"groups",
	"channels",
	"payment_orders",
	"redeem_codes",
	"security_secrets",
}

// InspectDatabase 返回备份清单与迁移 preflight 使用的只读数据库快照。
func (d *PgDumper) InspectDatabase(ctx context.Context) (*service.BackupDatabaseSnapshot, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("inspect backup database: nil sql db")
	}
	if err := d.db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping backup database: %w", err)
	}
	snapshot, err := inspectBackupDatabase(ctx, d.db)
	if err != nil {
		return nil, err
	}
	snapshot.TargetMigration, err = latestEmbeddedMigration()
	if err != nil {
		return nil, err
	}
	_, dumpErr := exec.LookPath("pg_dump")
	_, psqlErr := exec.LookPath("psql")
	snapshot.PGDumpAvailable = dumpErr == nil
	snapshot.PSQLAvailable = psqlErr == nil
	return snapshot, nil
}

func inspectBackupDatabase(ctx context.Context, db *sql.DB) (*service.BackupDatabaseSnapshot, error) {
	snapshot := &service.BackupDatabaseSnapshot{
		TableRows:      make(map[string]int64, len(backupCriticalTables)),
		UnsettledTasks: make(map[string]int64),
	}
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&snapshot.DatabaseVersion); err != nil {
		return nil, fmt.Errorf("query PostgreSQL version: %w", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT pg_database_size(current_database())").Scan(&snapshot.DatabaseSizeBytes); err != nil {
		return nil, fmt.Errorf("query database size: %w", err)
	}
	current, err := currentMigrationVersion(ctx, db)
	if err != nil {
		return nil, err
	}
	snapshot.CurrentMigration = current
	for _, table := range backupCriticalTables {
		count, err := tableRowCount(ctx, db, table)
		if err != nil {
			return nil, err
		}
		snapshot.TableRows[table] = count
	}
	var unsettledBatchImages int64
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM batch_image_jobs
		WHERE settled_at IS NULL
		  AND status NOT IN ('failed', 'cancelled', 'canceled', 'expired')
	`).Scan(&unsettledBatchImages); err != nil {
		return nil, fmt.Errorf("query unsettled batch image jobs: %w", err)
	}
	snapshot.UnsettledTasks["batch_image_jobs"] = unsettledBatchImages
	var unsettledPaymentOrders int64
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM payment_orders
		WHERE status IN ('PENDING', 'PROCESSING')
	`).Scan(&unsettledPaymentOrders); err != nil {
		return nil, fmt.Errorf("query unsettled payment orders: %w", err)
	}
	snapshot.UnsettledTasks["payment_orders"] = unsettledPaymentOrders
	return snapshot, nil
}

func currentMigrationVersion(ctx context.Context, db *sql.DB) (string, error) {
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return "", fmt.Errorf("check schema_migrations: %w", err)
	}
	if !exists {
		return "", nil
	}
	var version string
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(filename), '') FROM schema_migrations").Scan(&version); err != nil {
		return "", fmt.Errorf("query current migration: %w", err)
	}
	return version, nil
}

func latestEmbeddedMigration() (string, error) {
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return "", fmt.Errorf("list embedded migrations: %w", err)
	}
	if len(files) == 0 {
		return "", errors.New("no embedded migrations found")
	}
	sort.Strings(files)
	return files[len(files)-1], nil
}

func tableRowCount(ctx context.Context, db *sql.DB, table string) (int64, error) {
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", "public."+table).Scan(&exists); err != nil {
		return 0, fmt.Errorf("check table %s: %w", table, err)
	}
	if !exists {
		return 0, fmt.Errorf("required backup table %s is missing", table)
	}
	var count int64
	query := "SELECT COUNT(*) FROM " + pq.QuoteIdentifier(table)
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count table %s: %w", table, err)
	}
	return count, nil
}

// VerifyRestore 在同一 PostgreSQL 集群创建随机临时库，恢复完成后执行关键表、
// 迁移版本和加密字段 smoke test。临时库无论成功失败都会被终止连接并删除。
func (d *PgDumper) VerifyRestore(
	ctx context.Context,
	data io.Reader,
	expected *service.BackupManifest,
) (_ *service.BackupRecoveryReport, retErr error) {
	if d == nil || d.db == nil || d.cfg == nil {
		return nil, errors.New("verify backup restore: database is not configured")
	}
	tempDB := "sub2api_verify_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	quotedDB := pq.QuoteIdentifier(tempDB)
	if _, err := d.db.ExecContext(ctx, "CREATE DATABASE "+quotedDB+" TEMPLATE template0"); err != nil {
		return nil, fmt.Errorf("create temporary verification database: %w", err)
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cleanupErr := d.dropVerificationDatabase(cleanupCtx, tempDB)
		if cleanupErr != nil {
			retErr = errors.Join(retErr, cleanupErr)
		}
	}()

	if err := d.restoreToDatabase(ctx, data, tempDB); err != nil {
		return nil, fmt.Errorf("restore temporary verification database: %w", err)
	}
	verifyCfg := *d.cfg
	verifyCfg.DBName = tempDB
	verifyDB, err := sql.Open("postgres", verifyCfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open temporary verification database: %w", err)
	}
	defer func() { _ = verifyDB.Close() }()
	if err := verifyDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping temporary verification database: %w", err)
	}
	snapshot, err := inspectBackupDatabase(ctx, verifyDB)
	if err != nil {
		return nil, fmt.Errorf("inspect restored database: %w", err)
	}
	report := &service.BackupRecoveryReport{
		DatabaseVersion:  snapshot.DatabaseVersion,
		MigrationVersion: snapshot.CurrentMigration,
		TableRows:        snapshot.TableRows,
	}
	add := func(name, status, detail string) {
		report.Checks = append(report.Checks, service.BackupRecoveryCheck{Name: name, Status: status, Detail: detail})
	}
	add("critical_tables", "pass", fmt.Sprintf("%d 张关键表可读", len(backupCriticalTables)))
	if expected != nil && expected.CurrentMigration != "" && snapshot.CurrentMigration != expected.CurrentMigration {
		return nil, fmt.Errorf("restored migration mismatch: got %s, want %s", snapshot.CurrentMigration, expected.CurrentMigration)
	}
	add("migration", "pass", snapshot.CurrentMigration)

	var invalidCredentials int64
	if err := verifyDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM accounts
		WHERE credentials IS NULL OR jsonb_typeof(credentials) IS NULL
	`).Scan(&invalidCredentials); err != nil {
		return nil, fmt.Errorf("smoke test account credentials: %w", err)
	}
	if invalidCredentials > 0 {
		return nil, fmt.Errorf("smoke test found %d invalid account credential rows", invalidCredentials)
	}
	add("account_credentials", "pass", "账号凭证字段可读取且 JSON 结构有效")

	var emptySecrets int64
	if err := verifyDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM security_secrets WHERE key = '' OR value = ''").Scan(&emptySecrets); err != nil {
		return nil, fmt.Errorf("smoke test security secrets: %w", err)
	}
	if emptySecrets > 0 {
		return nil, fmt.Errorf("smoke test found %d empty security secret rows", emptySecrets)
	}
	add("security_secrets", "pass", "安全密钥记录结构正常")

	if expected != nil && len(expected.TableRows) > 0 {
		var differences []string
		for table, expectedCount := range expected.TableRows {
			if actual, ok := snapshot.TableRows[table]; ok && actual != expectedCount {
				differences = append(differences, fmt.Sprintf("%s:%d/%d", table, actual, expectedCount))
			}
		}
		if len(differences) > 0 {
			sort.Strings(differences)
			add("row_summary", "warn", "源库快照与恢复行数存在并发写入差异: "+strings.Join(differences, ", "))
		} else {
			add("row_summary", "pass", "关键表行数与备份清单一致")
		}
	}
	return report, nil
}

func (d *PgDumper) dropVerificationDatabase(ctx context.Context, name string) error {
	if _, err := d.db.ExecContext(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`, name); err != nil {
		return fmt.Errorf("terminate temporary verification connections: %w", err)
	}
	if _, err := d.db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+pq.QuoteIdentifier(name)); err != nil {
		return fmt.Errorf("drop temporary verification database: %w", err)
	}
	return nil
}
