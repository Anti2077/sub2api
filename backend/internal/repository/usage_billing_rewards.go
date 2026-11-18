package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// lockUsageBillingRewardRows keeps the upstream subscription -> user lock order
// when a reward must lock its owner before consuming cards. Lock all affected
// subscriptions first, including ordinary charges sharing the same batch.
func lockUsageBillingRewardRows(ctx context.Context, tx *sql.Tx, cmds []*service.UsageBillingCommand, applied []bool) error {
	needsRewards := false
	for i, cmd := range cmds {
		if applied[i] && cmd.RewardCapture != nil {
			needsRewards = true
			break
		}
	}
	if !needsRewards {
		return nil
	}
	_, ids := groupUsageBillingAmounts(cmds, applied, func(cmd *service.UsageBillingCommand) (int64, float64, bool) {
		if cmd.SubscriptionID == nil {
			return 0, 0, false
		}
		return *cmd.SubscriptionID, cmd.SubscriptionCost, true
	})
	if len(ids) > 0 {
		rows, err := tx.QueryContext(ctx, `SELECT id FROM user_subscriptions WHERE id = ANY($1::bigint[]) AND deleted_at IS NULL ORDER BY id FOR NO KEY UPDATE`, pq.Array(ids))
		if err != nil {
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	var lockedUser int64
	return tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, cmds[0].UserID).Scan(&lockedUser)
}

func readUsageBillingRewardReceipt(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	var paid float64
	err := tx.QueryRowContext(ctx, `SELECT paid_cost FROM reward_usage WHERE request_id=$1 AND api_key_id=$2`, cmd.RequestID, cmd.APIKeyID).Scan(&paid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	result.PaidCost = &paid
	return nil
}

// settleUsageBillingReward uses the same durable reward receipt for single and
// batch settlement, without changing the command or its idempotency fingerprint.
// The owner and subscription rows must already be locked by the caller.
func settleUsageBillingReward(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) (*service.UsageBillingCommand, error) {
	if cmd.RewardCapture == nil {
		return cmd, nil
	}
	c := cmd.RewardCapture
	if c.UserID != cmd.UserID {
		return nil, rewards.Bad("权益快照归属不匹配")
	}
	settled := *cmd
	original := cmd.BalanceCost + cmd.SubscriptionCost
	paid, err := rewards.Spend(ctx, tx, c, cmd.APIKeyID, cmd.RequestID, int64(cmd.InputTokens)+int64(cmd.OutputTokens)+int64(cmd.CacheCreationTokens)+int64(cmd.CacheReadTokens), original, cmd.RewardTextCost, cmd.RewardBaseRate)
	if err != nil {
		return nil, err
	}
	if cmd.SubscriptionID != nil {
		settled.SubscriptionCost = paid
		settled.BalanceCost = 0
	} else {
		settled.BalanceCost = paid
	}
	if cmd.APIKeyQuotaCost > 0 {
		settled.APIKeyQuotaCost = paid
	}
	if cmd.APIKeyRateLimitCost > 0 {
		settled.APIKeyRateLimitCost = paid
	}
	result.PaidCost = &paid
	return &settled, nil
}
