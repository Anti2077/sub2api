//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFarmExpansionPreservesPlantedSnapshots(t *testing.T) {
	ctx := context.Background()
	s, user, _, _, _ := rewardFixture(t)
	cfg, err := s.Config(ctx, integrationDB)
	require.NoError(t, err)
	cfg.Plots = 12
	cfg, err = s.SaveConfig(ctx, cfg, user.ID)
	require.NoError(t, err)
	_, err = s.Action(ctx, user.ID, rewards.FarmAction{Key: uuid.NewString(), Kind: "plant", Slot: 12, CropID: "sprout"})
	require.NoError(t, err)
	before, err := s.Farm(ctx, user.ID, user.ID)
	require.NoError(t, err)
	require.Len(t, before.Plots, 12)
	oldPlot := before.Plots[11]

	cfg.Plots = 16
	cfg, err = s.SaveConfig(ctx, cfg, user.ID)
	require.NoError(t, err)
	after, err := s.Farm(ctx, user.ID, user.ID)
	require.NoError(t, err)
	require.Len(t, after.Plots, 16)
	require.Equal(t, oldPlot, after.Plots[11], "expansion must preserve crop snapshot and timer")
	_, err = s.Action(ctx, user.ID, rewards.FarmAction{Key: uuid.NewString(), Kind: "plant", Slot: 16, CropID: "sprout"})
	require.NoError(t, err)
	_, err = s.Action(ctx, user.ID, rewards.FarmAction{Key: uuid.NewString(), Kind: "plant", Slot: 17, CropID: "sprout"})
	require.Error(t, err)

	cfg.Plots = 12
	_, err = s.SaveConfig(ctx, cfg, user.ID)
	require.NoError(t, err)
	after, err = s.Farm(ctx, user.ID, user.ID)
	require.NoError(t, err)
	require.Equal(t, oldPlot, after.Plots[11])
	require.Equal(t, 16, after.Plots[len(after.Plots)-1].Slot, "reducing configured plots must retain existing planted fields")
}
