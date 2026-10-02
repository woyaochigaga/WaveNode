package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	// runtimeSettingsVersionKey 是内部元数据行，不参与内容哈希，避免版本号递归计算自身。
	runtimeSettingsVersionKey = "__sub2api_runtime_settings_version"
	runtimeSettingsVersionTTL = 30 * time.Second
)

type RuntimeSettingsVersion struct {
	Version     string     `json:"version"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UpdatedBy   *int64     `json:"updated_by,omitempty"`
	EffectiveAt *time.Time `json:"effective_at,omitempty"`
}

type runtimeSettingsVersionSnapshot struct {
	RuntimeSettingsVersion
	LoadedAt time.Time
}

type runtimeSettingsVersionRecord struct {
	Version     string     `json:"version"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UpdatedBy   *int64     `json:"updated_by,omitempty"`
	EffectiveAt *time.Time `json:"effective_at,omitempty"`
}

// SettingVersionWriter 是可选扩展，既兼容现有测试桩，也让 PostgreSQL 仓储提供原子 CAS。
type SettingVersionWriter interface {
	SetMultipleWithVersion(ctx context.Context, settings map[string]string, expectedVersion, nextVersion, versionMetadata string) (bool, error)
}

var ErrRuntimeSettingsVersionConflict = infraerrors.Conflict(
	"RUNTIME_SETTINGS_VERSION_CONFLICT",
	"settings changed after this page was loaded; refresh and try again",
)

// ComputeSettingsVersion 为数据库设置生成稳定内容指纹。
// 密钥会参与哈希以感知变化，但任何返回值都不包含密钥原文。
func ComputeSettingsVersion(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != runtimeSettingsVersionKey {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, key := range keys {
		_, _ = h.Write([]byte(key))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(values[key]))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// RuntimeSettingsVersion 返回当前观察到的设置版本，并用短缓存避免每个网关请求都查数据库。
func (s *SettingService) RuntimeSettingsVersion(ctx context.Context) RuntimeSettingsVersion {
	if s == nil || s.settingRepo == nil {
		return RuntimeSettingsVersion{Version: "unavailable"}
	}
	if cached, _ := s.settingsVersionCache.Load().(*runtimeSettingsVersionSnapshot); cached != nil && time.Since(cached.LoadedAt) < runtimeSettingsVersionTTL {
		return cached.RuntimeSettingsVersion
	}
	result, _, _ := s.settingsVersionSF.Do("runtime-settings-version", func() (any, error) {
		values, err := s.settingRepo.GetAll(ctx)
		if err != nil {
			return nil, err
		}
		version := ComputeSettingsVersion(values)
		info := RuntimeSettingsVersion{Version: version, UpdatedAt: time.Now()}
		if raw := strings.TrimSpace(values[runtimeSettingsVersionKey]); raw != "" {
			var record runtimeSettingsVersionRecord
			if json.Unmarshal([]byte(raw), &record) == nil && record.Version == version {
				info.UpdatedAt = record.UpdatedAt
				info.UpdatedBy = record.UpdatedBy
				info.EffectiveAt = record.EffectiveAt
			}
		}
		snapshot := &runtimeSettingsVersionSnapshot{RuntimeSettingsVersion: info, LoadedAt: time.Now()}
		s.settingsVersionCache.Store(snapshot)
		return info, nil
	})
	if info, ok := result.(RuntimeSettingsVersion); ok {
		return info
	}
	if cached, _ := s.settingsVersionCache.Load().(*runtimeSettingsVersionSnapshot); cached != nil {
		return cached.RuntimeSettingsVersion
	}
	return RuntimeSettingsVersion{Version: "unavailable"}
}

func (s *SettingService) invalidateRuntimeSettingsVersion() {
	if s == nil {
		return
	}
	if cached, _ := s.settingsVersionCache.Load().(*runtimeSettingsVersionSnapshot); cached != nil {
		expired := *cached
		expired.LoadedAt = time.Time{}
		s.settingsVersionCache.Store(&expired)
	}
	s.settingsVersionSF.Forget("runtime-settings-version")
}

func runtimeSettingsVersionValue(version string, updatedAt time.Time, updatedBy *int64) string {
	// 当前设置写入后立即生效；保留独立字段便于后续支持定时发布。
	effectiveAt := updatedAt
	record := runtimeSettingsVersionRecord{
		Version: version, UpdatedAt: updatedAt, UpdatedBy: updatedBy, EffectiveAt: &effectiveAt,
	}
	data, _ := json.Marshal(record)
	return string(data)
}

func (s *SettingService) persistSettingsVersioned(ctx context.Context, updates map[string]string, expectedVersion string, updatedBy *int64) error {
	values, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	// 内容哈希才是并发比较依据；元数据行只负责记录操作者和生效时间，
	// 避免旧版本程序或手工改库后出现无法继续保存的版本漂移。
	current := ComputeSettingsVersion(values)
	if expectedVersion != "" && expectedVersion != current {
		return ErrRuntimeSettingsVersionConflict.WithMetadata(map[string]string{
			"expected_version": expectedVersion,
			"current_version":  current,
		})
	}
	merged := make(map[string]string, len(values)+len(updates))
	for key, value := range values {
		if key != runtimeSettingsVersionKey {
			merged[key] = value
		}
	}
	for key, value := range updates {
		merged[key] = value
	}
	nextVersion := ComputeSettingsVersion(merged)
	metadata := runtimeSettingsVersionValue(nextVersion, time.Now(), updatedBy)
	if writer, ok := s.settingRepo.(SettingVersionWriter); ok {
		committed, err := writer.SetMultipleWithVersion(ctx, updates, current, nextVersion, metadata)
		if err != nil {
			return err
		}
		if !committed {
			return ErrRuntimeSettingsVersionConflict.WithMetadata(map[string]string{
				"expected_version": expectedVersion,
				"current_version":  current,
			})
		}
	} else {
		// 兼容轻量测试桩和旧存储；仍提供版本化响应，但仓储升级前不具备跨进程原子 CAS。
		if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
			return err
		}
	}
	s.invalidateRuntimeSettingsVersion()
	return nil
}

// ReconcileRuntimeSettingsVersion 在同一次管理端保存包含独立设置写入时，
// 把版本元数据对齐到最终内容；已有正确元数据时不会重复改写时间和操作者。
func (s *SettingService) ReconcileRuntimeSettingsVersion(ctx context.Context, updatedBy *int64) error {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	values, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	current := ComputeSettingsVersion(values)
	if raw := strings.TrimSpace(values[runtimeSettingsVersionKey]); raw != "" {
		var record runtimeSettingsVersionRecord
		if json.Unmarshal([]byte(raw), &record) == nil && record.Version == current {
			return nil
		}
	}

	metadata := runtimeSettingsVersionValue(current, time.Now(), updatedBy)
	if writer, ok := s.settingRepo.(SettingVersionWriter); ok {
		committed, err := writer.SetMultipleWithVersion(ctx, map[string]string{}, current, current, metadata)
		if err != nil {
			return err
		}
		if !committed {
			return ErrRuntimeSettingsVersionConflict.WithMetadata(map[string]string{"current_version": current})
		}
	} else if err := s.settingRepo.Set(ctx, runtimeSettingsVersionKey, metadata); err != nil {
		return err
	}
	s.invalidateRuntimeSettingsVersion()
	return nil
}
