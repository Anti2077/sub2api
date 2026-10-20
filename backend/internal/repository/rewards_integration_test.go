//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/rewards"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func rewardFixture(t *testing.T) (*rewards.Store, *service.User, *service.Group, *service.APIKey, rewards.Template) {
	t.Helper()
	ctx := context.Background()
	client := testEntClient(t)
	u := mustCreateUser(t, client, &service.User{Email: uuid.NewString() + "@reward.test", PasswordHash: "hash", Balance: 100})
	g := mustCreateGroup(t, client, &service.Group{Name: "reward-" + uuid.NewString(), Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive})
	k := mustCreateApiKey(t, client, &service.APIKey{UserID: u.ID, GroupID: &g.ID, Key: "sk-" + uuid.NewString(), Status: service.StatusActive})
	s := rewards.New(integrationDB)
	// These fixtures commit transactions, so remove their assets before the
	// legacy suites delete users. Financial foreign keys intentionally restrict
	// deletion; do not weaken the production schema to accommodate tests.
	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM farm_orders WHERE user_id=$1`,
			`DELETE FROM farm_steals WHERE thief_id=$1 OR owner_id=$1`,
			`DELETE FROM farm_actions WHERE user_id=$1`,
			`DELETE FROM farm_ledger WHERE user_id=$1`,
			`DELETE FROM farm_plots WHERE user_id=$1`,
			`DELETE FROM farm_users WHERE user_id=$1`,
			`DELETE FROM reward_usage WHERE user_id=$1`,
			`DELETE FROM reward_events WHERE user_id=$1 OR actor_id=$1`,
			`DELETE FROM reward_cards WHERE user_id=$1`,
			`DELETE FROM usage_billing_dedup WHERE api_key_id IN (SELECT id FROM api_keys WHERE user_id=$1)`,
			`DELETE FROM api_keys WHERE user_id=$1`,
			`DELETE FROM users WHERE id=$1`,
		} {
			_, err := integrationDB.ExecContext(ctx, query, u.ID)
			require.NoError(t, err, query)
		}
	})
	cfg, e := s.Config(ctx, integrationDB)
	require.NoError(t, e)
	originalConfig := rewards.JSON(cfg)
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, `UPDATE reward_settings SET config=$1::jsonb,version=version+1 WHERE id=TRUE`, originalConfig)
		require.NoError(t, err)
	})
	cfg.Enabled = true
	cfg.FarmEnabled = true
	cfg.Maintenance = false
	cfg.ActivationEnabled = true
	cfg.PlantingEnabled = true
	cfg.StealingEnabled = true
	cfg.ExchangeEnabled = true
	cfg.AllowedUserIDs = nil
	cfg.StartsAt = nil
	cfg.EndsAt = nil
	_, e = s.SaveConfig(ctx, cfg, u.ID)
	require.NoError(t, e)
	tmpl, e := s.SaveTemplate(ctx, rewards.Template{Name: "token", Kind: "model_tokens", Model: "luna", GroupIDs: []int64{g.ID}, Tokens: 1000, Hours: 720, ClaimDays: 30, Published: true}, u.ID)
	require.NoError(t, e)
	return s, u, g, k, tmpl
}
func rewardCmd(u *service.User, k *service.APIKey, c *rewards.Capture, id string, tokens int) *service.UsageBillingCommand {
	return &service.UsageBillingCommand{RequestID: id, UserID: u.ID, APIKeyID: k.ID, Model: "luna", InputTokens: tokens, BalanceCost: 1, APIKeyQuotaCost: 1, RewardCapture: c, RewardTextCost: 1, RewardBaseRate: 1}
}
func TestRewardsAtomicConcurrentAndRetry(t *testing.T) {
	ctx := context.Background()
	s, u, g, k, tmpl := rewardFixture(t)
	id, e := s.Grant(ctx, u.ID, tmpl.ID, u.ID, uuid.NewString(), "test")
	require.NoError(t, e)
	require.NoError(t, s.Activate(ctx, u.ID, id))
	c, e := s.Capture(ctx, u.ID, g.ID, "luna", true)
	require.NoError(t, e)
	repo := NewUsageBillingRepository(testEntClient(t), integrationDB)
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repo.Apply(ctx, rewardCmd(u, k, c, fmt.Sprintf("reward-%d-%d", id, i), 100))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	var balance, quota float64
	var remaining int64
	require.NoError(t, integrationDB.QueryRow(`SELECT balance FROM users WHERE id=$1`, u.ID).Scan(&balance))
	require.NoError(t, integrationDB.QueryRow(`SELECT quota_used FROM api_keys WHERE id=$1`, k.ID).Scan(&quota))
	require.NoError(t, integrationDB.QueryRow(`SELECT remaining_tokens FROM reward_cards WHERE id=$1`, id).Scan(&remaining))
	require.Equal(t, float64(70), balance)
	require.Equal(t, float64(30), quota)
	require.Zero(t, remaining)
	var original, discount, paid float64
	var covered int64
	require.NoError(t, integrationDB.QueryRow(`SELECT SUM(original_cost),SUM(discount),SUM(paid_cost),SUM(covered_tokens) FROM reward_usage WHERE user_id=$1`, u.ID).Scan(&original, &discount, &paid, &covered))
	require.Equal(t, float64(40), original)
	require.Equal(t, float64(10), discount)
	require.Equal(t, float64(30), paid)
	require.Equal(t, int64(1000), covered)
	r, e := repo.Apply(ctx, rewardCmd(u, k, c, fmt.Sprintf("reward-%d-%d", id, 0), 100))
	require.NoError(t, e)
	require.False(t, r.Applied)
	require.NotNil(t, r.PaidCost)
	var count int
	require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM reward_usage WHERE user_id=$1`, u.ID).Scan(&count))
	require.Equal(t, 40, count)
}
func TestRewardsPartialSpecialFeesAndRollback(t *testing.T) {
	ctx := context.Background()
	s, u, g, k, tmpl := rewardFixture(t)
	tmpl.Tokens = 25
	tmpl.ID = 0
	tmpl, e := s.SaveTemplate(ctx, tmpl, u.ID)
	require.NoError(t, e)
	id, e := s.Grant(ctx, u.ID, tmpl.ID, u.ID, uuid.NewString(), "test")
	require.NoError(t, e)
	require.NoError(t, s.Activate(ctx, u.ID, id))
	c, e := s.Capture(ctx, u.ID, g.ID, "luna", true)
	require.NoError(t, e)
	cmd := rewardCmd(u, k, c, uuid.NewString(), 100)
	cmd.BalanceCost = 2
	cmd.RewardTextCost = 1
	repo := NewUsageBillingRepository(testEntClient(t), integrationDB)
	r, e := repo.Apply(ctx, cmd)
	require.NoError(t, e)
	require.Equal(t, 1.75, *r.PaidCost)
	var balance float64
	require.NoError(t, integrationDB.QueryRow(`SELECT balance FROM users WHERE id=$1`, u.ID).Scan(&balance))
	require.Equal(t, 98.25, balance)
	// An invalid account quota FK-equivalent error occurs after card allocation: the entire tx must roll back.
	_, e = integrationDB.Exec(`UPDATE reward_cards SET remaining_tokens=25,status='active' WHERE id=$1`, id)
	require.NoError(t, e)
	bad := rewardCmd(u, k, c, uuid.NewString(), 100)
	bad.AccountID = -999999
	bad.AccountType = service.AccountTypeAPIKey
	bad.AccountQuotaCost = 2
	_, e = repo.Apply(ctx, bad)
	require.Error(t, e)
	var left int64
	require.NoError(t, integrationDB.QueryRow(`SELECT remaining_tokens FROM reward_cards WHERE id=$1`, id).Scan(&left))
	require.Equal(t, int64(25), left)
	var dedup int
	require.NoError(t, integrationDB.QueryRow(`SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id=$1`, bad.RequestID).Scan(&dedup))
	require.Zero(t, dedup)
}
func TestFarmHarvestStealCompetitionAndShopIdempotency(t *testing.T) {
	ctx := context.Background()
	s, u, _, _, tmpl := rewardFixture(t)
	now := time.Now().UTC()
	s.Now = func() time.Time { return now }
	a := rewards.FarmAction{Key: uuid.NewString(), Kind: "plant", Slot: 1, CropID: "sprout"}
	_, e := s.Action(ctx, u.ID, a)
	require.NoError(t, e)
	_, e = s.Action(ctx, u.ID, rewards.FarmAction{Key: uuid.NewString(), Kind: "harvest", Slot: 1})
	require.Error(t, e)
	now = now.Add(2 * time.Hour)
	var wg sync.WaitGroup
	success := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Action(ctx, u.ID, rewards.FarmAction{Key: uuid.NewString(), Kind: "harvest", Slot: 1})
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
	f, e := s.Farm(ctx, u.ID, u.ID)
	require.NoError(t, e)
	require.Equal(t, int64(10), f.Crystals)
	shop, e := s.SaveShop(ctx, rewards.Shop{TemplateID: tmpl.ID, Price: 7, Stock: 1, DailyLimit: 1, TotalLimit: 1, Published: true}, u.ID)
	require.NoError(t, e)
	buy := rewards.FarmAction{Key: uuid.NewString(), Kind: "buy", ShopID: shop.ID}
	first, e := s.Action(ctx, u.ID, buy)
	require.NoError(t, e)
	second, e := s.Action(ctx, u.ID, buy)
	require.NoError(t, e)
	var x, y any
	require.NoError(t, json.Unmarshal(first, &x))
	require.NoError(t, json.Unmarshal(second, &y))
	require.Equal(t, x, y)
	f, e = s.Farm(ctx, u.ID, u.ID)
	require.NoError(t, e)
	require.Equal(t, int64(3), f.Crystals)
	_, e = s.Action(ctx, u.ID, rewards.FarmAction{Key: uuid.NewString(), Kind: "buy", ShopID: shop.ID})
	require.Error(t, e)
}
