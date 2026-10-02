package service

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *BackupService) requiresRecoveryVerification() bool {
	_, ok := s.dumper.(BackupRecoveryVerifier)
	return ok
}

// StartBackupVerification 异步执行临时数据库恢复演练，和正式恢复共用串行锁，
// 防止同一个 PostgreSQL 集群同时承受两次大体量恢复。
func (s *BackupService) StartBackupVerification(ctx context.Context, backupID string) (*BackupRecord, error) {
	if s.shuttingDown.Load() {
		return nil, infraerrors.ServiceUnavailable("SERVER_SHUTTING_DOWN", "server is shutting down")
	}
	verifier, ok := s.dumper.(BackupRecoveryVerifier)
	if !ok {
		return nil, infraerrors.ServiceUnavailable("BACKUP_VERIFICATION_UNAVAILABLE", "database driver does not support recovery verification")
	}

	s.opMu.Lock()
	if s.restoring {
		s.opMu.Unlock()
		return nil, ErrRestoreInProgress
	}
	s.restoring = true
	s.opMu.Unlock()
	launched := false
	defer func() {
		if !launched {
			s.opMu.Lock()
			s.restoring = false
			s.opMu.Unlock()
		}
	}()

	s3Cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return nil, err
	}
	objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
	if err != nil {
		return nil, fmt.Errorf("init object store: %w", err)
	}
	record, err := s.beginBackupVerification(ctx, backupID)
	if err != nil {
		return nil, err
	}
	launched = true
	result := *record
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.opMu.Lock()
			s.restoring = false
			s.opMu.Unlock()
		}()
		defer func() {
			if recovered := recover(); recovered != nil {
				record.VerificationStatus = "failed"
				record.VerificationError = fmt.Sprintf("internal panic: %v", recovered)
				_ = s.saveVerificationRecord(context.Background(), record)
			}
		}()
		s.executeBackupVerification(record, objectStore, verifier)
	}()
	return &result, nil
}

func (s *BackupService) executeBackupVerification(
	record *BackupRecord,
	objectStore BackupObjectStore,
	verifier BackupRecoveryVerifier,
) {
	ctx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)
	defer cancel()
	archivePath, err := s.downloadBackupArchive(ctx, objectStore, record)
	if err == nil {
		defer func() { _ = cleanupBackupFiles(archivePath) }()
		archive, openErr := os.Open(archivePath)
		if openErr != nil {
			err = fmt.Errorf("open verification archive: %w", openErr)
		} else {
			defer func() { _ = archive.Close() }()
			gzReader, gzipErr := gzip.NewReader(archive)
			if gzipErr != nil {
				err = fmt.Errorf("open verification gzip: %w", gzipErr)
			} else {
				report, verifyErr := verifier.VerifyRestore(ctx, gzReader, record.Manifest)
				closeErr := gzReader.Close()
				err = errors.Join(verifyErr, closeErr)
				if err == nil {
					record.VerificationReport = report
				}
			}
		}
	}
	if err != nil {
		record.VerificationStatus = "failed"
		record.VerificationError = err.Error()
		logger.LegacyPrintf("service.backup", "[Backup] 恢复演练失败: operation_id=%s backup_id=%s err=%v", record.VerificationOperationID, record.ID, err)
	} else {
		record.VerificationStatus = "passed"
		record.VerificationError = ""
		record.VerifiedAt = time.Now().UTC().Format(time.RFC3339)
		logger.LegacyPrintf("service.backup", "[Backup] 恢复演练通过: operation_id=%s backup_id=%s", record.VerificationOperationID, record.ID)
	}
	if saveErr := s.saveVerificationRecord(context.Background(), record); saveErr != nil {
		logger.LegacyPrintf("service.backup", "[Backup] 保存恢复演练结果失败: operation_id=%s err=%v", record.VerificationOperationID, saveErr)
	}
}

func (s *BackupService) downloadBackupArchive(
	ctx context.Context,
	objectStore BackupObjectStore,
	record *BackupRecord,
) (string, error) {
	if record == nil {
		return "", ErrBackupNotFound
	}
	var archivePath string
	var err error
	if len(record.Parts) > 0 {
		archivePath, err = s.downloadBackupParts(ctx, objectStore, record.Parts)
	} else {
		archivePath, err = downloadSingleBackup(ctx, objectStore, record.S3Key)
	}
	if err != nil {
		return "", err
	}
	if err := verifyBackupArchive(archivePath, record); err != nil {
		_ = cleanupBackupFiles(archivePath)
		return "", err
	}
	return archivePath, nil
}

func downloadSingleBackup(ctx context.Context, objectStore BackupObjectStore, key string) (path string, err error) {
	if strings.TrimSpace(key) == "" {
		return "", errors.New("backup object key is empty")
	}
	body, err := objectStore.Download(ctx, key)
	if err != nil {
		return "", fmt.Errorf("S3 download failed: %w", err)
	}
	defer func() { _ = body.Close() }()
	archive, err := os.CreateTemp("", "sub2api-restore-*.sql.gz")
	if err != nil {
		return "", fmt.Errorf("create restore archive: %w", err)
	}
	path = archive.Name()
	if _, err := io.Copy(archive, body); err != nil {
		_ = archive.Close()
		_ = cleanupBackupFiles(path)
		return "", fmt.Errorf("download backup archive: %w", err)
	}
	if err := archive.Close(); err != nil {
		_ = cleanupBackupFiles(path)
		return "", fmt.Errorf("close restore archive: %w", err)
	}
	return path, nil
}

func verifyBackupArchive(path string, record *BackupRecord) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat restore archive: %w", err)
	}
	if record.SizeBytes > 0 && info.Size() != record.SizeBytes {
		return fmt.Errorf("backup archive size mismatch: got %d, want %d", info.Size(), record.SizeBytes)
	}
	if record.Manifest != nil && record.Manifest.SHA256 != "" {
		checksum, err := backupFileSHA256(path)
		if err != nil {
			return err
		}
		if !strings.EqualFold(checksum, record.Manifest.SHA256) {
			return errors.New("backup archive checksum mismatch")
		}
	}
	archive, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open restore archive: %w", err)
	}
	defer func() { _ = archive.Close() }()
	gzReader, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	_, copyErr := io.Copy(io.Discard, gzReader)
	closeErr := gzReader.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return fmt.Errorf("validate backup gzip: %w", err)
	}
	return nil
}
