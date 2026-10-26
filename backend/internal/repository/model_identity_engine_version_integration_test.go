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

func TestIdentityManualAndScheduledRunsSnapshotCurrentEngine(t *testing.T) {
	ctx := context.Background()
	repo := NewIdentityRepository(integrationDB)
	now := time.Now().UTC()
	group := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("engine-pin-%d", now.UnixNano()), Platform: service.PlatformOpenAI, Status: service.StatusActive})
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("engine-pin-%d@example.com", now.UnixNano()), Status: service.StatusActive})
	account := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "engine-pin-target", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive})
	config := &service.IdentityConfig{AccountID: account.ID, UserID: user.ID, GroupID: group.ID}
	var originalSettings string
	require.NoError(t, integrationDB.QueryRow(`SELECT value FROM settings WHERE key='model_identity_settings'`).Scan(&originalSettings))
	t.Cleanup(func() {
		_, err := integrationDB.Exec(`DELETE FROM account_identity_configs WHERE account_id=$1`, account.ID)
		require.NoError(t, err)
		_, err = integrationDB.Exec(`DELETE FROM api_keys WHERE id=$1`, config.APIKeyID)
		require.NoError(t, err)
		_, err = integrationDB.Exec(`DELETE FROM accounts WHERE id=$1`, account.ID)
		require.NoError(t, err)
		_, err = integrationDB.Exec(`UPDATE settings SET value=$1 WHERE key='model_identity_settings'`, originalSettings)
		require.NoError(t, err)
		_, err = integrationDB.Exec(`DELETE FROM users WHERE id=$1`, user.ID)
		require.NoError(t, err)
		_, err = integrationDB.Exec(`DELETE FROM groups WHERE id=$1`, group.ID)
		require.NoError(t, err)
	})
	require.NoError(t, repo.Configure(ctx, config, "engine pin", fmt.Sprintf("engine-pin-%d", now.UnixNano())))
	_, err := repo.SaveTestUserSettings(ctx, user.ID, fmt.Sprintf(`{"test_user_id":%d,"concurrent_accounts":1,"probe_concurrency":4}`, user.ID))
	require.NoError(t, err)
	plan := &service.IdentityPlan{AccountID: account.ID, RequestModel: "gpt-6.1-sol", ExpectedModel: "gpt-6-astra", IntervalMinutes: 120, Enabled: true}
	require.NoError(t, repo.SavePlan(ctx, plan, now))
	manual, err := repo.Enqueue(ctx, plan.ID)
	require.NoError(t, err)
	require.Equal(t, service.IdentityEngineCommit, manual.EngineCommit)
	require.NoError(t, repo.Cancel(ctx, manual.ID))
	// Exercise the scheduler's separate SQL call, not just manual enqueue.
	require.NoError(t, repo.ScanDue(ctx, now.Add(3*time.Hour)))
	var scheduledID int64
	require.NoError(t, integrationDB.QueryRow(`SELECT id FROM account_identity_runs WHERE plan_id=$1 AND id<>$2`, plan.ID, manual.ID).Scan(&scheduledID))
	scheduled, err := repo.Run(ctx, scheduledID)
	require.NoError(t, err)
	require.Equal(t, service.IdentityEngineCommit, scheduled.EngineCommit)
	require.NoError(t, repo.Cancel(ctx, scheduled.ID))
}
