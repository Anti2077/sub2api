//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIdentitySelectionStaysOnTargetAndNeverFallsBack(t *testing.T) {
	group := int64(7)
	a := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 4, AccountGroups: []AccountGroup{{GroupID: group}}, Credentials: map[string]any{}}
	other := &Account{ID: 2, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: true, AccountGroups: []AccountGroup{{GroupID: group}}}
	repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{1: a, 2: other}, accounts: []Account{*a, *other}}
	svc := &GatewayService{accountRepo: repo, groupRepo: &mockGroupRepoForGateway{groups: map[int64]*Group{group: {ID: group, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}}}, cfg: testConfig()}
	ctx := WithIdentityTarget(context.Background(), 1, group)
	selected, e := svc.SelectAccountWithLoadAwareness(ctx, &group, "", "claude-sonnet-4-6", nil, "", 0)
	require.NoError(t, e)
	require.EqualValues(t, 1, selected.Account.ID)
	selected.ReleaseFunc()
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
