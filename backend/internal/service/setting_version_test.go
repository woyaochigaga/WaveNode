package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type versionedSettingRepo struct {
	values map[string]string
}

func (r *versionedSettingRepo) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (r *versionedSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}
func (r *versionedSettingRepo) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}
func (r *versionedSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		values[key] = r.values[key]
	}
	return values, nil
}
func (r *versionedSettingRepo) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		r.values[key] = value
	}
	return nil
}
func (r *versionedSettingRepo) SetMultipleWithVersion(_ context.Context, settings map[string]string, expectedVersion, _ string, versionMetadata string) (bool, error) {
	// 测试桩按真实仓储语义先比较内容版本，再一次性写入设置和版本元数据。
	if ComputeSettingsVersion(r.values) != expectedVersion {
		return false, nil
	}
	for key, value := range settings {
		r.values[key] = value
	}
	r.values[runtimeSettingsVersionKey] = versionMetadata
	return true, nil
}
func (r *versionedSettingRepo) GetAll(context.Context) (map[string]string, error) {
	values := make(map[string]string, len(r.values))
	for key, value := range r.values {
		values[key] = value
	}
	return values, nil
}
func (r *versionedSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestComputeSettingsVersionIsStableAndIgnoresMetadataRow(t *testing.T) {
	first := ComputeSettingsVersion(map[string]string{"b": "2", "a": "1"})
	second := ComputeSettingsVersion(map[string]string{
		"a": "1", "b": "2", runtimeSettingsVersionKey: `{"version":"old"}`,
	})

	require.Equal(t, first, second)
	require.Len(t, first, 16)
}

func TestPersistSettingsVersionedRejectsStalePageVersion(t *testing.T) {
	repo := &versionedSettingRepo{values: map[string]string{"feature": "new"}}
	service := &SettingService{settingRepo: repo}

	err := service.persistSettingsVersioned(context.Background(), map[string]string{"feature": "mine"}, ComputeSettingsVersion(map[string]string{"feature": "old"}), nil)

	require.Error(t, err)
	require.Equal(t, "RUNTIME_SETTINGS_VERSION_CONFLICT", infraerrors.Reason(err))
	require.Equal(t, "new", repo.values["feature"])
}

func TestPersistSettingsVersionedRecordsActorAndEffectiveTime(t *testing.T) {
	repo := &versionedSettingRepo{values: map[string]string{"feature": "old"}}
	settingService := &SettingService{settingRepo: repo}
	actorID := int64(42)

	err := settingService.persistSettingsVersioned(
		context.Background(),
		map[string]string{"feature": "new"},
		ComputeSettingsVersion(repo.values),
		&actorID,
	)

	require.NoError(t, err)
	var record runtimeSettingsVersionRecord
	require.NoError(t, json.Unmarshal([]byte(repo.values[runtimeSettingsVersionKey]), &record))
	require.Equal(t, ComputeSettingsVersion(repo.values), record.Version)
	require.Equal(t, actorID, *record.UpdatedBy)
	require.NotNil(t, record.EffectiveAt)
	require.Equal(t, record.UpdatedAt, *record.EffectiveAt)

	observed := settingService.RuntimeSettingsVersion(context.Background())
	require.Equal(t, record.Version, observed.Version)
	require.Equal(t, record.UpdatedAt, observed.UpdatedAt)
	require.Equal(t, record.UpdatedBy, observed.UpdatedBy)
	require.Equal(t, record.EffectiveAt, observed.EffectiveAt)
}

func TestReconcileRuntimeSettingsVersionAlignsSupplementalWrites(t *testing.T) {
	oldVersion := ComputeSettingsVersion(map[string]string{"feature": "old"})
	repo := &versionedSettingRepo{values: map[string]string{
		"feature":                 "new",
		runtimeSettingsVersionKey: runtimeSettingsVersionValue(oldVersion, time.Now(), nil),
	}}
	settingService := &SettingService{settingRepo: repo}
	actorID := int64(7)

	err := settingService.ReconcileRuntimeSettingsVersion(context.Background(), &actorID)

	require.NoError(t, err)
	var record runtimeSettingsVersionRecord
	require.NoError(t, json.Unmarshal([]byte(repo.values[runtimeSettingsVersionKey]), &record))
	require.Equal(t, ComputeSettingsVersion(repo.values), record.Version)
	require.Equal(t, actorID, *record.UpdatedBy)
}
