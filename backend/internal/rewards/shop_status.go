package rewards

import (
	"context"
	"database/sql"
)

// SetShopPublished never copies a stale inventory or changes the reward snapshot.
func (s *Store) SetShopPublished(ctx context.Context, id, version, actor int64, published bool) (int64, error) {
	var next int64
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var templateID int64
		if err := One(ctx, tx, `SELECT template_id FROM farm_shop WHERE id=$1 AND version=$2 FOR UPDATE`, []any{id, version}, &templateID); err != nil {
			if err == sql.ErrNoRows {
				return Conflict("商品已修改，请刷新")
			}
			return err
		}
		if published {
			if _, err := LoadTemplate(ctx, tx, templateID, true); err != nil {
				return err
			}
		}
		if err := One(ctx, tx, `UPDATE farm_shop SET published=$1,version=version+1 WHERE id=$2 AND version=$3 RETURNING version`, []any{published, id, version}, &next); err != nil {
			return err
		}
		return Audit(ctx, tx, 0, actor, 0, "shop_status_changed", map[string]any{"id": id, "version": next, "published": published})
	})
	return next, err
}
