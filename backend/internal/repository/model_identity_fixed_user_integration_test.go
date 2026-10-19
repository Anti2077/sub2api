//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestIdentityFixedUserChangesBindingsAtomicallyAndPreservesRuns(t *testing.T) {
	ctx := context.Background()
	repo := NewIdentityRepository(integrationDB)
	var originalSettings string
	require.NoError(t, integrationDB.QueryRow(`SELECT value FROM settings WHERE key='model_identity_settings'`).Scan(&originalSettings))
	stamp := time.Now().UnixNano()
	group := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("fixed-user-%d", stamp), Platform: service.PlatformOpenAI, Status: service.StatusActive})
	first := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("fixed-first-%d@example.com", stamp), Status: service.StatusActive})
	second := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("fixed-second-%d@example.com", stamp), Status: service.StatusActive})
	configs := []*service.IdentityConfig{}
	t.Cleanup(func() {
		for _, c := range configs {
			_, err := integrationDB.Exec(`DELETE FROM account_identity_configs WHERE account_id=$1`, c.AccountID)
			require.NoError(t, err)
			_, err = integrationDB.Exec(`DELETE FROM api_keys WHERE id=$1`, c.APIKeyID)
			require.NoError(t, err)
			_, err = integrationDB.Exec(`DELETE FROM accounts WHERE id=$1`, c.AccountID)
			require.NoError(t, err)
		}
		_, err := integrationDB.Exec(`UPDATE settings SET value=$1 WHERE key='model_identity_settings'`, originalSettings)
		require.NoError(t, err)
		_, err = integrationDB.Exec(`DELETE FROM users WHERE id IN ($1,$2)`, first.ID, second.ID)
		require.NoError(t, err)
		_, err = integrationDB.Exec(`DELETE FROM groups WHERE id=$1`, group.ID)
		require.NoError(t, err)
	})
	for i := 0; i < 2; i++ {
		a := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "fixed-user-target", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive})
		c := &service.IdentityConfig{AccountID: a.ID, UserID: first.ID, GroupID: group.ID}
		require.NoError(t, repo.Configure(ctx, c, "dedicated", fmt.Sprintf("fixed-%d-%d", stamp, i)))
		configs = append(configs, c)
	}
	settings := fmt.Sprintf(`{"test_user_id":%d,"concurrent_accounts":1,"probe_concurrency":4,"public_base_url":""}`, second.ID)
	keys, err := repo.SaveTestUserSettings(ctx, second.ID, settings)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	for _, c := range configs {
		stored, err := repo.Config(ctx, c.AccountID)
		require.NoError(t, err)
		require.Equal(t, second.ID, stored.UserID)
		require.Equal(t, c.APIKeyID, stored.APIKeyID)
		var owner int64
		require.NoError(t, integrationDB.QueryRow(`SELECT user_id FROM api_keys WHERE id=$1`, c.APIKeyID).Scan(&owner))
		require.Equal(t, second.ID, owner)
	}
	// A stale client must never put an account back on its old test user.
	require.ErrorContains(t, repo.Configure(ctx, configs[0], "stale", "unused"), "fixed test user changed")
	plan := &service.IdentityPlan{AccountID: configs[0].AccountID, RequestModel: "gpt-6.1-sol", ExpectedModel: "gpt-6-astra", IntervalMinutes: 120}
	require.NoError(t, repo.SavePlan(ctx, plan, time.Now()))
	run, err := repo.Enqueue(ctx, plan.ID)
	require.NoError(t, err)
	require.Equal(t, second.ID, run.UserID)
	next := fmt.Sprintf(`{"test_user_id":%d,"concurrent_accounts":1,"probe_concurrency":4}`, first.ID)
	_, err = repo.SaveTestUserSettings(ctx, first.ID, next)
	require.ErrorContains(t, err, "active detections")
	var persisted string
	require.NoError(t, integrationDB.QueryRow(`SELECT value FROM settings WHERE key='model_identity_settings'`).Scan(&persisted))
	require.Equal(t, settings, persisted)
	// Saving concurrency for the current user is allowed during an active run.
	_, err = repo.SaveTestUserSettings(ctx, second.ID, settings)
	require.NoError(t, err)
	require.NoError(t, repo.Cancel(ctx, run.ID))
	_, err = repo.SaveTestUserSettings(ctx, first.ID, next)
	require.NoError(t, err)
	historical, err := repo.Run(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, second.ID, historical.UserID)
	require.Equal(t, "cancelled", historical.Status)
}
