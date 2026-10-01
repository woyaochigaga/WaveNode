//go:build unit

package service

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type groupTestAccountRepo struct {
	accountRepoStub
	accounts []Account
}

func (r *groupTestAccountRepo) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return r.accounts, nil
}

func (r *groupTestAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := r.accounts[i]
			return &account, nil
		}
	}
	return nil, ErrAccountNotFound
}

func TestAccountTestService_GetGroupTestModelsKeepsCompositePlatformsIsolated(t *testing.T) {
	repo := &groupTestAccountRepo{accounts: []Account{
		{
			ID:          1,
			Platform:    PlatformAnthropic,
			Status:      StatusActive,
			Schedulable: true,
		},
	}}
	service := &AccountTestService{accountRepo: repo}

	models, err := service.GetGroupTestModels(context.Background(), &Group{ID: 8, Platform: PlatformComposite})

	require.NoError(t, err)
	require.Contains(t, models, "claude-sonnet-4-6")
	require.NotContains(t, models, "gpt-5.5")
}

func TestAccountTestService_TestGroupConnectionUsesSchedulableMatchingAccount(t *testing.T) {
	repo := &groupTestAccountRepo{accounts: []Account{
		{
			ID:          7,
			Platform:    PlatformAnthropic,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Extra:       map[string]any{"synthetic_ui_test": true},
			Credentials: map[string]any{
				"model_mapping": map[string]any{"claude-sonnet-4-6": "claude-sonnet-4-6"},
			},
		},
	}}
	service := &AccountTestService{accountRepo: repo}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/test", strings.NewReader(`{}`))
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	err := service.TestGroupConnection(ctx, &Group{ID: 8, Platform: PlatformAnthropic}, "claude-sonnet-4-6", "", "")

	require.NoError(t, err)
	require.Contains(t, recorder.Body.String(), `"status":"route_selected"`)
	require.Contains(t, recorder.Body.String(), `"type":"test_complete"`)
	require.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestAccountTestService_GetGroupTestModelsHonorsModelAllowlist(t *testing.T) {
	repo := &groupTestAccountRepo{accounts: []Account{
		{
			ID:          2,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
		},
	}}
	service := &AccountTestService{accountRepo: repo}
	group := &Group{
		ID:       9,
		Platform: PlatformOpenAI,
		ModelAllowlist: GroupModelAllowlist{
			Enabled: true,
			Models:  []string{"gpt-5-mini"},
		},
	}

	models, err := service.GetGroupTestModels(context.Background(), group)

	require.NoError(t, err)
	require.Equal(t, []string{"gpt-5-mini"}, models)
}

func TestAccountTestService_GetGroupTestModelsIncludesMixedAntigravityAccount(t *testing.T) {
	repo := &groupTestAccountRepo{accounts: []Account{
		{
			ID:          3,
			Platform:    PlatformAntigravity,
			Status:      StatusActive,
			Schedulable: true,
			Extra:       map[string]any{"mixed_scheduling": true},
			Credentials: map[string]any{
				"model_mapping": map[string]any{"claude-sonnet-4-6": "claude-sonnet-4-6"},
			},
		},
	}}
	service := &AccountTestService{accountRepo: repo}

	models, err := service.GetGroupTestModels(context.Background(), &Group{ID: 10, Platform: PlatformAnthropic})

	require.NoError(t, err)
	require.Contains(t, models, "claude-sonnet-4-6")
}

func TestAccountTestService_GetGroupTestModelsExcludesMediaAliases(t *testing.T) {
	repo := &groupTestAccountRepo{accounts: []Account{
		{
			ID:          4,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"fast-text":  "gpt-5-mini",
					"fast-image": "gpt-image-2",
				},
			},
		},
	}}
	service := &AccountTestService{accountRepo: repo}

	models, err := service.GetGroupTestModels(context.Background(), &Group{ID: 11, Platform: PlatformOpenAI})

	require.NoError(t, err)
	require.Contains(t, models, "fast-text")
	require.NotContains(t, models, "fast-image")
}
