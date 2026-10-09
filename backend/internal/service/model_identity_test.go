package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIdentityScheduleKeepsCadenceAndSkipsBacklog(t *testing.T) {
	due := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct{ now, next time.Time }{{due, due.Add(2 * time.Hour)}, {due.Add(9 * time.Hour), due.Add(10 * time.Hour)}, {due.Add(-time.Minute), due}} {
		require.Equal(t, tt.next, AdvanceIdentitySchedule(due, tt.now, 2*time.Hour))
	}
}
func TestIdentityPinRejectsGroupChangesAndFailedTarget(t *testing.T) {
	ctx := WithIdentityTarget(context.Background(), 42, 7)
	group := int64(7)
	id, pinned, err := identityTargetAllowed(ctx, &group, nil)
	require.NoError(t, err)
	require.True(t, pinned)
	require.EqualValues(t, 42, id)
	group = 8
	_, _, err = identityTargetAllowed(ctx, &group, nil)
	require.Error(t, err)
	group = 7
	_, _, err = identityTargetAllowed(ctx, &group, map[int64]struct{}{42: {}})
	require.Error(t, err)
	_, pinned, err = identityTargetAllowed(context.Background(), &group, map[int64]struct{}{42: {}})
	require.NoError(t, err)
	require.False(t, pinned)
}
func TestIdentityCapabilityIsRandomAndStoredAsHash(t *testing.T) {
	a, e := newIdentityToken()
	require.NoError(t, e)
	b, e := newIdentityToken()
	require.NoError(t, e)
	require.NotEqual(t, a, b)
	require.Len(t, a, 64)
	require.NotEqual(t, a, HashIdentityToken(a))
}
