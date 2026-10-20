package rewards

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Plot struct {
	ID          int64      `json:"id"`
	Slot        int        `json:"slot"`
	Crop        *Crop      `json:"crop"`
	PlantedAt   *time.Time `json:"planted_at"`
	ReadyAt     *time.Time `json:"ready_at"`
	Stolen      int64      `json:"stolen"`
	HarvestedAt *time.Time `json:"harvested_at"`
	Cycle       string     `json:"cycle"`
}
type Farm struct {
	UserID     int64     `json:"user_id"`
	Public     bool      `json:"public"`
	Crystals   int64     `json:"crystals"`
	Plots      []Plot    `json:"plots"`
	Config     Config    `json:"config"`
	ServerTime time.Time `json:"server_time"`
}

func nonce() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func lockFarmUsers(ctx context.Context, tx *sql.Tx, ids ...int64) error {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for i, id := range ids {
		if i > 0 && id == ids[i-1] {
			continue
		}
		if _, e := tx.ExecContext(ctx, `INSERT INTO farm_users(user_id) VALUES($1) ON CONFLICT DO NOTHING`, id); e != nil {
			return e
		}
		var v int64
		if e := tx.QueryRowContext(ctx, `SELECT user_id FROM farm_users WHERE user_id=$1 FOR UPDATE`, id).Scan(&v); e != nil {
			return e
		}
	}
	return nil
}
func (s *Store) Farm(ctx context.Context, user, viewer int64) (Farm, error) {
	var f Farm
	cfg, e := s.Config(ctx, s.DB)
	if e != nil {
		return f, e
	}
	if e = cfg.farmAllowed(viewer, s.now()); e != nil {
		return f, e
	}
	cfg.AllowedUserIDs = nil
	f = Farm{UserID: user, Config: cfg, ServerTime: s.now(), Plots: []Plot{}}
	if user == viewer {
		_, e = s.DB.ExecContext(ctx, `INSERT INTO farm_users(user_id) VALUES($1) ON CONFLICT DO NOTHING`, user)
		if e != nil {
			return f, e
		}
	}
	e = s.DB.QueryRowContext(ctx, `SELECT public,crystals FROM farm_users WHERE user_id=$1`, user).Scan(&f.Public, &f.Crystals)
	if e != nil {
		return f, e
	}
	if viewer != user {
		if !f.Public {
			return f, Denied("对方尚未开放农场")
		}
		f.Crystals = 0
		cfg.AllowedUserIDs = nil
		f.Config = cfg
	}
	rows, e := s.DB.QueryContext(ctx, `SELECT id,slot,crop,planted_at,ready_at,stolen,harvested_at,cycle FROM farm_plots WHERE user_id=$1 ORDER BY slot`, user)
	if e != nil {
		return f, e
	}
	defer func() { _ = rows.Close() }()
	slots := map[int]bool{}
	for rows.Next() {
		var p Plot
		var raw []byte
		if e = rows.Scan(&p.ID, &p.Slot, &raw, &p.PlantedAt, &p.ReadyAt, &p.Stolen, &p.HarvestedAt, &p.Cycle); e != nil {
			return f, e
		}
		if len(raw) > 0 && string(raw) != "null" {
			if e = json.Unmarshal(raw, &p.Crop); e != nil {
				return f, e
			}
		}
		f.Plots = append(f.Plots, p)
		slots[p.Slot] = true
	}
	for i := 1; i <= cfg.Plots; i++ {
		if !slots[i] {
			f.Plots = append(f.Plots, Plot{Slot: i})
		}
	}
	sort.Slice(f.Plots, func(i, j int) bool { return f.Plots[i].Slot < f.Plots[j].Slot })
	return f, rows.Err()
}

type FarmAction struct {
	Key     string `json:"request_key"`
	Kind    string `json:"kind"`
	Slot    int    `json:"slot"`
	CropID  string `json:"crop_id"`
	OwnerID int64  `json:"owner_id"`
	Public  bool   `json:"public"`
	ShopID  int64  `json:"shop_id"`
}

func (s *Store) Action(ctx context.Context, user int64, a FarmAction) (json.RawMessage, error) {
	if len(a.Key) < 8 || len(a.Key) > 128 {
		return nil, Bad("请提供8至128字符请求编号")
	}
	var receipt json.RawMessage
	e := s.tx(ctx, func(tx *sql.Tx) error {
		cfg, e := s.Config(ctx, tx)
		if e != nil {
			return e
		}
		if e = cfg.farmAllowed(user, s.now()); e != nil {
			return e
		}
		ids := []int64{user}
		if a.Kind == "steal" {
			if a.OwnerID <= 0 || a.OwnerID == user {
				return Bad("请选择其他人的农场")
			}
			ids = append(ids, a.OwnerID)
		}
		if e = lockFarmUsers(ctx, tx, ids...); e != nil {
			return e
		}
		var raw []byte
		var kind string
		e = tx.QueryRowContext(ctx, `SELECT kind,receipt FROM farm_actions WHERE user_id=$1 AND request_key=$2`, user, a.Key).Scan(&kind, &raw)
		if e == nil {
			var old struct {
				Action FarmAction `json:"action"`
			}
			if e = json.Unmarshal(raw, &old); e != nil {
				return e
			}
			if JSON(old.Action) != JSON(a) {
				return Conflict("请求编号已用于其他操作")
			}
			receipt = raw
			return nil
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		result := map[string]any{"action": a}
		now := s.now()
		switch a.Kind {
		case "privacy":
			_, e = tx.ExecContext(ctx, `UPDATE farm_users SET public=$1 WHERE user_id=$2`, a.Public, user)
		case "plant":
			if !cfg.PlantingEnabled {
				return Denied("暂不开放新种植")
			}
			if a.Slot < 1 || a.Slot > cfg.Plots {
				return Bad("地块未开放")
			}
			var crop *Crop
			for _, c := range cfg.Crops {
				if c.ID == a.CropID && c.Enabled {
					v := c
					crop = &v
				}
			}
			if crop == nil {
				return Bad("作物未开放")
			}
			crop.ConfigVersion = cfg.Version
			crop.ProtectionMinutes = cfg.ProtectionMinutes
			crop.StealPercent = cfg.StealPercent
			crop.MaxStolenPercent = cfg.MaxStolenPercent
			var id int64
			cycle := nonce()
			e = tx.QueryRowContext(ctx, `INSERT INTO farm_plots(user_id,slot,crop,planted_at,ready_at,cycle) VALUES($1,$2,$3::jsonb,$4,$5,$6) ON CONFLICT(user_id,slot) DO UPDATE SET crop=EXCLUDED.crop,planted_at=EXCLUDED.planted_at,ready_at=EXCLUDED.ready_at,cycle=EXCLUDED.cycle,stolen=0,harvested_at=NULL WHERE farm_plots.crop IS NULL OR farm_plots.harvested_at IS NOT NULL RETURNING id`, user, a.Slot, JSON(crop), now, now.Add(time.Duration(crop.Hours)*time.Hour), cycle).Scan(&id)
			if e == sql.ErrNoRows {
				return Conflict("地块已有作物")
			}
			result["plot_id"] = id
		case "harvest", "steal":
			owner := user
			if a.Kind == "steal" {
				owner = a.OwnerID
				if !cfg.StealingEnabled {
					return Denied("暂不开放偷菜")
				}
				var ownPublic, otherPublic bool
				if e = tx.QueryRowContext(ctx, `SELECT public FROM farm_users WHERE user_id=$1`, user).Scan(&ownPublic); e != nil {
					return e
				}
				if e = tx.QueryRowContext(ctx, `SELECT public FROM farm_users WHERE user_id=$1`, owner).Scan(&otherPublic); e != nil {
					return e
				}
				if !ownPublic || !otherPublic {
					return Denied("双方均需开放农场")
				}
			}
			var p Plot
			var b []byte
			e = tx.QueryRowContext(ctx, `SELECT id,crop,ready_at,stolen,harvested_at,cycle FROM farm_plots WHERE user_id=$1 AND slot=$2 FOR UPDATE`, owner, a.Slot).Scan(&p.ID, &b, &p.ReadyAt, &p.Stolen, &p.HarvestedAt, &p.Cycle)
			if e != nil {
				return e
			}
			if e = json.Unmarshal(b, &p.Crop); e != nil {
				return e
			}
			if p.Crop == nil || p.ReadyAt == nil || p.HarvestedAt != nil || now.Before(*p.ReadyAt) {
				return Conflict("作物尚未成熟或已收获")
			}
			amount := p.Crop.Yield - p.Stolen
			ref := p.Cycle
			if a.Kind == "steal" {
				if now.Before(p.ReadyAt.Add(time.Duration(p.Crop.ProtectionMinutes) * time.Minute)) {
					return Conflict("成熟保护期内")
				}
				start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
				var count, perOwner int
				e = tx.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(*) FILTER(WHERE owner_id=$2) FROM farm_steals WHERE thief_id=$1 AND created_at >=$3 AND created_at<$4`, user, owner, start, start.Add(24*time.Hour)).Scan(&count, &perOwner)
				if e != nil {
					return e
				}
				if count >= cfg.DailySteals || perOwner >= cfg.DailyPerOwner {
					return Conflict("今日偷取次数已用完")
				}
				amount = min(p.Crop.Yield*int64(p.Crop.StealPercent)/100, p.Crop.Yield*int64(p.Crop.MaxStolenPercent)/100-p.Stolen)
				if amount <= 0 {
					return Conflict("已达到可偷上限")
				}
				_, e = tx.ExecContext(ctx, `INSERT INTO farm_steals(thief_id,owner_id,plot_id,cycle,amount,created_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, user, owner, p.ID, p.Cycle, amount, now)
				if e != nil {
					return e
				}
				var n int64
				if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM farm_ledger WHERE user_id=$1 AND kind='steal' AND reference=$2`, user, ref).Scan(&n); e != nil {
					return e
				}
				if n > 0 {
					return Conflict("本轮作物已偷取")
				}
				_, e = tx.ExecContext(ctx, `UPDATE farm_plots SET stolen=stolen+$1 WHERE id=$2`, amount, p.ID)
			} else {
				_, e = tx.ExecContext(ctx, `UPDATE farm_plots SET harvested_at=$1 WHERE id=$2`, now, p.ID)
			}
			if e != nil {
				return e
			}
			if e = crystalChange(ctx, tx, user, amount, a.Kind, ref, user); e != nil {
				return e
			}
			result["crystals"] = amount
		case "buy":
			if !cfg.Enabled || !cfg.ExchangeEnabled {
				return Denied("暂不开放兑换")
			}
			id, err := s.buy(ctx, tx, user, a.ShopID, a.Key, now)
			e = err
			result["card_id"] = id
		default:
			return Bad("不支持的操作")
		}
		if e != nil {
			return e
		}
		receipt = json.RawMessage(JSON(result))
		_, e = tx.ExecContext(ctx, `INSERT INTO farm_actions(user_id,request_key,kind,receipt,created_at) VALUES($1,$2,$3,$4::jsonb,$5)`, user, a.Key, a.Kind, string(receipt), now)
		return e
	})
	return receipt, e
}
func crystalChange(ctx context.Context, tx *sql.Tx, user, delta int64, kind, ref string, actor int64) error {
	res, e := tx.ExecContext(ctx, `UPDATE farm_users SET crystals=crystals+$1 WHERE user_id=$2 AND crystals+$1>=0`, delta, user)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return Conflict("晶能不足")
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO farm_ledger(user_id,delta,kind,reference,actor_id) VALUES($1,$2,$3,$4,$5)`, user, delta, kind, ref, actor)
	return e
}
func (s *Store) Neighbors(ctx context.Context, user int64, search string) (json.RawMessage, error) {
	cfg, e := s.Config(ctx, s.DB)
	if e != nil {
		return nil, e
	}
	if e = cfg.farmAllowed(user, s.now()); e != nil {
		return nil, e
	}
	var raw []byte
	e = s.DB.QueryRowContext(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT f.user_id,COALESCE(NULLIF(u.username,''),'农友') AS nickname FROM farm_users f JOIN users u ON u.id=f.user_id WHERE f.public AND u.deleted_at IS NULL AND f.user_id<>$1 AND ($2='' OR u.username ILIKE '%'||$2||'%') ORDER BY f.user_id LIMIT 50) x`, user, strings.TrimSpace(search)).Scan(&raw)
	return raw, e
}
func (s *Store) FarmLedger(ctx context.Context, user int64) (json.RawMessage, error) {
	var b []byte
	e := s.DB.QueryRowContext(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT * FROM farm_ledger WHERE ($1::bigint=0 OR user_id=$1) ORDER BY id DESC LIMIT 200) x`, user).Scan(&b)
	return b, e
}

type Shop struct {
	ID         int64      `json:"id"`
	Version    int64      `json:"version"`
	TemplateID int64      `json:"template_id"`
	Template   Template   `json:"template"`
	Price      int64      `json:"price"`
	Stock      int64      `json:"stock"`
	DailyLimit int        `json:"daily_limit"`
	TotalLimit int        `json:"total_limit"`
	Published  bool       `json:"published"`
	StartsAt   *time.Time `json:"starts_at"`
	EndsAt     *time.Time `json:"ends_at"`
}

func (s *Store) SaveShop(ctx context.Context, v Shop, actor int64) (Shop, error) {
	if v.Price < 1 || v.Price > 1000000000 || v.Stock < 0 || v.DailyLimit < 1 || v.TotalLimit < v.DailyLimit || v.StartsAt != nil && v.EndsAt != nil && !v.EndsAt.After(*v.StartsAt) {
		return v, Bad("商店配置无效")
	}
	e := s.tx(ctx, func(tx *sql.Tx) error {
		t, e := LoadTemplate(ctx, tx, v.TemplateID, v.Published)
		if e != nil {
			return e
		}
		v.Template = t
		if v.ID == 0 {
			v.Version = 1
			e = tx.QueryRowContext(ctx, `INSERT INTO farm_shop(template_id,snapshot,price,stock,daily_limit,total_limit,published,starts_at,ends_at) VALUES($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, v.TemplateID, JSON(t), v.Price, v.Stock, v.DailyLimit, v.TotalLimit, v.Published, v.StartsAt, v.EndsAt).Scan(&v.ID)
		} else {
			e = tx.QueryRowContext(ctx, `UPDATE farm_shop SET template_id=$1,snapshot=$2::jsonb,price=$3,stock=$4,daily_limit=$5,total_limit=$6,published=$7,starts_at=$8,ends_at=$9,version=version+1 WHERE id=$10 AND version=$11 RETURNING version`, v.TemplateID, JSON(t), v.Price, v.Stock, v.DailyLimit, v.TotalLimit, v.Published, v.StartsAt, v.EndsAt, v.ID, v.Version).Scan(&v.Version)
			if e == sql.ErrNoRows {
				return Conflict("商品已修改，请刷新")
			}
		}
		if e != nil {
			return e
		}
		return Audit(ctx, tx, 0, actor, 0, "shop_saved", v)
	})
	return v, e
}
func (s *Store) Shop(ctx context.Context, admin bool) ([]Shop, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT id,version,template_id,snapshot,price,stock,daily_limit,total_limit,published,starts_at,ends_at FROM farm_shop WHERE $1 OR (published AND (starts_at IS NULL OR starts_at<=$2) AND (ends_at IS NULL OR ends_at>$2)) ORDER BY id`, admin, s.now())
	if e != nil {
		return nil, e
	}
	defer func() { _ = rows.Close() }()
	out := []Shop{}
	for rows.Next() {
		var v Shop
		var b []byte
		if e = rows.Scan(&v.ID, &v.Version, &v.TemplateID, &b, &v.Price, &v.Stock, &v.DailyLimit, &v.TotalLimit, &v.Published, &v.StartsAt, &v.EndsAt); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &v.Template); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) buy(ctx context.Context, tx *sql.Tx, user, shop int64, key string, now time.Time) (int64, error) {
	var v Shop
	var b []byte
	e := tx.QueryRowContext(ctx, `SELECT id,snapshot,price,stock,daily_limit,total_limit,published,starts_at,ends_at FROM farm_shop WHERE id=$1 FOR UPDATE`, shop).Scan(&v.ID, &b, &v.Price, &v.Stock, &v.DailyLimit, &v.TotalLimit, &v.Published, &v.StartsAt, &v.EndsAt)
	if e != nil {
		return 0, e
	}
	if !v.Published || v.Stock < 1 || v.StartsAt != nil && now.Before(*v.StartsAt) || v.EndsAt != nil && !now.Before(*v.EndsAt) {
		return 0, Conflict("商品未开放或已售罄")
	}
	var total, daily int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(*) FILTER(WHERE created_at >= date_trunc('day',$3::timestamptz AT TIME ZONE 'UTC') AT TIME ZONE 'UTC') FROM farm_orders WHERE user_id=$1 AND shop_id=$2`, user, shop, now).Scan(&total, &daily)
	if e != nil {
		return 0, e
	}
	if total >= v.TotalLimit || daily >= v.DailyLimit {
		return 0, Conflict("已达兑换限额")
	}
	if e = json.Unmarshal(b, &v.Template); e != nil {
		return 0, e
	}
	ref := fmt.Sprintf("shop:%d:%s", user, key)
	if e = crystalChange(ctx, tx, user, -v.Price, "exchange", ref, user); e != nil {
		return 0, e
	}
	id, e := Grant(ctx, tx, user, v.Template, "farm", ref, now)
	if e != nil {
		return 0, e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE farm_shop SET stock=stock-1,version=version+1 WHERE id=$1`, shop); e != nil {
		return 0, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO farm_orders(user_id,shop_id,card_id,price,request_key,created_at) VALUES($1,$2,$3,$4,$5,$6)`, user, shop, id, v.Price, key, now)
	return id, e
}
