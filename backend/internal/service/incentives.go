package service

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// IncentiveConfig is one independently saved campaign. Version guards concurrent editors.
type IncentiveConfig struct {
	Kind            string              `json:"kind"`
	Version         int64               `json:"version"`
	Enabled         bool                `json:"enabled"`
	Timezone        string              `json:"timezone"`
	Period          string              `json:"period"`
	ResetChances    bool                `json:"reset_chances"`
	GroupIDs        []int64             `json:"group_ids"`
	ExcludedUserIDs []int64             `json:"excluded_user_ids"`
	ExcludedModels  []string            `json:"excluded_models"`
	ExcludeAdmins   bool                `json:"exclude_admins"`
	SpendThreshold  float64             `json:"spend_threshold"`
	RateDecrease    float64             `json:"rate_decrease"`
	MinimumRate     float64             `json:"minimum_rate"`
	MaxChances      int64               `json:"max_chances"`
	Prizes          []DailyLotteryPrize `json:"prizes"`
}
type IncentiveGroupRate struct {
	GroupID     int64   `json:"group_id"`
	Name        string  `json:"name"`
	BaseRate    float64 `json:"base_rate"`
	CurrentRate float64 `json:"current_rate"`
}
type IncentiveStatus struct {
	Kind          string                  `json:"kind"`
	Enabled       bool                    `json:"enabled"`
	Eligible      bool                    `json:"eligible"`
	PeriodID      int64                   `json:"period_id"`
	StartsAt      time.Time               `json:"starts_at"`
	EndsAt        time.Time               `json:"ends_at"`
	Timezone      string                  `json:"timezone"`
	Spend         float64                 `json:"spend"`
	PersonalSpend float64                 `json:"personal_spend"`
	NextThreshold float64                 `json:"next_threshold"`
	Threshold     float64                 `json:"threshold"`
	Earned        int64                   `json:"earned"`
	Used          int64                   `json:"used"`
	Available     int64                   `json:"available"`
	Groups        []IncentiveGroupRate    `json:"groups"`
	Prizes        []DailyLotteryPrizeView `json:"prizes"`
}
type IncentiveService struct {
	client      *dbent.Client
	users       UserRepository
	groups      GroupRepository
	lottery     *DailyLotteryService
	invalidator APIKeyAuthCacheInvalidator
}

func NewIncentiveService(client *dbent.Client, users UserRepository, groups GroupRepository, lottery *DailyLotteryService, invalidator APIKeyAuthCacheInvalidator) *IncentiveService {
	return &IncentiveService{client: client, users: users, groups: groups, lottery: lottery, invalidator: invalidator}
}
func DefaultIncentiveConfig(kind string) IncentiveConfig {
	return IncentiveConfig{Kind: kind, Timezone: timezone.Location().String(), Period: "weekly", ResetChances: true, GroupIDs: []int64{}, ExcludedUserIDs: []int64{}, ExcludedModels: []string{}, ExcludeAdmins: true, SpendThreshold: 50, RateDecrease: 0.01, MinimumRate: 0.35, Prizes: DefaultDailyLotteryConfig().Prizes}
}
func validateIncentiveConfig(c IncentiveConfig) error {
	bad := func() error { return infraerrors.BadRequest("INCENTIVE_INVALID_CONFIG", "invalid incentive rules") }
	if c.Kind != "global_rate" && c.Kind != "lottery" {
		return bad()
	}
	if c.Period != "weekly" || !c.ResetChances {
		return bad()
	}
	for _, v := range []float64{c.SpendThreshold, c.RateDecrease, c.MinimumRate} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > 1000000 {
			return bad()
		}
	}
	if c.MaxChances < 0 || c.MaxChances > 1000000 || (c.Enabled && len(c.GroupIDs) == 0) {
		return bad()
	}
	for _, ids := range [][]int64{c.GroupIDs, c.ExcludedUserIDs} {
		seen := map[int64]bool{}
		for _, id := range ids {
			if id <= 0 || seen[id] {
				return bad()
			}
			seen[id] = true
		}
	}
	if c.Kind == "lottery" {
		return validateDailyLotteryConfig(DailyLotteryConfig{Prizes: c.Prizes})
	}
	return nil
}
func (s *IncentiveService) Configs(ctx context.Context) ([]IncentiveConfig, error) {
	result := []IncentiveConfig{DefaultIncentiveConfig("global_rate"), DefaultIncentiveConfig("lottery")}
	// Reuse the operator's existing prize definitions when first configuring consumption rewards.
	legacy, err := s.lottery.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	result[1].Prizes = legacy.Prizes
	rows, err := s.client.QueryContext(ctx, "SELECT kind,version,config FROM incentive_campaigns ORDER BY kind")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var kind string
		var version int64
		var raw []byte
		if err = rows.Scan(&kind, &version, &raw); err != nil {
			return nil, err
		}
		for i := range result {
			if result[i].Kind == kind {
				if err = json.Unmarshal(raw, &result[i]); err != nil {
					return nil, err
				}
				result[i].Version = version
			}
		}
	}
	return result, rows.Err()
}
func (s *IncentiveService) Save(ctx context.Context, c IncentiveConfig, actor int64) (IncentiveConfig, error) {
	c.Timezone = timezone.Location().String()
	if err := validateIncentiveConfig(c); err != nil {
		return c, err
	}
	for _, id := range c.GroupIDs {
		g, err := s.groups.GetByID(ctx, id)
		if err != nil {
			return c, err
		}
		if c.Kind == "global_rate" && c.MinimumRate > g.RateMultiplier {
			return c, infraerrors.BadRequest("INCENTIVE_INVALID_FLOOR", "minimum rate exceeds a group's base rate")
		}
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return c, err
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize initial insert as well as updates with usage accounting and resets.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(783219)"); err != nil {
		return c, err
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return c, err
	}
	rows, err := tx.QueryContext(ctx, `INSERT INTO incentive_campaigns(kind,version,config) SELECT $1,1,$2::jsonb WHERE $3=0
 ON CONFLICT(kind) DO NOTHING RETURNING version`, c.Kind, string(raw), c.Version)
	if err != nil {
		return c, err
	}
	inserted := rows.Next()
	_ = rows.Close()
	if !inserted {
		rows, err = tx.QueryContext(ctx, `UPDATE incentive_campaigns SET version=version+1,config=$2::jsonb,updated_at=clock_timestamp() WHERE kind=$1 AND version=$3 RETURNING version`, c.Kind, string(raw), c.Version)
		if err != nil {
			return c, err
		}
		if !rows.Next() {
			_ = rows.Close()
			return c, infraerrors.Conflict("INCENTIVE_CONFIG_CHANGED", "reload settings before saving")
		}
		err = rows.Scan(&c.Version)
		_ = rows.Close()
		if err != nil {
			return c, err
		}
	} else {
		c.Version = 1
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO incentive_config_history(kind,version,config,actor_id) VALUES($1,$2,$3::jsonb,$4)`, c.Kind, c.Version, string(raw), actor); err != nil {
		return c, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT incentive_ensure_period($1,clock_timestamp())", c.Kind)
	if err != nil {
		return c, err
	}
	var period int64
	if rows.Next() {
		err = rows.Scan(&period)
	}
	_ = rows.Close()
	if err != nil {
		return c, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE incentive_campaign_periods SET rule_spend=spend,rule_decrease=decrease WHERE id=$1`, period); err != nil {
		return c, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE incentive_user_progress SET rule_spend=spend,rule_earned=earned WHERE period_id=$1`, period); err != nil {
		return c, err
	}
	// Snapshot every config version, but keep historical rule segments immutable in the audit log.
	// New groups get a snapshot; existing group snapshots never change when settings are edited.
	if _, err = tx.ExecContext(ctx, `INSERT INTO incentive_group_rate_snapshots(period_id,group_id,name,base_rate)
 SELECT $1,id,name,rate_multiplier FROM groups WHERE id IN (SELECT jsonb_array_elements_text($2::jsonb->'group_ids')::bigint) ON CONFLICT DO NOTHING`, period, string(raw)); err != nil {
		return c, err
	}
	return c, tx.Commit()
}
func (s *IncentiveService) Status(ctx context.Context, userID int64) ([]IncentiveStatus, error) {
	configs, err := s.Configs(ctx)
	if err != nil {
		return nil, err
	}
	var user *User
	if userID > 0 {
		user, err = s.users.GetByID(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	out := []IncentiveStatus{}
	for _, c := range configs {
		start := timezone.StartOfWeek(timezone.Now())
		v := IncentiveStatus{Kind: c.Kind, Enabled: c.Enabled, Eligible: user == nil || (!slices.Contains(c.ExcludedUserIDs, userID) && (!c.ExcludeAdmins || user.Role != "admin")), StartsAt: start, EndsAt: start.AddDate(0, 0, 7), Timezone: timezone.Name(), Groups: []IncentiveGroupRate{}, Prizes: []DailyLotteryPrizeView{}, Threshold: c.SpendThreshold, NextThreshold: c.SpendThreshold}
		if c.Kind == "lottery" {
			v.Prizes = dailyLotteryPrizeViews(c.Prizes)
		}
		if c.Version == 0 {
			out = append(out, v)
			continue
		}
		var ruleSpend float64
		periodRows, e := s.client.QueryContext(ctx, `SELECT incentive_ensure_period($1,clock_timestamp())`, c.Kind)
		if e != nil {
			return nil, e
		}
		if periodRows.Next() {
			e = periodRows.Scan(&v.PeriodID)
		}
		_ = periodRows.Close()
		if e != nil {
			return nil, e
		}
		rows, e := s.client.QueryContext(ctx, `SELECT id,starts_at,ends_at,spend,rule_spend FROM incentive_campaign_periods WHERE id=$1`, v.PeriodID)
		if e != nil {
			return nil, e
		}
		if rows.Next() {
			e = rows.Scan(&v.PeriodID, &v.StartsAt, &v.EndsAt, &v.Spend, &ruleSpend)
		}
		_ = rows.Close()
		if e != nil {
			return nil, e
		}
		var personalRuleSpend float64
		rows, e = s.client.QueryContext(ctx, `SELECT spend,earned,used,rule_spend FROM incentive_user_progress WHERE period_id=$1 AND user_id=$2`, v.PeriodID, userID)
		if e != nil {
			return nil, e
		}
		if rows.Next() {
			e = rows.Scan(&v.PersonalSpend, &v.Earned, &v.Used, &personalRuleSpend)
		}
		_ = rows.Close()
		if e != nil {
			return nil, e
		}
		v.Available = v.Earned - v.Used
		spend := v.Spend - ruleSpend
		if c.Kind == "lottery" {
			spend = v.PersonalSpend - personalRuleSpend
		}
		v.NextThreshold = c.SpendThreshold - math.Mod(spend, c.SpendThreshold)
		rows, e = s.client.QueryContext(ctx, `SELECT r.group_id,r.name,r.base_rate,COALESCE((SELECT decrease FROM incentive_rate_history WHERE period_id=$1 ORDER BY id DESC LIMIT 1),0),g.rate_multiplier
 FROM incentive_group_rate_snapshots r JOIN groups g ON g.id=r.group_id WHERE r.period_id=$1 ORDER BY r.group_id`, v.PeriodID)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var g IncentiveGroupRate
			var decrease, current float64
			if e = rows.Scan(&g.GroupID, &g.Name, &g.BaseRate, &decrease, &current); e != nil {
				break
			}
			if !slices.Contains(c.GroupIDs, g.GroupID) {
				continue
			}
			g.CurrentRate = current
			if c.Enabled && c.Kind == "global_rate" && current == g.BaseRate {
				g.CurrentRate = math.Min(current, math.Max(c.MinimumRate, g.BaseRate-decrease))
			}
			v.Groups = append(v.Groups, g)
		}
		if e == nil {
			e = rows.Err()
		}
		_ = rows.Close()
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}

// ApplyRate deliberately leaves explicit user rates and separately priced image/video rates alone.
// Query historical decrease at PricingAt so an in-flight request cannot receive a later tier.
func (s *IncentiveService) ApplyRate(ctx context.Context, user *User, group *Group, model string, base float64, at time.Time) float64 {
	if s == nil || user == nil || group == nil || base != group.RateMultiplier {
		return base
	}
	rows, err := s.client.QueryContext(ctx, `SELECT r.base_rate,(SELECT config FROM incentive_config_history WHERE kind='global_rate' AND action='config' AND created_at<=$3 ORDER BY id DESC LIMIT 1),COALESCE((SELECT decrease FROM incentive_rate_history h WHERE h.period_id=p.id AND h.created_at<=$3 ORDER BY h.id DESC LIMIT 1),0)
 FROM incentive_campaigns c JOIN incentive_campaign_periods p ON p.kind=c.kind
 JOIN incentive_group_rate_snapshots r ON r.period_id=p.id AND r.group_id=$1
 WHERE c.kind='global_rate' AND (p.closed_at IS NULL OR p.closed_at>$3) AND p.starts_at<=$3 AND p.ends_at>$3
 AND p.activated_at<=$3
 AND NOT EXISTS(SELECT 1 FROM user_group_rate_multipliers WHERE user_id=$2 AND group_id=$1 AND rate_multiplier IS NOT NULL)`, group.ID, user.ID, at)
	if err != nil {
		return base
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return base
	}
	var snapshot, decrease float64
	var raw []byte
	if rows.Scan(&snapshot, &raw, &decrease) != nil {
		return base
	}
	var c IncentiveConfig
	if json.Unmarshal(raw, &c) != nil {
		return base
	}
	if !c.Enabled || snapshot != group.RateMultiplier || !slices.Contains(c.GroupIDs, group.ID) || slices.Contains(c.ExcludedUserIDs, user.ID) || slices.Contains(c.ExcludedModels, model) || (c.ExcludeAdmins && user.Role == "admin") {
		return base
	}
	return math.Min(base, math.Max(c.MinimumRate, snapshot-decrease))
}
func (s *IncentiveService) History(ctx context.Context, userID int64) (map[string]any, error) {
	queries := map[string]string{"rewards": `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT id,period_id,user_id,prize,reward_amount,created_at FROM incentive_reward_ledger WHERE ($1::bigint=0 OR user_id=$1) ORDER BY id DESC LIMIT 100) x`, "chances": `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT * FROM incentive_lottery_chance_ledger WHERE ($1::bigint=0 OR user_id=$1) ORDER BY id DESC LIMIT 100) x`}
	if userID == 0 {
		queries["periods"] = `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT * FROM incentive_campaign_periods WHERE $1::bigint=0 ORDER BY id DESC LIMIT 100) x`
		queries["rates"] = `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT * FROM incentive_rate_history WHERE $1::bigint=0 ORDER BY id DESC LIMIT 100) x`
		queries["config_changes"] = `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT * FROM incentive_config_history WHERE $1::bigint=0 ORDER BY id DESC LIMIT 100) x`
	}
	out := map[string]any{}
	for key, q := range queries {
		rows, err := s.client.QueryContext(ctx, q, userID)
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if rows.Next() {
			err = rows.Scan(&raw)
		}
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
		out[key] = raw
	}
	return out, nil
}
func (s *IncentiveService) Reset(ctx context.Context, id, actor int64) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(783219)"); err != nil {
		return err
	}
	// Same campaign-before-period lock order as accounting and draw.
	rows, err := tx.QueryContext(ctx, `SELECT c.kind FROM incentive_campaigns c JOIN incentive_campaign_periods p ON p.kind=c.kind WHERE p.id=$1 FOR UPDATE OF c`, id)
	if err != nil {
		return err
	}
	var kind string
	if rows.Next() {
		err = rows.Scan(&kind)
	}
	_ = rows.Close()
	if err != nil {
		return err
	}
	if kind == "" {
		return infraerrors.NotFound("INCENTIVE_PERIOD_NOT_FOUND", "period not found")
	}
	changed, err := tx.ExecContext(ctx, `UPDATE incentive_campaign_periods SET closed_at=clock_timestamp() WHERE id=$1 AND closed_at IS NULL`, id)
	if err != nil {
		return err
	}
	count, err := changed.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return tx.Commit()
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO incentive_config_history(kind,version,config,actor_id,action) SELECT kind,version,config,$2,'reset:'||$3::text FROM incentive_campaigns WHERE kind=$1`, kind, actor, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *IncentiveService) Draw(ctx context.Context, userID int64, key string) (map[string]any, error) {
	if len(key) < 8 || len(key) > 100 {
		return nil, infraerrors.BadRequest("INCENTIVE_REQUEST_KEY", "a stable request key is required")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT config FROM incentive_campaigns WHERE kind='lottery' FOR UPDATE`)
	if err != nil {
		return nil, err
	}
	var raw []byte
	if rows.Next() {
		err = rows.Scan(&raw)
	}
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	var c IncentiveConfig
	if len(raw) == 0 {
		return nil, ErrDailyLotteryDisabled
	}
	if err = json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	// A retry returns its original receipt, even if the period has since expired.
	rows, err = tx.QueryContext(ctx, `SELECT prize,reward_amount FROM incentive_reward_ledger WHERE user_id=$1 AND request_key=$2`, userID, key)
	if err != nil {
		return nil, err
	}
	var prizeRaw json.RawMessage
	var amount float64
	if rows.Next() {
		err = rows.Scan(&prizeRaw, &amount)
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
		return map[string]any{"prize": prizeRaw, "reward_amount": amount}, nil
	}
	_ = rows.Close()
	user, err := s.users.GetByID(dbent.NewTxContext(ctx, tx), userID)
	if err != nil {
		return nil, err
	}
	if !c.Enabled || slices.Contains(c.ExcludedUserIDs, userID) || (c.ExcludeAdmins && user.Role == "admin") {
		return nil, ErrDailyLotteryDisabled
	}
	rows, err = tx.QueryContext(ctx, `UPDATE incentive_user_progress SET used=used+1 WHERE user_id=$1 AND period_id=incentive_ensure_period('lottery',clock_timestamp()) AND used<earned RETURNING period_id`, userID)
	if err != nil {
		return nil, err
	}
	var period int64
	if rows.Next() {
		err = rows.Scan(&period)
	}
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	if period == 0 {
		return nil, infraerrors.Conflict("INCENTIVE_NO_CHANCES", "no available draw chances")
	}
	prize, err := s.lottery.selectPrize(c.Prizes)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(prize)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO incentive_reward_ledger(period_id,user_id,request_key,prize,reward_amount) VALUES($1,$2,$3,$4::jsonb,$5)`, period, userID, key, string(encoded), prize.RewardAmount); err != nil {
		return nil, err
	}
	if prize.RewardAmount > 0 {
		if _, err = s.users.AdjustBalance(dbent.NewTxContext(ctx, tx), userID, prize.RewardAmount); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if s.invalidator != nil {
		s.invalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	return map[string]any{"prize": prize, "reward_amount": prize.RewardAmount}, nil
}

func (s *SettingService) SetIncentiveService(i *IncentiveService) { s.incentives = i }
