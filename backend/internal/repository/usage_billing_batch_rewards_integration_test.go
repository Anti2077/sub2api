//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUsageBillingBatchRewards_SettlementAndReplay(t *testing.T) {
	for _, subscription := range []bool{false, true} {
		name := "balance"
		if subscription {
			name = "subscription"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store, user, group, key, tmpl := rewardFixture(t)
			cardID, err := store.Grant(ctx, user.ID, tmpl.ID, user.ID, uuid.NewString(), "batch regression")
			require.NoError(t, err)
			require.NoError(t, store.Activate(ctx, user.ID, cardID))
			capture, err := store.Capture(ctx, user.ID, group.ID, "luna", true)
			require.NoError(t, err)
			require.NotNil(t, capture)
			repo := NewUsageBillingRepository(testEntClient(t), integrationDB)
			cmds := []*service.UsageBillingCommand{
				rewardCmd(user, key, capture, uuid.NewString(), 600),
				rewardCmd(user, key, capture, uuid.NewString(), 600),
				rewardCmd(user, key, nil, uuid.NewString(), 600),
			}
			account := mustCreateAccount(t, testEntClient(t), &service.Account{
				Name: "batch-reward-" + uuid.NewString(), Type: service.AccountTypeAPIKey,
				Extra: map[string]any{"quota_limit": 100.0},
			})
			for _, cmd := range cmds {
				cmd.APIKeyRateLimitCost = 1
				cmd.AccountID, cmd.AccountType, cmd.AccountQuotaCost = account.ID, service.AccountTypeAPIKey, 2
			}
			var subscriptionID int64
			if subscription {
				sub := mustCreateSubscription(t, testEntClient(t), &service.UserSubscription{UserID: user.ID, GroupID: group.ID})
				subscriptionID = sub.ID
				t.Cleanup(func() {
					_, err := integrationDB.Exec(`DELETE FROM user_subscriptions WHERE id=$1`, subscriptionID)
					require.NoError(t, err)
				})
				for _, cmd := range cmds {
					cmd.SubscriptionID = &subscriptionID
					cmd.SubscriptionCost, cmd.BalanceCost = cmd.BalanceCost, 0
				}
			}
			outcomes, err := repo.ApplyBatch(ctx, cmds)
			require.NoError(t, err)
			require.Len(t, outcomes, 3)
			for _, out := range outcomes {
				require.NoError(t, out.Err)
				require.True(t, out.Result.Applied)
			}
			require.NotNil(t, outcomes[0].Result.PaidCost)
			require.Zero(t, *outcomes[0].Result.PaidCost)
			require.NotNil(t, outcomes[1].Result.PaidCost)
			require.InDelta(t, 0.33333333, *outcomes[1].Result.PaidCost, 1e-9)
			require.Nil(t, outcomes[2].Result.PaidCost)
			for _, cmd := range cmds {
				require.Equal(t, 1.0, cmd.APIKeyQuotaCost, "settlement must not mutate retry commands")
				if subscription {
					require.Equal(t, 1.0, cmd.SubscriptionCost)
				} else {
					require.Equal(t, 1.0, cmd.BalanceCost)
				}
			}
			var left int64
			var balance, used float64
			require.NoError(t, integrationDB.QueryRow(`SELECT remaining_tokens FROM reward_cards WHERE id=$1`, cardID).Scan(&left))
			require.Zero(t, left)
			require.NoError(t, integrationDB.QueryRow(`SELECT balance FROM users WHERE id=$1`, user.ID).Scan(&balance))
			require.NoError(t, integrationDB.QueryRow(`SELECT quota_used FROM api_keys WHERE id=$1`, key.ID).Scan(&used))
			require.InDelta(t, 1.33333333, used, 1e-9)
			var rateUsage, accountUsage float64
			require.NoError(t, integrationDB.QueryRow(`SELECT usage_5h FROM api_keys WHERE id=$1`, key.ID).Scan(&rateUsage))
			require.NoError(t, integrationDB.QueryRow(`SELECT (extra->>'quota_used')::numeric FROM accounts WHERE id=$1`, account.ID).Scan(&accountUsage))
			require.InDelta(t, 1.33333333, rateUsage, 1e-9)
			require.Equal(t, 6.0, accountUsage, "reward discounts must not change upstream account cost")
			if subscription {
				require.Equal(t, 100.0, balance)
				var daily, weekly, monthly float64
				require.NoError(t, integrationDB.QueryRow(`SELECT daily_usage_usd,weekly_usage_usd,monthly_usage_usd FROM user_subscriptions WHERE id=$1`, subscriptionID).Scan(&daily, &weekly, &monthly))
				for _, usage := range []float64{daily, weekly, monthly} {
					require.InDelta(t, 1.33333333, usage, 1e-9)
				}
			} else {
				require.InDelta(t, 98.66666667, balance, 1e-9)
			}

			// Replays still return the original paid amount after the active capture disappears.
			for _, cmd := range cmds {
				cmd.RewardCapture = nil
			}
			outcomes, err = repo.ApplyBatch(ctx, cmds)
			require.NoError(t, err)
			for _, out := range outcomes {
				require.NoError(t, out.Err)
				require.False(t, out.Result.Applied)
			}
			require.NotNil(t, outcomes[0].Result.PaidCost)
			require.Zero(t, *outcomes[0].Result.PaidCost)
			require.NotNil(t, outcomes[1].Result.PaidCost)
			require.InDelta(t, 0.33333333, *outcomes[1].Result.PaidCost, 1e-9)
			_, err = integrationDB.Exec(`INSERT INTO usage_billing_dedup_archive (request_id,api_key_id,request_fingerprint,created_at)
				SELECT request_id,api_key_id,request_fingerprint,created_at FROM usage_billing_dedup WHERE api_key_id=$1`, key.ID)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, err := integrationDB.Exec(`DELETE FROM usage_billing_dedup_archive WHERE api_key_id=$1`, key.ID)
				require.NoError(t, err)
			})
			_, err = integrationDB.Exec(`DELETE FROM usage_billing_dedup WHERE api_key_id=$1`, key.ID)
			require.NoError(t, err)
			outcomes, err = repo.ApplyBatch(ctx, cmds)
			require.NoError(t, err)
			for _, out := range outcomes {
				require.NoError(t, out.Err)
				require.False(t, out.Result.Applied)
			}
			require.NotNil(t, outcomes[1].Result.PaidCost)
			require.InDelta(t, 0.33333333, *outcomes[1].Result.PaidCost, 1e-9)
			cmds[0].OutputTokens++
			cmds[0].RequestFingerprint = ""
			outcomes, err = repo.ApplyBatch(ctx, cmds)
			require.NoError(t, err)
			require.ErrorIs(t, outcomes[0].Err, service.ErrUsageBillingRequestConflict)
			require.Nil(t, outcomes[0].Result)
			var receipts int
			require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM reward_usage WHERE user_id=$1`, user.ID).Scan(&receipts))
			require.Equal(t, 2, receipts)
		})
	}
}

func TestUsageBillingBatchRewards_ConcurrentSingleAndBatchSubscriptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, user, group, key, tmpl := rewardFixture(t)
	cardID, err := store.Grant(ctx, user.ID, tmpl.ID, user.ID, uuid.NewString(), "concurrent batch")
	require.NoError(t, err)
	require.NoError(t, store.Activate(ctx, user.ID, cardID))
	capture, err := store.Capture(ctx, user.ID, group.ID, "luna", true)
	require.NoError(t, err)
	require.NotNil(t, capture)
	sub := mustCreateSubscription(t, testEntClient(t), &service.UserSubscription{UserID: user.ID, GroupID: group.ID})
	t.Cleanup(func() {
		_, err := integrationDB.Exec(`DELETE FROM user_subscriptions WHERE id=$1`, sub.ID)
		require.NoError(t, err)
	})
	repo := NewUsageBillingRepository(testEntClient(t), integrationDB)
	command := func() *service.UsageBillingCommand {
		cmd := rewardCmd(user, key, capture, uuid.NewString(), 100)
		cmd.SubscriptionID = &sub.ID
		cmd.SubscriptionCost, cmd.BalanceCost = cmd.BalanceCost, 0
		return cmd
	}
	start, errs := make(chan struct{}), make(chan error, 2)
	for _, batch := range []bool{false, true} {
		go func(batch bool) {
			<-start
			for range 10 {
				if batch {
					outcomes, err := repo.ApplyBatch(ctx, []*service.UsageBillingCommand{command(), command()})
					if err != nil {
						errs <- err
						return
					}
					for _, out := range outcomes {
						if out.Err != nil {
							errs <- out.Err
							return
						}
						if out.Result == nil || !out.Result.Applied {
							errs <- fmt.Errorf("batch command was not applied")
							return
						}
					}
				} else {
					if _, err := repo.Apply(ctx, command()); err != nil {
						errs <- err
						return
					}
				}
			}
			errs <- nil
		}(batch)
	}
	close(start)
	for range 2 {
		require.NoError(t, <-errs)
	}
	var used float64
	var left int64
	var receipts int
	require.NoError(t, integrationDB.QueryRow(`SELECT daily_usage_usd FROM user_subscriptions WHERE id=$1`, sub.ID).Scan(&used))
	require.Equal(t, 20.0, used)
	require.NoError(t, integrationDB.QueryRow(`SELECT quota_used FROM api_keys WHERE id=$1`, key.ID).Scan(&used))
	require.Equal(t, 20.0, used)
	require.NoError(t, integrationDB.QueryRow(`SELECT remaining_tokens FROM reward_cards WHERE id=$1`, cardID).Scan(&left))
	require.Zero(t, left)
	require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM reward_usage WHERE user_id=$1`, user.ID).Scan(&receipts))
	require.Equal(t, 30, receipts)
}

func TestUsageBillingBatchRewards_RollsBackCardAndReceiptOnError(t *testing.T) {
	ctx := context.Background()
	store, user, group, key, tmpl := rewardFixture(t)
	cardID, err := store.Grant(ctx, user.ID, tmpl.ID, user.ID, uuid.NewString(), "batch rollback")
	require.NoError(t, err)
	require.NoError(t, store.Activate(ctx, user.ID, cardID))
	capture, err := store.Capture(ctx, user.ID, group.ID, "luna", true)
	require.NoError(t, err)
	require.NotNil(t, capture)
	invalid := *capture
	invalid.UserID++
	cmds := []*service.UsageBillingCommand{
		rewardCmd(user, key, capture, uuid.NewString(), 600),
		rewardCmd(user, key, &invalid, uuid.NewString(), 600),
	}
	_, err = NewUsageBillingRepository(testEntClient(t), integrationDB).ApplyBatch(ctx, cmds)
	require.Error(t, err)
	var left int64
	var balance float64
	var receipts, dedup int
	require.NoError(t, integrationDB.QueryRow(`SELECT remaining_tokens FROM reward_cards WHERE id=$1`, cardID).Scan(&left))
	require.NoError(t, integrationDB.QueryRow(`SELECT balance FROM users WHERE id=$1`, user.ID).Scan(&balance))
	require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM reward_usage WHERE user_id=$1`, user.ID).Scan(&receipts))
	require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM usage_billing_dedup WHERE api_key_id=$1`, key.ID).Scan(&dedup))
	require.Equal(t, int64(1000), left)
	require.Equal(t, 100.0, balance)
	require.Zero(t, receipts)
	require.Zero(t, dedup)
}
