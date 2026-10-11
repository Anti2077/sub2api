package repository

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountFolderRejectsThirdLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &accountFolderRepository{db: db}
	parent := int64(2)
	mock.ExpectBegin()
	mock.ExpectExec("pg_advisory_xact_lock").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT parent_id").WithArgs(parent).WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(1))
	mock.ExpectRollback()
	_, err = repo.Save(context.Background(), 0, "third level", &parent)
	require.ErrorContains(t, err, "two levels")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountFolderMoveRollsBackWhenAccountDeleted(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &accountFolderRepository{db: db}
	folder := int64(3)
	mock.ExpectBegin()
	mock.ExpectExec("pg_advisory_xact_lock").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id FROM account_folders").WithArgs(folder).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(folder))
	mock.ExpectQuery("SELECT id FROM accounts").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectRollback()
	_, err = repo.Move(context.Background(), []int64{10, 11}, &folder)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountFolderFilterAppliedBeforePaginationCount(t *testing.T) {
	for _, folder := range []string{"7", "unfiled"} {
		t.Run(folder, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			repo := &accountRepository{client: client}
			ctx, err := service.WithAccountFolderFilter(context.Background(), folder)
			require.NoError(t, err)
			if folder == "7" {
				mock.ExpectQuery(`SELECT COUNT\(.*EXISTS.*account_folder_members.*folder_children.*parent_id`).WithArgs(int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
			} else {
				mock.ExpectQuery(`SELECT COUNT\(.*NOT.*EXISTS.*account_folder_members`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			}
			count, err := repo.accountListFilteredQuery(ctx, "", "", "", "", 0, "").Count(ctx)
			require.NoError(t, err)
			require.Positive(t, count)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
