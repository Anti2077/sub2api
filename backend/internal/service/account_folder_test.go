package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountFolderScopeValidation(t *testing.T) {
	for _, value := range []string{"0", "-1", "abc", "1 OR 1=1"} {
		_, err := WithAccountFolderFilter(context.Background(), value)
		require.Error(t, err)
	}
	ctx, err := WithAccountFolderFilter(context.Background(), "unfiled")
	require.NoError(t, err)
	id, ok := AccountFolderFilterFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(-1), id)
	ctx, err = WithAccountFolderFilter(context.Background(), "123")
	require.NoError(t, err)
	id, ok = AccountFolderFilterFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(123), id)
}

type folderScopedBulkRepo struct {
	AccountRepository
	scope    int64
	hasScope bool
}

func (r *folderScopedBulkRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, typ, status, search string, groupID int64, privacy string) ([]Account, *pagination.PaginationResult, error) {
	r.scope, r.hasScope = AccountFolderFilterFromContext(ctx)
	return []Account{{ID: 10}, {ID: 20}}, &pagination.PaginationResult{Total: 2}, nil
}
func TestAccountFolderFilteredBulkEditKeepsScope(t *testing.T) {
	repo := &folderScopedBulkRepo{}
	svc := &adminServiceImpl{accountRepo: repo}
	ids, err := svc.resolveBulkUpdateTargetIDs(context.Background(), &BulkUpdateAccountFilters{Folder: "9"})
	require.NoError(t, err)
	require.Equal(t, []int64{10, 20}, ids)
	require.True(t, repo.hasScope)
	require.Equal(t, int64(9), repo.scope)
}
