package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewardTextChargeRetainsMediaFees(t *testing.T) {
	// $6 of text and $4 of media at 0.35x: only $2.10 is eligible.
	require.InDelta(t, 2.1, rewardTextCharge(&CostBreakdown{TotalCost: 10, ActualCost: 3.5, ImageInputCost: 3, ImageOutputCost: 1}), 1e-12)
	require.Zero(t, rewardTextCharge(&CostBreakdown{TotalCost: 10, ActualCost: 3.5, ImageInputCost: 10}))
	require.Zero(t, rewardTextCharge(nil))
}
