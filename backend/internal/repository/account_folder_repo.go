package repository

import (
	"context"
	"database/sql"
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
	rows, err := r.db.QueryContext(ctx, `SELECT f.id,f.name,f.parent_id,COUNT(DISTINCT a.id),COUNT(DISTINCT a.id) FILTER (WHERE a.status='error')
        FROM account_folders f LEFT JOIN account_folders child ON child.parent_id=f.id
        LEFT JOIN account_folder_members m ON m.folder_id=f.id OR m.folder_id=child.id
        LEFT JOIN accounts a ON a.id=m.account_id AND a.deleted_at IS NULL
        GROUP BY f.id ORDER BY LOWER(f.name),f.id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		f := service.AccountFolder{}
		if err = rows.Scan(&f.ID, &f.Name, &f.ParentID, &f.AccountCount, &f.ErrorCount); err != nil {
			return nil, err
		}
		result.Folders = append(result.Folders, f)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(*) FILTER (WHERE NOT EXISTS(SELECT 1 FROM account_folder_members m WHERE m.account_id=a.id)) FROM accounts a WHERE a.deleted_at IS NULL`).Scan(&result.Total, &result.Unfiled)
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
func (r *accountFolderRepository) Save(ctx context.Context, id int64, name string, parent *int64) (*service.AccountFolder, error) {
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
	f := &service.AccountFolder{}
	if id == 0 {
		err = tx.QueryRowContext(ctx, `INSERT INTO account_folders(name,parent_id) VALUES($1,$2) RETURNING id,name,parent_id`, name, parent).Scan(&f.ID, &f.Name, &f.ParentID)
	} else {
		err = tx.QueryRowContext(ctx, `UPDATE account_folders SET name=$2,parent_id=$3 WHERE id=$1 RETURNING id,name,parent_id`, id, name, parent).Scan(&f.ID, &f.Name, &f.ParentID)
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
func (r *accountFolderRepository) Move(ctx context.Context, ids []int64, folder *int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(246,1)`); err != nil {
		return 0, err
	}
	if folder != nil {
		var id int64
		if err = tx.QueryRowContext(ctx, `SELECT id FROM account_folders WHERE id=$1`, *folder).Scan(&id); err != nil {
			return 0, folderDBError(err)
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM accounts WHERE id=ANY($1::bigint[]) AND deleted_at IS NULL FOR UPDATE`, pq.Array(ids))
	if err != nil {
		return 0, err
	}
	count := int64(0)
	for rows.Next() {
		count++
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return 0, err
	}
	if count != int64(len(ids)) {
		return 0, folderDBError(sql.ErrNoRows)
	}
	if folder == nil {
		_, err = tx.ExecContext(ctx, `DELETE FROM account_folder_members WHERE account_id=ANY($1::bigint[])`, pq.Array(ids))
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO account_folder_members(account_id,folder_id) SELECT UNNEST($1::bigint[]),$2 ON CONFLICT(account_id) DO UPDATE SET folder_id=EXCLUDED.folder_id`, pq.Array(ids), *folder)
	}
	if err != nil {
		return 0, err
	}
	return count, tx.Commit()
}
