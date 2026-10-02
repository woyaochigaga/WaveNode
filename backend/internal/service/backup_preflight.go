package service

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
	backupPreflightPass  = "pass"
	backupPreflightWarn  = "warn"
	backupPreflightFail  = "fail"
	backupFreshnessLimit = 24 * time.Hour
)

// BackupDatabaseSnapshot 是备份与迁移 preflight 共用的只读数据库摘要。
type BackupDatabaseSnapshot struct {
	DatabaseVersion   string           `json:"database_version"`
	CurrentMigration  string           `json:"current_migration"`
	TargetMigration   string           `json:"target_migration"`
	DatabaseSizeBytes int64            `json:"database_size_bytes"`
	TableRows         map[string]int64 `json:"table_rows"`
	UnsettledTasks    map[string]int64 `json:"unsettled_tasks"`
	PGDumpAvailable   bool             `json:"pg_dump_available"`
	PSQLAvailable     bool             `json:"psql_available"`
}

// BackupManifest 固化备份生成时的环境与完整性证据。
type BackupManifest struct {
	SHA256            string           `json:"sha256"`
	SizeBytes         int64            `json:"size_bytes"`
	DatabaseVersion   string           `json:"database_version,omitempty"`
	CurrentMigration  string           `json:"current_migration,omitempty"`
	TargetMigration   string           `json:"target_migration,omitempty"`
	DatabaseSizeBytes int64            `json:"database_size_bytes,omitempty"`
	TableRows         map[string]int64 `json:"table_rows,omitempty"`
	CapturedAt        string           `json:"captured_at"`
}

type BackupRecoveryCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// BackupRecoveryReport 是临时数据库恢复后生成的可审计证据。
type BackupRecoveryReport struct {
	DatabaseVersion  string                `json:"database_version"`
	MigrationVersion string                `json:"migration_version"`
	TableRows        map[string]int64      `json:"table_rows"`
	Checks           []BackupRecoveryCheck `json:"checks"`
}

type BackupPreflightCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Blocking bool   `json:"blocking"`
	Detail   string `json:"detail"`
}

type BackupPreflightReport struct {
	OperationID       string                 `json:"operation_id"`
	Ready             bool                   `json:"ready"`
	CheckedAt         string                 `json:"checked_at"`
	CurrentMigration  string                 `json:"current_migration,omitempty"`
	TargetMigration   string                 `json:"target_migration,omitempty"`
	DatabaseVersion   string                 `json:"database_version,omitempty"`
	DatabaseSizeBytes int64                  `json:"database_size_bytes,omitempty"`
	DiskFreeBytes     int64                  `json:"disk_free_bytes,omitempty"`
	LatestBackupAt    string                 `json:"latest_backup_at,omitempty"`
	LatestBackupAge   int64                  `json:"latest_backup_age_seconds,omitempty"`
	UnsettledTasks    map[string]int64       `json:"unsettled_tasks,omitempty"`
	Checks            []BackupPreflightCheck `json:"checks"`
}

func (s *BackupService) captureBackupManifest(ctx context.Context) (*BackupManifest, error) {
	manifest := &BackupManifest{CapturedAt: time.Now().UTC().Format(time.RFC3339)}
	inspector, ok := s.dumper.(BackupDatabaseInspector)
	if !ok {
		// 自定义/测试 dumper 没有数据库检查能力时仍生成完整性 manifest，
		// 但不会伪造数据库版本和行数证据。
		return manifest, nil
	}
	snapshot, err := inspector.InspectDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("capture backup database manifest: %w", err)
	}
	manifest.DatabaseVersion = snapshot.DatabaseVersion
	manifest.CurrentMigration = snapshot.CurrentMigration
	manifest.TargetMigration = snapshot.TargetMigration
	manifest.DatabaseSizeBytes = snapshot.DatabaseSizeBytes
	manifest.TableRows = snapshot.TableRows
	return manifest, nil
}

// RunPreflight 汇总迁移前需要人工或自动确认的只读检查，不修改数据库或 Redis。
func (s *BackupService) RunPreflight(ctx context.Context) (*BackupPreflightReport, error) {
	report := &BackupPreflightReport{
		OperationID:    uuid.NewString(),
		Ready:          true,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		UnsettledTasks: map[string]int64{},
	}
	add := func(name, status, detail string, blocking bool) {
		report.Checks = append(report.Checks, BackupPreflightCheck{Name: name, Status: status, Detail: detail, Blocking: blocking})
		if blocking && status == backupPreflightFail {
			report.Ready = false
		}
	}

	inspector, ok := s.dumper.(BackupDatabaseInspector)
	if !ok {
		add("database", backupPreflightFail, "当前数据库驱动不支持 preflight", true)
	} else if snapshot, err := inspector.InspectDatabase(ctx); err != nil {
		add("database", backupPreflightFail, err.Error(), true)
	} else {
		report.DatabaseVersion = snapshot.DatabaseVersion
		report.CurrentMigration = snapshot.CurrentMigration
		report.TargetMigration = snapshot.TargetMigration
		report.DatabaseSizeBytes = snapshot.DatabaseSizeBytes
		report.UnsettledTasks = snapshot.UnsettledTasks
		add("database", backupPreflightPass, "数据库连接和只读查询正常", true)
		migrationDetail := fmt.Sprintf("当前 %s，目标 %s", snapshot.CurrentMigration, snapshot.TargetMigration)
		add("migration", backupPreflightPass, migrationDetail, true)
		if !snapshot.PGDumpAvailable || !snapshot.PSQLAvailable {
			add("postgres_tools", backupPreflightFail, "pg_dump 或 psql 不可用", true)
		} else {
			add("postgres_tools", backupPreflightPass, "pg_dump 与 psql 可用", true)
		}
		var unsettled int64
		for _, count := range snapshot.UnsettledTasks {
			unsettled += count
		}
		if unsettled > 0 {
			add("unsettled_tasks", backupPreflightWarn, fmt.Sprintf("仍有 %d 个未收口任务", unsettled), false)
		} else {
			add("unsettled_tasks", backupPreflightPass, "没有未收口任务", false)
		}
	}

	if !s.encryptionKeyConfigured {
		add("encryption_key", backupPreflightFail, "未配置可跨重启复用的固定加密密钥", true)
	} else {
		add("encryption_key", backupPreflightPass, "固定加密密钥已配置", true)
	}

	if checker, ok := s.lockCache.(BackupCacheHealthChecker); !ok {
		add("redis", backupPreflightFail, "Redis 健康检查不可用", true)
	} else if err := checker.BackupHealthCheck(ctx); err != nil {
		add("redis", backupPreflightFail, err.Error(), true)
	} else {
		add("redis", backupPreflightPass, "Redis 连接正常", true)
	}

	if free, err := backupDiskFreeBytes(); err != nil {
		add("disk", backupPreflightFail, err.Error(), true)
	} else {
		report.DiskFreeBytes = free
		required := report.DatabaseSizeBytes * 2
		if required < 1024*1024*1024 {
			required = 1024 * 1024 * 1024
		}
		if free < required {
			add("disk", backupPreflightFail, fmt.Sprintf("临时目录可用 %d 字节，至少需要 %d 字节", free, required), true)
		} else {
			add("disk", backupPreflightPass, fmt.Sprintf("临时目录可用 %d 字节", free), true)
		}
	}

	records, err := s.ListBackups(ctx)
	if err != nil {
		add("backup_age", backupPreflightFail, err.Error(), true)
		return report, nil
	}
	if latest, ok := latestVerifiedBackup(records); ok {
		report.LatestBackupAt = latest.FinishedAt
		finishedAt, _ := time.Parse(time.RFC3339, latest.FinishedAt)
		age := time.Since(finishedAt)
		report.LatestBackupAge = int64(age.Seconds())
		if age > backupFreshnessLimit {
			add("backup_age", backupPreflightWarn, fmt.Sprintf("最近已演练备份距今 %s", age.Round(time.Minute)), false)
		} else {
			add("backup_age", backupPreflightPass, fmt.Sprintf("最近已演练备份距今 %s", age.Round(time.Minute)), false)
		}
	} else {
		add("backup_age", backupPreflightWarn, "没有通过恢复演练的备份", false)
	}
	return report, nil
}

func latestVerifiedBackup(records []BackupRecord) (BackupRecord, bool) {
	var latest BackupRecord
	found := false
	for _, record := range records {
		if record.Status != "completed" || record.VerificationStatus != "passed" || record.FinishedAt == "" {
			continue
		}
		if !found || record.FinishedAt > latest.FinishedAt {
			latest = record
			found = true
		}
	}
	return latest, found
}

func backupDiskFreeBytes() (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(os.TempDir(), &stat); err != nil {
		return 0, fmt.Errorf("读取临时目录磁盘空间失败: %w", err)
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
