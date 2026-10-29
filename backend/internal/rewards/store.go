// Package rewards owns account-bound promotional assets. Gateway adapters must
// use the same database transaction as balance billing when spending an asset.
package rewards

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	apierrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}
type Store struct {
	DB  *sql.DB
	Now func() time.Time
}

func New(db *sql.DB) *Store { return &Store{DB: db, Now: time.Now} }
func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
func Bad(msg string) error      { return apierrors.BadRequest("REWARD_INVALID", msg) }
func Conflict(msg string) error { return apierrors.Conflict("REWARD_CONFLICT", msg) }
func Denied(msg string) error   { return apierrors.Forbidden("REWARD_DISABLED", msg) }
func One(ctx context.Context, q Executor, query string, args []any, dst ...any) error {
	rows, e := q.QueryContext(ctx, query, args...)
	if e != nil {
		return e
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if e = rows.Err(); e != nil {
			return e
		}
		return sql.ErrNoRows
	}
	return rows.Scan(dst...)
}
func JSON(v any) string { b, _ := json.Marshal(v); return string(b) }
func (s *Store) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback() }()
	if e = fn(tx); e != nil {
		return e
	}
	return tx.Commit()
}
func Audit(ctx context.Context, q Executor, user, actor, card int64, kind string, detail any) error {
	_, e := q.ExecContext(ctx, `INSERT INTO reward_events(user_id,actor_id,card_id,kind,detail) VALUES($1,$2,$3,$4,$5::jsonb)`, user, actor, card, kind, JSON(detail))
	return e
}

type Template struct {
	ID        int64    `json:"id"`
	Version   int64    `json:"version"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Model     string   `json:"model"`
	Models    []string `json:"models,omitempty"`
	GroupIDs  []int64  `json:"group_ids"`
	Rate      float64  `json:"rate"`
	Tokens    int64    `json:"tokens"`
	Hours     int      `json:"hours"`
	ClaimDays int      `json:"claim_days"`
	Published bool     `json:"published"`
}

// ModelIDs also supports previously issued single-model snapshots.
func (t Template) ModelIDs() []string {
	if len(t.Models) > 0 {
		return t.Models
	}
	if t.Model != "" {
		return []string{t.Model}
	}
	return nil
}
func (t Template) matchesModel(model string) bool {
	for _, id := range t.ModelIDs() {
		if id == model {
			return true
		}
	}
	return false
}
func (t Template) overlaps(other Template) bool {
	if t.Kind != other.Kind {
		return false
	}
	for _, g := range t.GroupIDs {
		if !contains(other.GroupIDs, g) {
			continue
		}
		if t.Kind == "group_rate" {
			return true
		}
		for _, model := range t.ModelIDs() {
			if other.matchesModel(model) {
				return true
			}
		}
	}
	return false
}

func (t Template) Validate() error {
	if strings.TrimSpace(t.Name) == "" || len(t.Name) > 120 || len(t.GroupIDs) == 0 || len(t.GroupIDs) > 100 || t.Hours < 1 || t.Hours > 8760 || t.ClaimDays < 1 || t.ClaimDays > 365 {
		return Bad("名称、分组、有效时长或保存期限无效")
	}
	for _, g := range t.GroupIDs {
		if g <= 0 {
			return Bad("请选择有效分组")
		}
	}
	if t.Kind != "model_tokens" && t.Hours%24 != 0 {
		return Bad("有效天数必须为1至365的整数")
	}
	switch t.Kind {
	case "model_day", "model_tokens":
		models := t.ModelIDs()
		if len(models) == 0 || len(models) > 100 {
			return Bad("请选择1至100个精确模型 ID")
		}
		for _, model := range models {
			if model == "" || len(model) > 200 || strings.ContainsAny(model, "*?") || strings.ContainsFunc(model, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
				return Bad("请选择精确模型 ID，不支持通配符或空白字符")
			}
		}
		if t.Kind == "model_tokens" && (t.Tokens <= 0 || t.Tokens > 1000000000000) {
			return Bad("Token额度无效")
		}
	case "group_rate":
		if len(t.GroupIDs) != 1 || t.Rate < 0 || t.Rate > 100 || math.IsNaN(t.Rate) || math.IsInf(t.Rate, 0) {
			return Bad("分组倍率无效")
		}
	default:
		return Bad("不支持的卡券类型")
	}
	return nil
}
func (s *Store) SaveTemplate(ctx context.Context, t Template, actor int64) (Template, error) {
	if e := t.Validate(); e != nil {
		return t, e
	}
	if t.Kind == "group_rate" {
		t.Model, t.Models = "", nil
	} else {
		ids := t.ModelIDs()
		t.Models = nil
		seen := map[string]bool{}
		for _, id := range ids {
			if !seen[id] {
				t.Models = append(t.Models, id)
				seen[id] = true
			}
		}
		t.Model = t.Models[0]
	}
	e := s.tx(ctx, func(tx *sql.Tx) error {
		for _, g := range t.GroupIDs {
			var id int64
			if e := One(ctx, tx, `SELECT id FROM groups WHERE id=$1 AND deleted_at IS NULL`, []any{g}, &id); e != nil {
				return Bad("分组不存在")
			}
		}
		if t.ID == 0 {
			t.Version = 1
			if e := One(ctx, tx, `INSERT INTO reward_templates(config,published) VALUES($1::jsonb,$2) RETURNING id`, []any{JSON(t), t.Published}, &t.ID); e != nil {
				return e
			}
		} else {
			var ver int64
			if e := One(ctx, tx, `UPDATE reward_templates SET config=$1::jsonb,published=$2,version=version+1 WHERE id=$3 AND version=$4 RETURNING version`, []any{JSON(t), t.Published, t.ID, t.Version}, &ver); e != nil {
				if errors.Is(e, sql.ErrNoRows) {
					return Conflict("模板已修改，请刷新")
				}
				return e
			}
			t.Version = ver
		}
		return Audit(ctx, tx, 0, actor, 0, "template_saved", t)
	})
	return t, e
}
func LoadTemplate(ctx context.Context, q Executor, id int64, published bool) (Template, error) {
	var t Template
	var raw []byte
	e := One(ctx, q, `SELECT id,version,config,published FROM reward_templates WHERE id=$1 AND (NOT $2 OR published)`, []any{id, published}, &t.ID, &t.Version, &raw, &t.Published)
	if e != nil {
		return t, e
	}
	var data Template
	if e = json.Unmarshal(raw, &data); e != nil {
		return t, e
	}
	data.ID = t.ID
	data.Version = t.Version
	data.Published = t.Published
	return data, nil
}
func (s *Store) Templates(ctx context.Context) ([]Template, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT id FROM reward_templates ORDER BY id DESC`)
	if e != nil {
		return nil, e
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if e = rows.Scan(&id); e != nil {
			_ = rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	_ = rows.Close()
	if e != nil {
		return nil, e
	}
	out := []Template{}
	for _, id := range ids {
		t, e := LoadTemplate(ctx, s.DB, id, false)
		if e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, nil
}

type Card struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Template     Template   `json:"template"`
	Status       string     `json:"status"`
	Source       string     `json:"source"`
	Remaining    int64      `json:"remaining_tokens"`
	ClaimExpires time.Time  `json:"claim_expires_at"`
	Activated    *time.Time `json:"activated_at"`
	Expires      *time.Time `json:"expires_at"`
	Created      time.Time  `json:"created_at"`
}

func Grant(ctx context.Context, q Executor, user int64, t Template, source, key string, now time.Time) (int64, error) {
	var id int64
	e := One(ctx, q, `INSERT INTO reward_cards(user_id,template_id,template_version,snapshot,source,source_key,claim_expires_at,remaining_tokens,created_at) VALUES($1,$2,$3,$4::jsonb,$5,$6,$7,$8,$9) ON CONFLICT(source_key) DO NOTHING RETURNING id`, []any{user, t.ID, t.Version, JSON(t), source, key, now.Add(time.Duration(t.ClaimDays) * 24 * time.Hour), t.Tokens, now}, &id)
	if errors.Is(e, sql.ErrNoRows) {
		var owner int64
		e = One(ctx, q, `SELECT id,user_id FROM reward_cards WHERE source_key=$1`, []any{key}, &id, &owner)
		if owner != user {
			return 0, Conflict("重复发卡来源不匹配")
		}
		return id, e
	}
	if e != nil {
		return 0, e
	}
	return id, Audit(ctx, q, user, 0, id, "granted", map[string]any{"source": source, "source_key": key})
}
func (s *Store) Grant(ctx context.Context, user, template, actor int64, key, reason string) (int64, error) {
	if strings.TrimSpace(reason) == "" || len(key) < 8 {
		return 0, Bad("请填写发放原因和请求编号")
	}
	var id int64
	e := s.tx(ctx, func(tx *sql.Tx) error {
		// Serialize retries before checking the mutable publication/settings state.
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, fmt.Sprintf("admin-grant:%d:%s", actor, key)); err != nil {
			return err
		}
		var oldUser, oldTemplate int64
		err := One(ctx, tx, `SELECT id,user_id,template_id FROM reward_cards WHERE source_key=$1`, []any{fmt.Sprintf("admin:%d:%s", actor, key)}, &id, &oldUser, &oldTemplate)
		if err == nil {
			if oldUser != user || oldTemplate != template {
				return Conflict("请求编号已用于其他发卡")
			}
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}
		cfg, e := s.Config(ctx, tx)
		if e != nil {
			return e
		}
		if !cfg.Enabled {
			return Denied("卡券未开放")
		}
		t, e := LoadTemplate(ctx, tx, template, true)
		if e != nil {
			return e
		}
		id, e = Grant(ctx, tx, user, t, "admin", fmt.Sprintf("admin:%d:%s", actor, key), s.now())
		if e != nil {
			return e
		}
		return Audit(ctx, tx, user, actor, id, "admin_grant", map[string]string{"reason": reason})
	})
	return id, e
}
func (s *Store) Cards(ctx context.Context, user int64) ([]Card, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT id,user_id,snapshot,status,source,remaining_tokens,claim_expires_at,activated_at,expires_at,created_at FROM reward_cards WHERE ($1::bigint=0 OR user_id=$1) ORDER BY id DESC LIMIT 500`, user)
	if e != nil {
		return nil, e
	}
	defer func() { _ = rows.Close() }()
	out := []Card{}
	for rows.Next() {
		var c Card
		var raw []byte
		if e = rows.Scan(&c.ID, &c.UserID, &raw, &c.Status, &c.Source, &c.Remaining, &c.ClaimExpires, &c.Activated, &c.Expires, &c.Created); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(raw, &c.Template); e != nil {
			return nil, e
		}
		if c.Status == "unused" && !s.now().Before(c.ClaimExpires) || c.Status == "active" && c.Expires != nil && !s.now().Before(*c.Expires) {
			c.Status = "expired"
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s *Store) Activate(ctx context.Context, user, id int64) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		cfg, e := s.Config(ctx, tx)
		if e != nil {
			return e
		}
		if !cfg.Enabled || !cfg.ActivationEnabled {
			return Denied("暂不开放激活")
		}
		var uid int64
		if e = One(ctx, tx, `SELECT id FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, []any{user}, &uid); e != nil {
			return e
		}
		var raw []byte
		var status string
		var claim time.Time
		if e = One(ctx, tx, `SELECT snapshot,status,claim_expires_at FROM reward_cards WHERE user_id=$1 AND id=$2 FOR UPDATE`, []any{user, id}, &raw, &status, &claim); e != nil {
			return e
		}
		if status == "active" {
			return nil
		}
		now := s.now()
		if status != "unused" || !now.Before(claim) {
			return Conflict("卡片已使用或过期")
		}
		var t Template
		if e = json.Unmarshal(raw, &t); e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, `SELECT snapshot FROM reward_cards WHERE user_id=$1 AND status='active' AND expires_at>$2`, user, now)
		if e != nil {
			return e
		}
		overlap := false
		for rows.Next() {
			var b []byte
			if e = rows.Scan(&b); e != nil {
				break
			}
			var other Template
			if e = json.Unmarshal(b, &other); e != nil {
				break
			}
			if t.overlaps(other) {
				overlap = true
			}
		}
		rowErr := rows.Err()
		_ = rows.Close()
		if e != nil {
			return e
		}
		if rowErr != nil {
			return rowErr
		}
		if overlap {
			return Conflict("同范围同类型卡片正在使用")
		}
		_, e = tx.ExecContext(ctx, `UPDATE reward_cards SET status='active',activated_at=$1,expires_at=$2 WHERE id=$3`, now, now.Add(time.Duration(t.Hours)*time.Hour), id)
		if e != nil {
			return e
		}
		return Audit(ctx, tx, user, user, id, "activated", t)
	})
}
func (s *Store) Revoke(ctx context.Context, id, actor int64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return Bad("撤销必须填写原因")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		var user int64
		var status string
		if err := One(ctx, tx, `SELECT user_id,status FROM reward_cards WHERE id=$1 FOR UPDATE`, []any{id}, &user, &status); err != nil {
			return err
		}
		if status == "revoked" {
			return nil
		}
		if e := One(ctx, tx, `UPDATE reward_cards SET status='revoked',revoked_at=$1 WHERE id=$2 AND status<>'revoked' RETURNING user_id`, []any{s.now(), id}, &user); e != nil {
			return e
		}
		return Audit(ctx, tx, user, actor, id, "revoked", map[string]string{"reason": reason})
	})
}
func contains(a []int64, v int64) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

type Capture struct {
	UserID    int64     `json:"user_id"`
	GroupID   int64     `json:"group_id"`
	Model     string    `json:"model"`
	At        time.Time `json:"at"`
	Free      bool      `json:"free"`
	Rate      *float64  `json:"rate,omitempty"`
	Cards     []int64   `json:"cards"`
	TokenCard int64     `json:"token_card"`
	TextOnly  bool      `json:"text_only"`
}
type captureKey struct{}

func WithCapture(ctx context.Context, c *Capture) context.Context {
	return context.WithValue(ctx, captureKey{}, c)
}
func FromContext(ctx context.Context) *Capture {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(captureKey{}).(*Capture)
	return c
}
func (s *Store) Capture(ctx context.Context, user, group int64, model string, textOnly bool) (*Capture, error) {
	cfg, e := s.Config(ctx, s.DB)
	if e != nil {
		return nil, e
	}
	if !cfg.Enabled {
		return nil, nil
	}
	now := s.now()
	rows, e := s.DB.QueryContext(ctx, `SELECT id,snapshot,remaining_tokens FROM reward_cards WHERE user_id=$1 AND status='active' AND activated_at<=$2 AND expires_at>$2 ORDER BY id`, user, now)
	if e != nil {
		return nil, e
	}
	defer func() { _ = rows.Close() }()
	c := &Capture{UserID: user, GroupID: group, Model: model, At: now, TextOnly: textOnly, Cards: []int64{}}
	for rows.Next() {
		var id, left int64
		var raw []byte
		if e = rows.Scan(&id, &raw, &left); e != nil {
			return nil, e
		}
		var t Template
		if e = json.Unmarshal(raw, &t); e != nil {
			return nil, e
		}
		if !contains(t.GroupIDs, group) {
			continue
		}
		switch t.Kind {
		case "model_day":
			if t.matchesModel(model) {
				c.Free = true
				c.Cards = append(c.Cards, id)
			}
		case "group_rate":
			if c.Rate == nil || t.Rate < *c.Rate {
				rate := t.Rate
				c.Rate = &rate
				c.Cards = append(c.Cards, id)
			}
		case "model_tokens":
			if t.matchesModel(model) && left > 0 {
				c.TokenCard = id
				c.Cards = append(c.Cards, id)
			}
		}
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	if len(c.Cards) == 0 {
		return nil, nil
	}
	return c, nil
}

// Quote and Spend operate only on customer text charges; supplier costs never enter here.
func AdjustText(original, text float64, c *Capture, currentRate float64) (float64, float64) {
	text = math.Max(0, math.Min(original, text))
	originalAmount := decimal.NewFromFloat(original)
	textAmount := decimal.NewFromFloat(text)
	discount := decimal.Zero
	if c != nil {
		if c.Rate != nil && currentRate > 0 && *c.Rate < currentRate {
			discount = textAmount.Mul(decimal.NewFromInt(1).Sub(decimal.NewFromFloat(*c.Rate).Div(decimal.NewFromFloat(currentRate))))
		}
		if c.Free {
			discount = textAmount
		}
	}
	paid, _ := originalAmount.Sub(discount).Round(8).Float64()
	remaining, _ := textAmount.Sub(discount).Round(8).Float64()
	return paid, remaining
}

func money(v float64) float64 {
	f, _ := decimal.NewFromFloat(math.Max(0, v)).Round(8).Float64()
	return f
}

// Spend must run after the existing request deduplication claim and inside its transaction.
func Spend(ctx context.Context, tx *sql.Tx, c *Capture, key int64, request string, totalTokens int64, original, text, currentRate float64) (float64, error) {
	paid, text := AdjustText(original, text, c, currentRate)
	if c == nil {
		return money(original), nil
	}
	covered := int64(0)
	if !c.Free && c.TokenCard > 0 && totalTokens > 0 && text > 0 {
		var left int64
		var owner int64
		if e := tx.QueryRowContext(ctx, `SELECT remaining_tokens,user_id FROM reward_cards WHERE id=$1 FOR UPDATE`, c.TokenCard).Scan(&left, &owner); e != nil {
			return 0, e
		}
		if owner != c.UserID {
			return 0, Bad("卡片归属不匹配")
		}
		covered = min(left, totalTokens)
		if covered > 0 {
			_, e := tx.ExecContext(ctx, `UPDATE reward_cards SET remaining_tokens=remaining_tokens-$1,status=CASE WHEN remaining_tokens=$1 AND status='active' THEN 'exhausted' ELSE status END WHERE id=$2`, covered, c.TokenCard)
			if e != nil {
				return 0, e
			}
			off := decimal.NewFromFloat(text).Mul(decimal.NewFromInt(covered)).Div(decimal.NewFromInt(totalTokens))
			v, _ := off.Round(8).Float64()
			paid = money(paid - v)
		}
	}
	original = money(original)
	discount := money(original - paid)
	_, e := tx.ExecContext(ctx, `INSERT INTO reward_usage(user_id,api_key_id,request_id,model,group_id,card_ids,tokens,covered_tokens,original_cost,discount,paid_cost,created_at) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11,$12)`, c.UserID, key, request, c.Model, c.GroupID, JSON(c.Cards), totalTokens, covered, original, discount, paid, c.At)
	return paid, e
}
func (s *Store) Records(ctx context.Context, user int64, kind string) (json.RawMessage, error) {
	query := `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT * FROM reward_usage WHERE ($1::bigint=0 OR user_id=$1) ORDER BY id DESC LIMIT 200) x`
	if kind == "events" {
		query = `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT * FROM reward_events WHERE ($1::bigint=0 OR user_id=$1) ORDER BY id DESC LIMIT 200) x`
	}
	var raw []byte
	e := s.DB.QueryRowContext(ctx, query, user).Scan(&raw)
	return raw, e
}
