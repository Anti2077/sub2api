//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIdentitySelectionStaysOnTargetAndNeverFallsBack(t *testing.T) {
	group := int64(7)
	a := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: false, Concurrency: 4, AccountGroups: []AccountGroup{{GroupID: group}}, Credentials: map[string]any{}}
	other := &Account{ID: 2, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: true, AccountGroups: []AccountGroup{{GroupID: group}}}
	repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{1: a, 2: other}, accounts: []Account{*a, *other}}
	svc := &GatewayService{accountRepo: repo, groupRepo: &mockGroupRepoForGateway{groups: map[int64]*Group{group: {ID: group, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}}}, cfg: testConfig()}
	ctx := WithIdentityTarget(context.Background(), 1, group)
	selected, e := svc.SelectAccountWithLoadAwareness(ctx, &group, "", "claude-sonnet-4-6", nil, "", 0)
	require.NoError(t, e)
	require.EqualValues(t, 1, selected.Account.ID)
	require.False(t, a.Schedulable)
	require.False(t, selected.Account.Schedulable)
	selected.ReleaseFunc()
	// Use the ordinary wait queue when only the pinned account is full.
	svc.concurrencyService = NewConcurrencyService(&mockConcurrencyCache{acquireResults: map[int64]bool{1: false}})
	svc.cfg.Gateway.Scheduling.FallbackWaitTimeout = 30 * time.Second
	svc.cfg.Gateway.Scheduling.FallbackMaxWaiting = 100
	selected, e = svc.SelectAccountWithLoadAwareness(ctx, &group, "", "claude-sonnet-4-6", nil, "", 0)
	require.NoError(t, e)
	require.EqualValues(t, 1, selected.Account.ID)
	require.False(t, selected.Acquired)
	require.EqualValues(t, 1, selected.WaitPlan.AccountID)
	require.Equal(t, a.Concurrency, selected.WaitPlan.MaxConcurrency)
	require.Equal(t, 30*time.Second, selected.WaitPlan.Timeout)
	a.Status = StatusError
	selected, e = svc.SelectAccountWithLoadAwareness(ctx, &group, "", "claude-sonnet-4-6", nil, "", 0)
	require.Error(t, e)
	require.Nil(t, selected)
	a.Status = StatusActive
	a.AccountGroups = nil
	selected, e = svc.SelectAccountWithLoadAwareness(ctx, &group, "", "claude-sonnet-4-6", nil, "", 0)
	require.Error(t, e)
	require.Nil(t, selected)
}

// Hydration owns releasing an acquired slot on failure, including pinned probes.
func TestIdentitySelectionHydrationFailureReleasesSlotOnce(t *testing.T) {
	group := int64(7)
	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 4, AccountGroups: []AccountGroup{{GroupID: group}}, Credentials: map[string]any{}}
	ctx, cancel := context.WithCancel(WithIdentityTarget(context.Background(), account.ID, group))
	defer cancel()
	cache := &identityHydrationConcurrencyCache{cancel: cancel}
	svc := &GatewayService{
		accountRepo: &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{account.ID: account}},
		groupRepo:   &mockGroupRepoForGateway{groups: map[int64]*Group{group: {ID: group, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}}},
		cfg:         testConfig(), concurrencyService: NewConcurrencyService(cache),
		schedulerSnapshot: &SchedulerSnapshotService{},
	}
	selected, pinned, err := svc.selectIdentityTarget(ctx, &group, "claude-sonnet-4-6", nil)
	require.Error(t, err)
	require.True(t, pinned)
	require.Nil(t, selected)
	require.Equal(t, 1, cache.releases)
}

type identityHydrationConcurrencyCache struct {
	mockConcurrencyCache
	cancel   context.CancelFunc
	releases int
}

func (c *identityHydrationConcurrencyCache) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	c.cancel()
	return true, nil
}
func (c *identityHydrationConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error {
	c.releases++
	return nil
}
