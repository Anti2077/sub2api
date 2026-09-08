//go:build unit

package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetUserAvatarURLsReturnsTrimmedNonEmptyURLs(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := newUserRepositoryWithSQL(nil, db)

	mock.ExpectQuery(regexp.QuoteMeta("FROM user_avatars\nWHERE user_id = ANY($1)")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "url"}).
			AddRow(int64(7), " https://cdn.example.com/7.png ").
			AddRow(int64(9), ""))

	avatarURLs, err := repo.GetUserAvatarURLs(context.Background(), []int64{7, 7, -1, 9})

	require.NoError(t, err)
	require.Equal(t, map[int64]string{7: "https://cdn.example.com/7.png"}, avatarURLs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserAvatarURLsSkipsQueryForEmptyIDs(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := newUserRepositoryWithSQL(nil, db)

	avatarURLs, err := repo.GetUserAvatarURLs(context.Background(), []int64{0, -1})

	require.NoError(t, err)
	require.Empty(t, avatarURLs)
	require.NoError(t, mock.ExpectationsWereMet())
}
