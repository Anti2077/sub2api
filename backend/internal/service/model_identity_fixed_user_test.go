package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type identityFixedUserRepo struct {
	UserRepository
	user *User
}

func (r *identityFixedUserRepo) GetByID(context.Context, int64) (*User, error) { return r.user, nil }

type identityFixedSettingsRepo struct {
	IdentityRepository
	writes int
}

func (r *identityFixedSettingsRepo) SaveTestUserSettings(context.Context, int64, string) ([]string, error) {
	r.writes++
	return nil, nil
}

func TestIdentityFixedUserRequiresAnActiveUserAndCannotBeCleared(t *testing.T) {
	ctx := context.Background()
	t.Setenv("MODEL_IDENTITY_ENGINE_URL", "http://127.0.0.1:8081")
	store := &identitySettingsRepo{values: map[string]string{identitySettingsSetting: `{"test_user_id":5,"concurrent_accounts":1,"probe_concurrency":4}`}}
	repo := &identityFixedSettingsRepo{}
	users := &identityFixedUserRepo{}
	svc := NewModelIdentityService(repo, &APIKeyService{userRepo: users}, nil, store)
	candidate := DefaultIdentitySettings()
	candidate.TestUserID = 7
	for _, user := range []*User{nil, {ID: 7, Status: StatusDisabled}} {
		users.user = user
		_, err := svc.SaveSettings(ctx, candidate)
		require.ErrorContains(t, err, "active fixed test user")
	}
	require.Zero(t, repo.writes)
	users.user = &User{ID: 7, Status: StatusActive}
	_, err := svc.SaveSettings(ctx, candidate)
	require.NoError(t, err)
	require.Equal(t, 1, repo.writes)
	candidate.TestUserID = 0
	_, err = svc.SaveSettings(ctx, candidate)
	require.ErrorContains(t, err, "select a fixed test user")
	require.Equal(t, 1, repo.writes)
	_, err = svc.ValidateRun(ctx, &IdentityRun{UserID: 8})
	require.ErrorContains(t, err, "does not belong to the fixed test user")
}

func TestIdentityLocalCatalogTakesPrecedenceAndLegacyCatalogCannotQueueRuns(t *testing.T) {
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/models", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"engine_commit": IdentityEngineCommit, "models": []IdentityModel{{ID: "gpt-6-astra", Name: "gpt-6-astra/6.1-sol", Family: "gpt"}}})
	}))
	defer local.Close()
	svc := &ModelIdentityService{engineURL: local.URL, remoteURL: "http://legacy.invalid/api/probe/run", client: local.Client()}
	models, err := svc.Models(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, "gpt-6-astra", models[0].ID)
	require.Equal(t, IdentityEngineCommit, svc.EngineCommit())
	svc.engineURL = ""
	_, err = svc.Enqueue(context.Background(), 1)
	require.ErrorContains(t, err, "identity engine is not configured")
}
