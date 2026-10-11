package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIdentityProbeAllowsManualPauseOnlyForBoundTarget(t *testing.T) {
	a := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: false, GroupIDs: []int64{7}}
	ctx := WithIdentityTarget(context.Background(), 42, 7)
	require.True(t, a.IsSchedulableForModelWithContext(ctx, "gpt-6-sol"))
	require.False(t, a.Schedulable)
	require.False(t, a.IsSchedulable())
	require.False(t, a.IsSchedulableForModel("gpt-6-sol"))
	require.False(t, a.IsSchedulableForModelWithContext(WithIdentityTarget(context.Background(), 43, 7), "gpt-6-sol"))
	require.False(t, a.IsSchedulableForModelWithContext(WithIdentityTarget(context.Background(), 42, 8), "gpt-6-sol"))
}

func TestIdentityProbePauseExceptionPreservesAvailabilityGates(t *testing.T) {
	future, past := time.Now().Add(time.Hour), time.Now().Add(-time.Hour)
	for _, tc := range []struct {
		name  string
		block func(*Account)
	}{
		{"disabled", func(a *Account) { a.Status = StatusDisabled }},
		{"error", func(a *Account) { a.Status = StatusError }},
		{"expired", func(a *Account) { a.AutoPauseOnExpired = true; a.ExpiresAt = &past }},
		{"overloaded", func(a *Account) { a.OverloadUntil = &future }},
		{"rate_limited", func(a *Account) { a.RateLimitResetAt = &future }},
		{"cooldown", func(a *Account) { a.TempUnschedulableUntil = &future }},
		{"quota_exhausted", func(a *Account) { a.Extra = map[string]any{"quota_limit": 100.0, "quota_used": 100.0} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
				Status: StatusActive, Schedulable: false, GroupIDs: []int64{7}}
			tc.block(a)
			require.False(t, a.IsSchedulableForModelWithContext(WithIdentityTarget(context.Background(), 42, 7), "gpt-6-sol"))
			require.False(t, a.Schedulable)
		})
	}
}
