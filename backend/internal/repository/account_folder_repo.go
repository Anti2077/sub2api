package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type accountFolderRepository struct{ db *sql.DB }

func NewAccountFolderRepository(db *sql.DB) service.AccountFolderRepository {
	return &accountFolderRepository{db: db}
}

func (r *accountFolderRepository) Overview(ctx context.Context) (*service.AccountFolderOverview, error) {
	result := &service.AccountFolderOverview{Folders: []service.AccountFolder{}}
	rows, err := r.db.QueryContext(ctx, `SELECT f.id,f.name,f.parent_id,f.rules,
        (SELECT COUNT(*) FROM accounts a WHERE a.deleted_at IS NULL AND EXISTS (
            SELECT 1 FROM account_folders scope WHERE (scope.id=f.id OR scope.parent_id=f.id)
            AND account_matches_smart_folder(a.platform,a.type,a.status,a.name,a.extra,a.schedulable,a.expires_at,scope.rules))),
        (SELECT COUNT(*) FROM accounts a WHERE a.deleted_at IS NULL AND a.status='error' AND EXISTS (
            SELECT 1 FROM account_folders scope WHERE (scope.id=f.id OR scope.parent_id=f.id)
            AND account_matches_smart_folder(a.platform,a.type,a.status,a.name,a.extra,a.schedulable,a.expires_at,scope.rules)))
        FROM account_folders f ORDER BY LOWER(f.name),f.id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		f := service.AccountFolder{}
		var rules []byte
		if err = rows.Scan(&f.ID, &f.Name, &f.ParentID, &rules, &f.AccountCount, &f.ErrorCount); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(rules, &f.Rules); err != nil {
			return nil, err
		}
		result.Folders = append(result.Folders, f)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(*) FILTER (WHERE NOT EXISTS(SELECT 1 FROM account_folders f WHERE account_matches_smart_folder(a.platform,a.type,a.status,a.name,a.extra,a.schedulable,a.expires_at,f.rules))) FROM accounts a WHERE a.deleted_at IS NULL`).Scan(&result.Total, &result.Unfiled)
	return result, err
}

func folderDBError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return infraerrors.NotFound("ACCOUNT_FOLDER_NOT_FOUND", "Folder or account no longer exists")
	}
	var pgerr *pq.Error
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		return infraerrors.BadRequest("ACCOUNT_FOLDER_DUPLICATE", "A folder with this name already exists here")
	}
	return err
}
func (r *accountFolderRepository) Save(ctx context.Context, id int64, name string, parent *int64, rules *service.AccountFolderRules) (*service.AccountFolder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize topology changes so concurrent reparenting cannot create cycles
	// or a third level. Move/delete share this lock to avoid orphan races.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(246,1)`); err != nil {
		return nil, err
	}
	if parent != nil {
		var parentParent *int64
		if err = tx.QueryRowContext(ctx, `SELECT parent_id FROM account_folders WHERE id=$1`, *parent).Scan(&parentParent); err != nil {
			return nil, folderDBError(err)
		}
		var children bool
		if id > 0 {
			err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM account_folders WHERE parent_id=$1)`, id).Scan(&children)
			if err != nil {
				return nil, err
			}
		}
		if parentParent != nil || children {
			return nil, infraerrors.BadRequest("ACCOUNT_FOLDER_DEPTH", "Folders support at most two levels")
		}
	}
	payload, err := json.Marshal(rules)
	if err != nil {
		return nil, err
	}
	f := &service.AccountFolder{Rules: rules}
	if id == 0 {
		err = tx.QueryRowContext(ctx, `INSERT INTO account_folders(name,parent_id,rules) VALUES($1,$2,$3) RETURNING id,name,parent_id`, name, parent, string(payload)).Scan(&f.ID, &f.Name, &f.ParentID)
	} else {
		err = tx.QueryRowContext(ctx, `UPDATE account_folders SET name=$2,parent_id=$3,rules=$4 WHERE id=$1 RETURNING id,name,parent_id`, id, name, parent, string(payload)).Scan(&f.ID, &f.Name, &f.ParentID)
	}
	if err != nil {
		return nil, folderDBError(err)
	}
	return f, tx.Commit()
}
func (r *accountFolderRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(246,1)`); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM account_folders WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return folderDBError(sql.ErrNoRows)
	}
	return tx.Commit()
}
func (r *accountFolderRepository) Tags(ctx context.Context) ([]service.AccountTag, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tag,COUNT(DISTINCT a.id) FROM accounts a
 CROSS JOIN LATERAL jsonb_array_elements_text(CASE WHEN jsonb_typeof(a.extra->'account_tags')='array' THEN a.extra->'account_tags' ELSE '[]'::jsonb END) tag
 WHERE a.deleted_at IS NULL GROUP BY tag ORDER BY LOWER(tag),tag`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := []service.AccountTag{}
	for rows.Next() {
		var tag service.AccountTag
		if err = rows.Scan(&tag.Name, &tag.AccountCount); err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

// One statement updates only the label key. Row locks protect concurrent edits;
// any missing target or tag-limit failure rolls the entire transaction back.
func (r *accountFolderRepository) UpdateTags(ctx context.Context, ids []int64, tags []string, operation string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `UPDATE accounts a SET extra=COALESCE(extra,'{}'::jsonb) || jsonb_build_object('account_tags',
   (SELECT COALESCE(jsonb_agg(tag ORDER BY tag),'[]'::jsonb) FROM (
      SELECT DISTINCT tag FROM (
         SELECT jsonb_array_elements_text(CASE WHEN jsonb_typeof(a.extra->'account_tags')='array' THEN a.extra->'account_tags' ELSE '[]'::jsonb END) AS tag
         UNION ALL SELECT UNNEST($2::text[]) WHERE $3='add'
      ) candidates WHERE $3='add' OR NOT tag=ANY($2::text[])
   ) labels)) , updated_at=NOW()
 WHERE a.id=ANY($1::bigint[]) AND a.deleted_at IS NULL RETURNING jsonb_array_length(extra->'account_tags')`, pq.Array(ids), pq.Array(tags), operation)
	if err != nil {
		return 0, err
	}
	count := int64(0)
	overflow := false
	for rows.Next() {
		var size int
		if err = rows.Scan(&size); err != nil {
			break
		}
		count++
		overflow = overflow || size > service.MaxAccountTags
	}
	if err == nil {
		err = rows.Err()
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	if count != int64(len(ids)) {
		return 0, folderDBError(sql.ErrNoRows)
	}
	if overflow {
		return 0, infraerrors.BadRequest("INVALID_ACCOUNT_TAGS", "An account supports at most 32 tags")
	}
	return count, tx.Commit()
}
