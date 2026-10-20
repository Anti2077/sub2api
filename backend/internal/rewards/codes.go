package rewards

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *Store) Codes(ctx context.Context, template int64, count, days int, actor int64) ([]string, error) {
	return s.CodesWithKey(ctx, template, count, days, actor, "")
}
func (s *Store) CodesWithKey(ctx context.Context, template int64, count, days int, actor int64, key string) ([]string, error) {
	if count < 1 || count > 100 || days < 1 || days > 365 {
		return nil, Bad("每批1至100个兑换码，有效期1至365天")
	}
	codes := []string{}
	if key != "" && (len(key) < 8 || len(key) > 128 || strings.TrimSpace(key) != key) {
		return nil, Bad("请求编号无效")
	}
	e := s.tx(ctx, func(tx *sql.Tx) error {
		if key != "" {
			if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, fmt.Sprintf("reward-codes:%d:%s", actor, key)); err != nil {
				return err
			}
			var oldTemplate int64
			var oldCount, oldDays int
			var raw []byte
			err := One(ctx, tx, `SELECT template_id,count,days,codes FROM reward_admin_batches WHERE actor_id=$1 AND request_key=$2`, []any{actor, key}, &oldTemplate, &oldCount, &oldDays, &raw)
			if err == nil {
				if oldTemplate != template || oldCount != count || oldDays != days {
					return Conflict("请求编号已用于不同兑换码批次")
				}
				return json.Unmarshal(raw, &codes)
			}
			if err != sql.ErrNoRows {
				return err
			}
		}
		t, e := LoadTemplate(ctx, tx, template, true)
		if e != nil {
			return e
		}
		for i := 0; i < count; i++ {
			code := nonce()
			var id int64
			e = tx.QueryRowContext(ctx, `INSERT INTO redeem_codes(code,type,value,status,expires_at) VALUES($1,'card',0,'unused',$2) RETURNING id`, code, s.now().Add(time.Duration(days)*24*time.Hour)).Scan(&id)
			if e != nil {
				return e
			}
			_, e = tx.ExecContext(ctx, `INSERT INTO reward_redemptions(code_id,template_id,template_version,snapshot) VALUES($1,$2,$3,$4::jsonb)`, id, t.ID, t.Version, JSON(t))
			if e != nil {
				return e
			}
			codes = append(codes, code)
		}
		if key != "" {
			if _, err := tx.ExecContext(ctx, `INSERT INTO reward_admin_batches(actor_id,request_key,template_id,count,days,codes) VALUES($1,$2,$3,$4,$5,$6::jsonb)`, actor, key, template, count, days, JSON(codes)); err != nil {
				return err
			}
		}
		return Audit(ctx, tx, 0, actor, 0, "codes_generated", map[string]any{"template_id": template, "count": count})
	})
	return codes, e
}
func (s *Store) AdminCodes(ctx context.Context) (json.RawMessage, error) {
	var raw []byte
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(x)),'[]') FROM (SELECT c.id,c.code,c.status,c.expires_at,c.used_by,c.used_at,c.created_at,r.snapshot AS template FROM redeem_codes c JOIN reward_redemptions r ON r.code_id=c.id ORDER BY c.id DESC LIMIT 500) x`).Scan(&raw)
	return raw, err
}

// SetCodeStatus serializes against redemption and never reopens a used code.
func (s *Store) SetCodeStatus(ctx context.Context, id, actor int64, status string) error {
	if status != "unused" && status != "disabled" {
		return Bad("兑换码状态无效")
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		var current string
		var expires sql.NullTime
		if err := One(ctx, tx, `SELECT c.status,c.expires_at FROM redeem_codes c JOIN reward_redemptions r ON r.code_id=c.id WHERE c.id=$1 FOR UPDATE OF c`, []any{id}, &current, &expires); err != nil {
			return err
		}
		if current == "used" || current == "expired" || expires.Valid && !s.now().Before(expires.Time) {
			return Conflict("兑换码已使用或已过期")
		}
		if current == status {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE redeem_codes SET status=$1 WHERE id=$2`, status, id); err != nil {
			return err
		}
		return Audit(ctx, tx, 0, actor, 0, "code_status_changed", map[string]any{"id": id, "status": status})
	})
}
func Redeem(ctx context.Context, q Executor, user, code int64) (int64, error) {
	var raw []byte
	if e := One(ctx, q, `SELECT snapshot FROM reward_redemptions WHERE code_id=$1`, []any{code}, &raw); e != nil {
		return 0, e
	}
	var t Template
	if e := json.Unmarshal(raw, &t); e != nil {
		return 0, e
	}
	return Grant(ctx, q, user, t, "code", fmt.Sprintf("code:%d", code), time.Now().UTC())
}

// RedemptionReceipt reads the exact issued snapshot inside the redemption transaction.
func RedemptionReceipt(ctx context.Context, q Executor, user, id int64) (*Card, error) {
	var card Card
	var raw []byte
	err := One(ctx, q, `SELECT id,user_id,snapshot,status,source,remaining_tokens,claim_expires_at,activated_at,expires_at,created_at FROM reward_cards WHERE id=$1 AND user_id=$2`, []any{id, user}, &card.ID, &card.UserID, &raw, &card.Status, &card.Source, &card.Remaining, &card.ClaimExpires, &card.Activated, &card.Expires, &card.Created)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &card.Template); err != nil {
		return nil, err
	}
	return &card, nil
}
