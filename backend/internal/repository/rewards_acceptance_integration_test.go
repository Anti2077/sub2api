//go:build integration

package repository

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRewardsLifecycleScopeAndRequestSnapshot(t *testing.T) {
	ctx := context.Background()
	s, u, g, k, tmpl := rewardFixture(t)
	now := time.Now().UTC()
	s.Now = func() time.Time { return now }
	tmpl.ID, tmpl.Version, tmpl.Kind, tmpl.Hours = 0, 0, "model_day", 24
	tmpl, err := s.SaveTemplate(ctx, tmpl, u.ID)
	require.NoError(t, err)
	first, err := s.Grant(ctx, u.ID, tmpl.ID, u.ID, uuid.NewString(), "snapshot acceptance")
	require.NoError(t, err)
	second, err := s.Grant(ctx, u.ID, tmpl.ID, u.ID, uuid.NewString(), "next manual activation")
	require.NoError(t, err)
	require.NoError(t, s.Activate(ctx, u.ID, first))
	require.Error(t, s.Activate(ctx, u.ID, second))
	for _, tc := range []struct {
		group int64
		model string
	}{{g.ID + 1000000, "luna"}, {g.ID, "wrong-model"}} {
		c, err := s.Capture(ctx, u.ID, tc.group, tc.model, true)
		require.NoError(t, err)
		require.Nil(t, c)
	}
	captured, err := s.Capture(ctx, u.ID, g.ID, "luna", true)
	require.NoError(t, err)
	require.True(t, rewards.FreeText(captured))
	// Publishing a new template version must not mutate the already issued card.
	tmpl.Model = "new-model"
	_, err = s.SaveTemplate(ctx, tmpl, u.ID)
	require.NoError(t, err)
	c, err := s.Capture(ctx, u.ID, g.ID, "luna", true)
	require.NoError(t, err)
	require.True(t, rewards.FreeText(c))
	require.NoError(t, s.Revoke(ctx, first, u.ID, "test revocation"))
	c, err = s.Capture(ctx, u.ID, g.ID, "luna", true)
	require.NoError(t, err)
	require.Nil(t, c)
	// The in-flight request retains its admission-time entitlement after revoke.
	repo := NewUsageBillingRepository(testEntClient(t), integrationDB)
	cmd := rewardCmd(u, k, captured, uuid.NewString(), 100)
	r, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.Zero(t, *r.PaidCost)
	// A restarted worker may rehydrate the original command without an active
	// entitlement. The durable request/key receipt still determines the bill.
	cmd.RewardCapture = nil
	r, err = repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, r.Applied)
	require.NotNil(t, r.PaidCost)
	require.Zero(t, *r.PaidCost)
	require.NoError(t, s.Activate(ctx, u.ID, second))
	now = now.Add(24 * time.Hour)
	c, err = s.Capture(ctx, u.ID, g.ID, "luna", true)
	require.NoError(t, err)
	require.Nil(t, c, "24-hour boundary is exclusive and no automatic renewal")
}

func TestRewardsRandomTokenUsageIntegerOracle(t *testing.T) {
	ctx := context.Background()
	s, u, g, k, tmpl := rewardFixture(t)
	tmpl.ID, tmpl.Version, tmpl.Tokens = 0, 0, 10003
	tmpl, err := s.SaveTemplate(ctx, tmpl, u.ID)
	require.NoError(t, err)
	card, err := s.Grant(ctx, u.ID, tmpl.ID, u.ID, uuid.NewString(), "random integer oracle")
	require.NoError(t, err)
	require.NoError(t, s.Activate(ctx, u.ID, card))
	c, err := s.Capture(ctx, u.ID, g.ID, "luna", true)
	require.NoError(t, err)
	repo := NewUsageBillingRepository(testEntClient(t), integrationDB)
	rng := rand.New(rand.NewSource(721))
	left, paidUnits, covered := int64(10003), int64(0), int64(0)
	for i := 0; i < 100; i++ {
		n := int64(rng.Intn(500) + 1)
		use := min(left, n)
		left -= use
		covered += use
		// Independent oracle: 100 money units per token and 17 special-fee
		// units per call. No call into the production discount calculator.
		want := (n-use)*100 + 17
		paidUnits += want
		cmd := rewardCmd(u, k, c, fmt.Sprintf("random-%d-%d", card, i), int(n))
		cmd.RewardTextCost = float64(n*100) / 1e8
		cmd.BalanceCost = float64(n*100+17) / 1e8
		cmd.APIKeyQuotaCost = cmd.BalanceCost
		r, err := repo.Apply(ctx, cmd)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%.8f", float64(want)/1e8), fmt.Sprintf("%.8f", *r.PaidCost))
	}
	var remaining, used int64
	var paid string
	require.NoError(t, integrationDB.QueryRow(`SELECT remaining_tokens FROM reward_cards WHERE id=$1`, card).Scan(&remaining))
	require.NoError(t, integrationDB.QueryRow(`SELECT SUM(covered_tokens),SUM(paid_cost)::text FROM reward_usage WHERE user_id=$1`, u.ID).Scan(&used, &paid))
	require.Equal(t, left, remaining)
	require.Equal(t, covered, used)
	require.Equal(t, fmt.Sprintf("%.8f", float64(paidUnits)/1e8), paid)
}

func TestFarmProtectionCapSnapshotAndHarvestStealRace(t *testing.T) {
	ctx := context.Background()
	s, owner, _, _, _ := rewardFixture(t)
	// PostgreSQL rounds timestamps to microseconds. Align the fake clock so
	// the exact protection boundary cannot precede the stored ready_at.
	now := time.Now().UTC().Truncate(time.Microsecond)
	s.Now = func() time.Time { return now }
	act := func(user int64, kind string, slot int, other int64) error {
		_, err := s.Action(ctx, user, rewards.FarmAction{Key: uuid.NewString(), Kind: kind, Slot: slot, OwnerID: other, CropID: "sprout", Public: true})
		return err
	}
	require.NoError(t, act(owner.ID, "privacy", 0, 0))
	require.NoError(t, act(owner.ID, "plant", 1, 0))
	require.NoError(t, act(owner.ID, "plant", 2, 0))
	thieves := make([]int64, 5)
	for i := range thieves {
		_, thief, _, _, _ := rewardFixture(t)
		thieves[i] = thief.ID
		require.NoError(t, act(thief.ID, "privacy", 0, 0))
	}
	now = now.Add(time.Hour)
	require.Error(t, act(thieves[0], "steal", 1, owner.ID), "30-minute protection")
	// Changing future crop settings does not rewrite an existing crop snapshot.
	cfg, err := s.Config(ctx, integrationDB)
	require.NoError(t, err)
	cfg.Crops[0].Yield, cfg.ProtectionMinutes, cfg.MaxStolenPercent = 1000, 120, 0
	_, err = s.SaveConfig(ctx, cfg, owner.ID)
	require.NoError(t, err)
	now = now.Add(30 * time.Minute)
	for i := 0; i < 4; i++ {
		require.NoError(t, act(thieves[i], "steal", 1, owner.ID))
	}
	require.Error(t, act(thieves[4], "steal", 1, owner.ID), "40-percent lifetime cap")
	require.Error(t, act(thieves[0], "steal", 1, owner.ID), "same thief/cycle cannot collect twice")
	require.NoError(t, act(owner.ID, "harvest", 1, 0))
	var crystals int64
	require.NoError(t, integrationDB.QueryRow(`SELECT crystals FROM farm_users WHERE user_id=$1`, owner.ID).Scan(&crystals))
	require.Equal(t, int64(6), crystals, "owner keeps 60 percent of original yield 10")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_ = act(owner.ID, "harvest", 2, 0)
			} else {
				_ = act(thieves[4], "steal", 2, owner.ID)
			}
		}(i)
	}
	wg.Wait()
	var total int64
	require.NoError(t, integrationDB.QueryRow(`SELECT SUM(delta) FROM farm_ledger WHERE reference=(SELECT cycle FROM farm_plots WHERE user_id=$1 AND slot=2)`, owner.ID).Scan(&total))
	require.Equal(t, int64(10), total, "competing harvest/steal conserves total yield")
}

func TestFarmLastStockConcurrentBuyAndRollback(t *testing.T) {
	ctx := context.Background()
	s, u, _, _, tmpl := rewardFixture(t)
	_, err := s.Farm(ctx, u.ID, u.ID)
	require.NoError(t, err)
	_, err = integrationDB.Exec(`UPDATE farm_users SET crystals=100 WHERE user_id=$1`, u.ID)
	require.NoError(t, err)
	shop, err := s.SaveShop(ctx, rewards.Shop{TemplateID: tmpl.ID, Price: 7, Stock: 1, DailyLimit: 1, TotalLimit: 1, Published: true}, u.ID)
	require.NoError(t, err)
	var wg sync.WaitGroup
	success := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Action(ctx, u.ID, rewards.FarmAction{Key: uuid.NewString(), Kind: "buy", ShopID: shop.ID})
			success <- err == nil
		}()
	}
	wg.Wait()
	close(success)
	n := 0
	for ok := range success {
		if ok {
			n++
		}
	}
	require.Equal(t, 1, n)
	var stock, crystals, orders int64
	require.NoError(t, integrationDB.QueryRow(`SELECT stock FROM farm_shop WHERE id=$1`, shop.ID).Scan(&stock))
	require.NoError(t, integrationDB.QueryRow(`SELECT crystals FROM farm_users WHERE user_id=$1`, u.ID).Scan(&crystals))
	require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM farm_orders WHERE shop_id=$1`, shop.ID).Scan(&orders))
	require.Zero(t, stock)
	require.Equal(t, int64(93), crystals)
	require.Equal(t, int64(1), orders)
}

func TestRewardsDurableGatewayClaim(t *testing.T) {
	ctx := context.Background()
	s, _, _, k, _ := rewardFixture(t)
	id := uuid.NewString()
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := s.ClaimGateway(ctx, k.ID, id, "fingerprint", "original-request")
			require.NoError(t, e)
			if r.Claimed {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, winners)
	// New Store represents a restarted application; claims survive it.
	restarted := rewards.New(s.DB)
	r, e := restarted.ClaimGateway(ctx, k.ID, id, "fingerprint", "new-attempt")
	require.NoError(t, e)
	require.False(t, r.Claimed)
	require.Equal(t, "pending", r.State)
	require.NoError(t, s.FinishGateway(ctx, k.ID, id, 200, "text/event-stream", []byte("data: done\n\n"), false))
	r, e = restarted.ClaimGateway(ctx, k.ID, id, "different-fingerprint", "new-attempt")
	require.NoError(t, e)
	require.False(t, r.Claimed)
	require.Equal(t, "fingerprint", r.Fingerprint)
	require.Equal(t, "original-request", r.RequestID)
	require.Equal(t, "complete", r.State)
	require.Equal(t, "data: done\n\n", string(r.Body))
}
