//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/stretchr/testify/require"
)

func TestRewardsRedemptionReceiptSnapshotAndRollback(t *testing.T) {
	ctx := context.Background()
	s, u, _, _, tmpl := rewardFixture(t)
	codes, err := s.Codes(ctx, tmpl.ID, 1, 30, u.ID)
	require.NoError(t, err)
	var codeID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT id FROM redeem_codes WHERE code=$1", codes[0]).Scan(&codeID))
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM reward_redemptions WHERE code_id=$1", codeID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM redeem_codes WHERE id=$1", codeID)
		require.NoError(t, err)
	})
	original := tmpl
	tmpl.Name, tmpl.Tokens, tmpl.Model = "Edited later", 999999, "different-model"
	_, err = s.SaveTemplate(ctx, tmpl, u.ID)
	require.NoError(t, err)
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	id, err := rewards.Redeem(ctx, tx, u.ID, codeID)
	require.NoError(t, err)
	receipt, err := rewards.RedemptionReceipt(ctx, tx, u.ID, id)
	require.NoError(t, err)
	require.Equal(t, original, receipt.Template)
	require.Equal(t, original.Tokens, receipt.Remaining)
	require.Equal(t, "unused", receipt.Status)
	require.Nil(t, receipt.Activated)
	_, err = rewards.RedemptionReceipt(ctx, tx, u.ID+1000000, id)
	require.Error(t, err)
	require.NoError(t, tx.Rollback())
	var count int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM reward_cards WHERE source_key=$1", fmt.Sprintf("code:%d", codeID)).Scan(&count))
	require.Zero(t, count)
	tx, err = integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	id, err = rewards.Redeem(ctx, tx, u.ID, codeID)
	require.NoError(t, err)
	receipt, err = rewards.RedemptionReceipt(ctx, tx, u.ID, id)
	require.NoError(t, err)
	require.Equal(t, original, receipt.Template)
	retryID, err := rewards.Redeem(ctx, tx, u.ID, codeID)
	require.NoError(t, err)
	require.Equal(t, id, retryID)
	require.NoError(t, tx.Commit())
	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id=$1", u.ID).Scan(&balance))
	require.Equal(t, float64(100), balance)
}
