//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// Image tier filtering must apply to authenticated probes without permitting
// the ordinary scheduler to substitute a different account for their target.
func TestIdentityImageRoutingPreservesTargetAndSizeRestrictions(t *testing.T) {
	for _, mode := range openAIImageSizeSchedulerModes {
		t.Run(mode.name, func(t *testing.T) {
			groupID := int64(94001)
			target := newOpenAIImageSizeTestAccount(82001, 9, "4K")
			target.Schedulable = false // Authenticated probes may inspect paused targets.
			target.GroupIDs = []int64{groupID}
			other := newOpenAIImageSizeTestAccount(82002, 0)
			other.GroupIDs = []int64{groupID}
			svc := newOpenAIImageSizeSchedulerTestService(mode, []Account{target, other}, nil, schedulerTestConcurrencyCache{})
			ctx := WithIdentityTarget(context.Background(), target.ID, groupID)
			ctx = WithOpenAIImageSizeRouting(ctx, "3840x2160")
			selection, _, err := svc.SelectAccountWithSchedulerForImages(ctx, &groupID, "", "gpt-image-2", nil, OpenAIImagesCapabilityNative)
			require.NoError(t, err)
			require.Equal(t, target.ID, selection.Account.ID)
			require.True(t, selection.Acquired)
			require.False(t, selection.Account.Schedulable)
			selection.ReleaseFunc()

			// A healthy unrestricted alternative must never mask a target's tier veto.
			disallowed := WithOpenAIImageSizeRouting(ctx, "1024x1024")
			selection, _, err = svc.SelectAccountWithSchedulerForImages(disallowed, &groupID, "", "gpt-image-2", nil, OpenAIImagesCapabilityNative)
			require.Error(t, err)
			require.Nil(t, selection)

			selection, _, err = svc.SelectAccountWithSchedulerForImages(ctx, &groupID, "", "gpt-image-2", map[int64]struct{}{target.ID: {}}, OpenAIImagesCapabilityNative)
			require.ErrorContains(t, err, "failover is forbidden")
			require.Nil(t, selection)
		})
	}
}
