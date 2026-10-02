package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type settingRepository struct {
	client *ent.Client
}

func NewSettingRepository(client *ent.Client) service.SettingRepository {
	return &settingRepository{client: client}
}

func (r *settingRepository) Get(ctx context.Context, key string) (*service.Setting, error) {
	m, err := r.client.Setting.Query().Where(setting.KeyEQ(key)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, service.ErrSettingNotFound
		}
		return nil, err
	}
	return &service.Setting{
		ID:        m.ID,
		Key:       m.Key,
		Value:     m.Value,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *settingRepository) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (r *settingRepository) Set(ctx context.Context, key, value string) error {
	now := time.Now()
	return r.client.Setting.
		Create().
		SetKey(key).
		SetValue(value).
		SetUpdatedAt(now).
		OnConflictColumns(setting.FieldKey).
		UpdateNewValues().
		Exec(ctx)
}

func (r *settingRepository) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	settings, err := r.client.Setting.Query().Where(setting.KeyIn(keys...)).All(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

func (r *settingRepository) SetMultiple(ctx context.Context, settings map[string]string) error {
	if len(settings) == 0 {
		return nil
	}

	now := time.Now()
	builders := make([]*ent.SettingCreate, 0, len(settings))
	for key, value := range settings {
		builders = append(builders, r.client.Setting.Create().SetKey(key).SetValue(value).SetUpdatedAt(now))
	}
	return r.client.Setting.
		CreateBulk(builders...).
		OnConflictColumns(setting.FieldKey).
		UpdateNewValues().
		Exec(ctx)
}

// SetMultipleWithVersion 在同一事务内写入设置和版本元数据。
// 版本行会先加行锁再比较，避免多个实例上的管理员页面互相覆盖配置。
func (r *settingRepository) SetMultipleWithVersion(ctx context.Context, settings map[string]string, expectedVersion, nextVersion, versionMetadata string) (bool, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return false, err
	}
	rollback := func(cause error) (bool, error) {
		_ = tx.Rollback()
		return false, cause
	}

	const versionKey = "__sub2api_runtime_settings_version"
	versionRow, err := tx.Setting.Query().Where(setting.KeyEQ(versionKey)).ForUpdate().Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return rollback(err)
	}
	// 拿到版本行锁后重新计算真实内容指纹，不能只相信元数据行；
	// 这样手工改库或旧版本程序写入设置后，CAS 仍能识别最新内容。
	rows, err := tx.Setting.Query().All(ctx)
	if err != nil {
		return rollback(err)
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	current := service.ComputeSettingsVersion(values)
	if expectedVersion != "" && expectedVersion != current {
		return rollback(nil)
	}

	now := time.Now()
	for key, value := range settings {
		if key == versionKey {
			continue
		}
		if err := tx.Setting.Create().SetKey(key).SetValue(value).SetUpdatedAt(now).
			OnConflictColumns(setting.FieldKey).UpdateNewValues().Exec(ctx); err != nil {
			return rollback(err)
		}
	}
	if versionRow == nil {
		if err := tx.Setting.Create().SetKey(versionKey).SetValue(versionMetadata).SetUpdatedAt(now).Exec(ctx); err != nil {
			return rollback(err)
		}
	} else {
		if err := tx.Setting.UpdateOneID(versionRow.ID).SetValue(versionMetadata).SetUpdatedAt(now).Exec(ctx); err != nil {
			return rollback(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *settingRepository) GetAll(ctx context.Context) (map[string]string, error) {
	settings, err := r.client.Setting.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

func (r *settingRepository) Delete(ctx context.Context, key string) error {
	_, err := r.client.Setting.Delete().Where(setting.KeyEQ(key)).Exec(ctx)
	return err
}
