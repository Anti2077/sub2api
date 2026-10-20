//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestRewardsManagementCodeRetryAndStatus(t *testing.T) {
	ctx := context.Background()
	s, u, _, _, tmpl := rewardFixture(t)
	t.Cleanup(func() {
		_, e := integrationDB.ExecContext(ctx, "DELETE FROM reward_admin_batches WHERE actor_id=$1", u.ID)
		require.NoError(t, e)
		_, e = integrationDB.ExecContext(ctx, "DELETE FROM reward_redemptions WHERE template_id=$1", tmpl.ID)
		require.NoError(t, e)
	})
	key := uuid.NewString()
	var wg sync.WaitGroup
	codes := make([][]string, 12)
	errs := make([]error, 12)
	for i := range codes {
		wg.Add(1)
		go func(i int) { defer wg.Done(); codes[i], errs[i] = s.CodesWithKey(ctx, tmpl.ID, 3, 30, u.ID, key) }(i)
	}
	wg.Wait()
	for i := range codes {
		require.NoError(t, errs[i])
		require.Len(t, codes[i], 3)
		require.Equal(t, codes[0], codes[i])
	}
	var n int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM reward_redemptions WHERE template_id=$1", tmpl.ID).Scan(&n))
	require.Equal(t, 3, n)
	_, e := s.CodesWithKey(ctx, tmpl.ID, 4, 30, u.ID, key)
	require.Error(t, e)
	tmpl.Published = false
	tmpl, e = s.SaveTemplate(ctx, tmpl, u.ID)
	require.NoError(t, e)
	retry, e := rewards.New(integrationDB).CodesWithKey(ctx, tmpl.ID, 3, 30, u.ID, key)
	require.NoError(t, e)
	require.Equal(t, codes[0], retry)
	var id int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT id FROM redeem_codes WHERE code=$1", codes[0][0]).Scan(&id))
	require.NoError(t, s.SetCodeStatus(ctx, id, u.ID, "disabled"))
	require.NoError(t, s.SetCodeStatus(ctx, id, u.ID, "disabled"))
	require.NoError(t, s.SetCodeStatus(ctx, id, u.ID, "unused"))
	// Fail only this fixture's audit write, after the status UPDATE executes.
	// actor_id has no foreign key, so an invalid actor cannot simulate failure.
	faultName := fmt.Sprintf("reward_management_audit_fail_%d", u.ID)
	_, e = integrationDB.ExecContext(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
	BEGIN
		IF NEW.actor_id = %d AND NEW.kind = 'code_status_changed' THEN
			RAISE EXCEPTION 'management fixture audit failure';
		END IF;
		RETURN NEW;
	END $$`, faultName, u.ID))
	require.NoError(t, e)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON reward_events", faultName))
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", faultName))
		require.NoError(t, err)
	})
	_, e = integrationDB.ExecContext(ctx, fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON reward_events FOR EACH ROW EXECUTE FUNCTION %s()", faultName, faultName))
	require.NoError(t, e)
	var auditBefore, auditAfter int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM reward_events WHERE actor_id=$1 AND kind='code_status_changed'", u.ID).Scan(&auditBefore))
	require.ErrorContains(t, s.SetCodeStatus(ctx, id, u.ID, "disabled"), "management fixture audit failure")
	var unchanged string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT status FROM redeem_codes WHERE id=$1", id).Scan(&unchanged))
	require.Equal(t, "unused", unchanged)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM reward_events WHERE actor_id=$1 AND kind='code_status_changed'", u.ID).Scan(&auditAfter))
	require.Equal(t, auditBefore, auditAfter)
	_, e = integrationDB.ExecContext(ctx, fmt.Sprintf("DROP TRIGGER %s ON reward_events", faultName))
	require.NoError(t, e)
	require.NoError(t, s.SetCodeStatus(ctx, id, u.ID, "disabled"))
	require.NoError(t, s.SetCodeStatus(ctx, id, u.ID, "unused"))
	tx, e := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, e)
	card, e := rewards.Redeem(ctx, tx, u.ID, id)
	require.NoError(t, e)
	_, e = tx.ExecContext(ctx, "UPDATE redeem_codes SET status='used',used_by=$1,used_at=NOW() WHERE id=$2", u.ID, id)
	require.NoError(t, e)
	require.NoError(t, tx.Commit())
	require.Positive(t, card)
	require.Error(t, s.SetCodeStatus(ctx, id, u.ID, "unused"))
	require.Error(t, s.SetCodeStatus(ctx, id, u.ID, "disabled"))
	var second int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT id FROM redeem_codes WHERE code=$1", codes[0][1]).Scan(&second))
	_, e = integrationDB.ExecContext(ctx, "UPDATE redeem_codes SET expires_at=NOW()-INTERVAL '1 second' WHERE id=$1", second)
	require.NoError(t, e)
	require.Error(t, s.SetCodeStatus(ctx, second, u.ID, "unused"))
}
func TestRewardsManagementGrantReplayAndRevoke(t *testing.T) {
	ctx := context.Background()
	s, u, _, _, tmpl := rewardFixture(t)
	key := uuid.NewString()
	var wg sync.WaitGroup
	ids := make([]int64, 12)
	errs := make([]error, 12)
	for i := range ids {
		wg.Add(1)
		go func(i int) { defer wg.Done(); ids[i], errs[i] = s.Grant(ctx, u.ID, tmpl.ID, u.ID, key, "batch") }(i)
	}
	wg.Wait()
	for i := range ids {
		require.NoError(t, errs[i])
		require.Equal(t, ids[0], ids[i])
	}
	var n int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM reward_events WHERE actor_id=$1 AND kind='admin_grant'", u.ID).Scan(&n))
	require.Equal(t, 1, n)
	tmpl.Published = false
	_, e := s.SaveTemplate(ctx, tmpl, u.ID)
	require.NoError(t, e)
	id, e := rewards.New(integrationDB).Grant(ctx, u.ID, tmpl.ID, u.ID, key, "batch")
	require.NoError(t, e)
	require.Equal(t, ids[0], id)
	_, e = s.Grant(ctx, u.ID, tmpl.ID+1, u.ID, key, "batch")
	require.Error(t, e)
	require.NoError(t, s.Revoke(ctx, id, u.ID, "revoke"))
	require.NoError(t, s.Revoke(ctx, id, u.ID, "retry"))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM reward_events WHERE card_id=$1 AND kind='revoked'", id).Scan(&n))
	require.Equal(t, 1, n)
}
func TestRewardsManagementShopStatusPreservesInventoryAndSnapshot(t *testing.T) {
	ctx := context.Background()
	s, u, _, _, tmpl := rewardFixture(t)
	item, e := s.SaveShop(ctx, rewards.Shop{TemplateID: tmpl.ID, Price: 10, Stock: 3, DailyLimit: 1, TotalLimit: 3, Published: true}, u.ID)
	require.NoError(t, e)
	t.Cleanup(func() {
		_, e := integrationDB.ExecContext(ctx, "DELETE FROM farm_orders WHERE shop_id=$1", item.ID)
		require.NoError(t, e)
		_, e = integrationDB.ExecContext(ctx, "DELETE FROM farm_shop WHERE id=$1", item.ID)
		require.NoError(t, e)
	})
	_, e = s.Farm(ctx, u.ID, u.ID)
	require.NoError(t, e)
	_, e = integrationDB.ExecContext(ctx, "UPDATE farm_users SET crystals=100 WHERE user_id=$1", u.ID)
	require.NoError(t, e)
	tmpl.Tokens = 9000
	_, e = s.SaveTemplate(ctx, tmpl, u.ID)
	require.NoError(t, e)
	var wg sync.WaitGroup
	wg.Add(2)
	var buyErr, statusErr error
	go func() {
		defer wg.Done()
		_, buyErr = s.Action(ctx, u.ID, rewards.FarmAction{Key: uuid.NewString(), Kind: "buy", ShopID: item.ID})
	}()
	go func() { defer wg.Done(); _, statusErr = s.SetShopPublished(ctx, item.ID, item.Version, u.ID, false) }()
	wg.Wait()
	var stock int64
	var published bool
	var ver int64
	var raw []byte
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT stock,published,version,snapshot FROM farm_shop WHERE id=$1", item.ID).Scan(&stock, &published, &ver, &raw))
	if buyErr == nil {
		require.EqualValues(t, 2, stock)
		require.Error(t, statusErr)
	} else {
		require.NoError(t, statusErr)
		require.False(t, published)
		require.EqualValues(t, 3, stock)
	}
	var snap rewards.Template
	require.NoError(t, json.Unmarshal(raw, &snap))
	require.EqualValues(t, 1000, snap.Tokens)
	next, e := s.SetShopPublished(ctx, item.ID, ver, u.ID, true)
	require.NoError(t, e)
	_, e = s.SetShopPublished(ctx, item.ID, ver, u.ID, false)
	require.Error(t, e)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT stock,snapshot FROM farm_shop WHERE id=$1", item.ID).Scan(&stock, &raw))
	require.NoError(t, json.Unmarshal(raw, &snap))
	require.EqualValues(t, 1000, snap.Tokens)
	_, e = s.SetShopPublished(ctx, item.ID, next, u.ID, false)
	require.NoError(t, e)
}
