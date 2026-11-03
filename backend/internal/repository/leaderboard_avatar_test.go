//go:build unit

package repository

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetLeaderboardAvatarReferencesProjectsOnlyMetadataAndChecksVisibility(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := newUserRepositoryWithSQL(nil, db)
	mock.ExpectQuery(`(?s)SELECT a.user_id,\s+CASE WHEN a.url LIKE 'data:%' THEN '' ELSE trim\(a.url\) END,\s+CASE WHEN a.url LIKE 'data:%' THEN md5\(a.url\) ELSE '' END.*u.deleted_at IS NULL.*u.username_confirmed AND NOT u.leaderboard_anonymous`).
		WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"user_id", "url", "version"}).AddRow(7, "", "version").AddRow(9, "https://example.com/a.png", ""))
	refs, err := repo.GetLeaderboardAvatarReferences(context.Background(), []int64{7, 9})
	require.NoError(t, err)
	require.Equal(t, service.LeaderboardAvatarReference{Version: "version"}, refs[7])
	require.Equal(t, service.LeaderboardAvatarReference{URL: "https://example.com/a.png"}, refs[9])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLeaderboardAvatarReferencesEmptyIDsSkipsDatabase(t *testing.T) {
	db, mock := newSQLMock(t)
	refs, err := newUserRepositoryWithSQL(nil, db).GetLeaderboardAvatarReferences(context.Background(), []int64{-1, 0})
	require.NoError(t, err)
	require.Empty(t, refs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLeaderboardAvatarChecksExactVersionAndCurrentPrivacy(t *testing.T) {
	for _, exists := range []bool{false, true} {
		db, mock := newSQLMock(t)
		rows := sqlmock.NewRows([]string{"url"})
		if exists {
			rows.AddRow("data:image/gif;base64,R0lGODlh")
		}
		mock.ExpectQuery(`(?s)WHERE u.username = \$1 AND md5\(a.url\) = \$2 AND a.url LIKE 'data:%'.*u.deleted_at IS NULL AND u.username_confirmed AND NOT u.leaderboard_anonymous`).
			WithArgs("alice", "version").WillReturnRows(rows)
		avatar, err := newUserRepositoryWithSQL(nil, db).GetLeaderboardAvatar(context.Background(), "alice", "version")
		require.NoError(t, err)
		if exists {
			require.Equal(t, "data:image/gif;base64,R0lGODlh", avatar.URL)
		} else {
			require.Nil(t, avatar)
		}
		require.NoError(t, mock.ExpectationsWereMet())
	}
}
