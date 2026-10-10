//go:build integration

package repository

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestIdentityRepositoryLifecycleAndAtomicClaims(t *testing.T) {
	ctx := context.Background()
	repo := NewIdentityRepository(integrationDB)
	now := time.Now().UTC().Truncate(time.Second)
	group := mustCreateGroup(t, integrationEntClient, &service.Group{Name: fmt.Sprintf("identity-%d", now.UnixNano()), Platform: service.PlatformOpenAI, Status: service.StatusActive, RateMultiplier: 1})
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("identity-%d@example.com", now.UnixNano()), Status: service.StatusActive, Concurrency: 10})
	configs := []*service.IdentityConfig{}
	// These fixtures use the shared DB rather than a rollback-only Ent transaction.
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		for _, c := range configs {
			_, err := integrationDB.ExecContext(cleanupCtx, `DELETE FROM account_identity_configs WHERE account_id=$1`, c.AccountID)
			require.NoError(t, err)
			_, err = integrationDB.ExecContext(cleanupCtx, `DELETE FROM api_keys WHERE id=$1`, c.APIKeyID)
			require.NoError(t, err)
			_, err = integrationDB.ExecContext(cleanupCtx, `DELETE FROM accounts WHERE id=$1`, c.AccountID)
			require.NoError(t, err)
		}
		_, err := integrationDB.ExecContext(cleanupCtx, `DELETE FROM users WHERE id=$1`, user.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(cleanupCtx, `DELETE FROM groups WHERE id=$1`, group.ID)
		require.NoError(t, err)
	})
	for i := 0; i < 3; i++ {
		account := mustCreateAccount(t, integrationEntClient, &service.Account{Name: fmt.Sprintf("identity-account-%d", i), Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 4})
		c := &service.IdentityConfig{AccountID: account.ID, UserID: user.ID, GroupID: group.ID}
		require.NoError(t, repo.Configure(ctx, c, "identity dedicated", fmt.Sprintf("identity-%d-%d", now.UnixNano(), i)))
		configs = append(configs, c)
	}
	originalID := configs[0].APIKeyID
	require.NoError(t, repo.Configure(ctx, configs[0], "updated name", "unused-new-key"))
	require.Equal(t, originalID, configs[0].APIKeyID)
	for _, invalid := range []string{
		`deleted_at=NOW()`,
		`expires_at=NOW()-INTERVAL '1 minute'`,
		`quota=1,quota_used=1`,
	} {
		_, err := integrationDB.Exec(`UPDATE api_keys SET `+invalid+` WHERE id=$1`, originalID)
		require.NoError(t, err)
		require.Error(t, repo.Configure(ctx, configs[0], "cannot replace", "unused-replacement"))
		_, err = integrationDB.Exec(`UPDATE api_keys SET deleted_at=NULL,expires_at=NULL,quota=0,quota_used=0 WHERE id=$1`, originalID)
		require.NoError(t, err)
	}
	var name string
	require.NoError(t, integrationDB.QueryRow(`SELECT name FROM api_keys WHERE id=$1`, originalID).Scan(&name))
	require.Equal(t, "updated name", name)
	_, err := integrationDB.Exec(`UPDATE api_keys SET status='disabled' WHERE id=$1`, originalID)
	require.NoError(t, err)
	require.Error(t, repo.Configure(ctx, configs[0], "cannot replace", "unused-replacement"))
	require.Equal(t, originalID, configs[0].APIKeyID)
	_, err = integrationDB.Exec(`UPDATE api_keys SET status='active' WHERE id=$1`, originalID)
	require.NoError(t, err)
	plans := []*service.IdentityPlan{}
	for _, c := range configs {
		p := &service.IdentityPlan{AccountID: c.AccountID, RequestModel: "gpt-5.5", ExpectedModel: "openai/gpt-5.5", IntervalMinutes: 120, Enabled: true}
		require.NoError(t, repo.SavePlan(ctx, p, now))
		plans = append(plans, p)
	}
	page, err := repo.PlannedAccounts(ctx, 1, 1, "identity-account-")
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Equal(t, []int64{configs[2].AccountID}, page.AccountIDs)
	page, err = repo.PlannedAccounts(ctx, 2, 1, "identity-account-")
	require.NoError(t, err)
	require.Equal(t, []int64{configs[1].AccountID}, page.AccountIDs)
	page, err = repo.PlannedAccounts(ctx, 1, 20, "identity-account-0")
	require.NoError(t, err)
	require.Equal(t, []int64{configs[0].AccountID}, page.AccountIDs)
	due := now.Add(9 * time.Hour)
	require.NoError(t, repo.ScanDue(ctx, due))
	for _, p := range plans {
		saved, e := repo.Plan(ctx, p.ID)
		require.NoError(t, e)
		require.Equal(t, now.Add(10*time.Hour), *saved.NextRunAt)
	}
	require.NoError(t, repo.ScanDue(ctx, due))
	history, e := repo.History(ctx, plans[0].ID)
	require.NoError(t, e)
	require.Len(t, history, 1)
	var wg sync.WaitGroup
	var mu sync.Mutex
	claimed := []*service.IdentityRun{}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := repo.Claim(ctx, service.HashIdentityToken("test-capability"), due)
			if e == nil {
				mu.Lock()
				claimed = append(claimed, r)
				mu.Unlock()
			} else {
				t.Logf("claim: %v", e)
			}
		}()
	}
	wg.Wait()
	require.Len(t, claimed, 2)
	run := claimed[0]
	_, e = repo.Authorize(ctx, run.ID, service.HashIdentityToken("wrong"), due)
	require.Error(t, e)
	_, e = repo.Authorize(ctx, run.ID, service.HashIdentityToken("test-capability"), due)
	require.NoError(t, e)
	_, e = repo.Authorize(ctx, run.ID, service.HashIdentityToken("test-capability"), due.Add(20*time.Minute))
	require.Error(t, e)
	for i := 0; i < service.IdentityProbeConcurrency; i++ {
		require.NoError(t, repo.ProbeSlot(ctx, run.ID, true))
	}
	require.Error(t, repo.ProbeSlot(ctx, run.ID, true))
	require.NoError(t, repo.ProbeSlot(ctx, run.ID, false))
	require.NoError(t, repo.Cancel(ctx, run.ID))
	_, e = repo.Authorize(ctx, run.ID, service.HashIdentityToken("test-capability"), due)
	require.Error(t, e)
	require.NoError(t, repo.Finish(ctx, run.ID, "completed", []byte(`{"verdict":"matched"}`)))
	saved, e := repo.Run(ctx, run.ID)
	require.NoError(t, e)
	require.Equal(t, "cancelled", saved.Status)
	require.JSONEq(t, `{"verdict":"inconclusive"}`, string(saved.Report))
	require.NoError(t, repo.ScanDue(ctx, due.Add(21*time.Minute)))
	saved, e = repo.Run(ctx, claimed[1].ID)
	require.NoError(t, e)
	require.Equal(t, "timed_out", saved.Status)
	manual, e := repo.Enqueue(ctx, plans[0].ID)
	require.NoError(t, e)
	require.NotNil(t, manual)
	require.NoError(t, repo.Cancel(ctx, manual.ID))
	p, e := repo.Plan(ctx, plans[0].ID)
	require.NoError(t, e)
	require.Equal(t, now.Add(10*time.Hour), *p.NextRunAt)
	p.Enabled = false
	require.NoError(t, repo.SavePlan(ctx, p, due))
	p, e = repo.Plan(ctx, p.ID)
	require.NoError(t, e)
	require.Nil(t, p.NextRunAt)
	p.Enabled = true
	require.NoError(t, repo.SavePlan(ctx, p, due))
	p, e = repo.Plan(ctx, p.ID)
	require.NoError(t, e)
	require.Equal(t, due.Add(2*time.Hour), *p.NextRunAt)
	for i := 0; i < 55; i++ {
		pending, e := repo.Enqueue(ctx, p.ID)
		require.NoError(t, e)
		require.NoError(t, repo.Cancel(ctx, pending.ID))
	}
	var retained int
	require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM account_identity_runs WHERE plan_id=$1`, p.ID).Scan(&retained))
	require.Equal(t, 50, retained)
	remaining, err := repo.History(ctx, plans[2].ID)
	require.NoError(t, err)
	for _, pending := range remaining {
		require.NoError(t, repo.Cancel(ctx, pending.ID))
	}
	require.NoError(t, repo.DeletePlan(ctx, plans[2].ID))
	page, err = repo.PlannedAccounts(ctx, 1, 20, "identity-account-")
	require.NoError(t, err)
	require.Equal(t, 2, page.Total, "configuration without a plan is not listed")
	_, err = integrationDB.Exec(`UPDATE accounts SET deleted_at=NOW() WHERE id=$1`, configs[1].AccountID)
	require.NoError(t, err)
	page, err = repo.PlannedAccounts(ctx, 1, 20, "identity-account-")
	require.NoError(t, err)
	require.Equal(t, []int64{configs[0].AccountID}, page.AccountIDs)
}
