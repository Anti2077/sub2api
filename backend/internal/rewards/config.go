package rewards

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type Crop struct {
	ConfigVersion     int64  `json:"config_version"`
	ID                string `json:"id"`
	Name              string `json:"name"`
	Hours             int    `json:"hours"`
	Yield             int64  `json:"yield"`
	Enabled           bool   `json:"enabled"`
	ProtectionMinutes int    `json:"protection_minutes"`
	StealPercent      int    `json:"steal_percent"`
	MaxStolenPercent  int    `json:"max_stolen_percent"`
}
type Config struct {
	Version           int64      `json:"version"`
	Enabled           bool       `json:"enabled"`
	ActivationEnabled bool       `json:"activation_enabled"`
	FarmEnabled       bool       `json:"farm_enabled"`
	PlantingEnabled   bool       `json:"planting_enabled"`
	ExchangeEnabled   bool       `json:"exchange_enabled"`
	StealingEnabled   bool       `json:"stealing_enabled"`
	Maintenance       bool       `json:"maintenance"`
	Plots             int        `json:"plots"`
	ProtectionMinutes int        `json:"protection_minutes"`
	StealPercent      int        `json:"steal_percent"`
	MaxStolenPercent  int        `json:"max_stolen_percent"`
	DailySteals       int        `json:"daily_steals"`
	DailyPerOwner     int        `json:"daily_per_owner"`
	StartsAt          *time.Time `json:"starts_at"`
	EndsAt            *time.Time `json:"ends_at"`
	AllowedUserIDs    []int64    `json:"allowed_user_ids"`
	Crops             []Crop     `json:"crops"`
}

func (s *Store) Config(ctx context.Context, q Executor) (Config, error) {
	var c Config
	var b []byte
	var ver int64
	e := One(ctx, q, `SELECT version,config FROM reward_settings WHERE id=TRUE`, nil, &ver, &b)
	if e != nil {
		return c, e
	}
	e = json.Unmarshal(b, &c)
	c.Version = ver
	return c, e
}
func (c Config) Validate() error {
	if c.Plots < 1 || c.Plots > 16 || c.ProtectionMinutes < 0 || c.ProtectionMinutes > 1440 || c.StealPercent < 1 || c.StealPercent > 40 || c.MaxStolenPercent < 0 || c.MaxStolenPercent > 40 || c.DailySteals < 0 || c.DailySteals > 100 || c.DailyPerOwner < 0 || c.DailyPerOwner > 100 || len(c.Crops) > 50 {
		return Bad("农场规则超出允许范围（主人至少保留60%）")
	}
	if c.StartsAt != nil && c.EndsAt != nil && !c.EndsAt.After(*c.StartsAt) {
		return Bad("活动结束时间必须晚于开始时间")
	}
	seen := map[string]bool{}
	for _, v := range c.Crops {
		if v.ID == "" || len(v.ID) > 60 || seen[v.ID] || v.Name == "" || len(v.Name) > 120 || v.Hours < 1 || v.Hours > 720 || v.Yield < 1 || v.Yield > 1000000 {
			return Bad("作物配置无效：成熟时间至少1小时")
		}
		seen[v.ID] = true
	}
	return nil
}
func (s *Store) SaveConfig(ctx context.Context, c Config, actor int64) (Config, error) {
	if e := c.Validate(); e != nil {
		return c, e
	}
	e := s.tx(ctx, func(tx *sql.Tx) error {
		var ver int64
		e := One(ctx, tx, `UPDATE reward_settings SET config=$1::jsonb,version=version+1 WHERE id=TRUE AND version=$2 RETURNING version`, []any{JSON(c), c.Version}, &ver)
		if e == sql.ErrNoRows {
			return Conflict("规则已修改，请刷新")
		}
		if e != nil {
			return e
		}
		c.Version = ver
		return Audit(ctx, tx, 0, actor, 0, "config_saved", c)
	})
	return c, e
}
func (c Config) farmAllowed(user int64, now time.Time) error {
	if !c.FarmEnabled || c.Maintenance {
		return Denied("农场暂未开放或正在维护")
	}
	if c.StartsAt != nil && now.Before(*c.StartsAt) || c.EndsAt != nil && !now.Before(*c.EndsAt) {
		return Denied("不在活动开放时间")
	}
	if len(c.AllowedUserIDs) > 0 && !contains(c.AllowedUserIDs, user) {
		return Denied("暂未开放参与")
	}
	return nil
}
