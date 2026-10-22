//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountFolderHierarchyAndDeleteIntegration(t *testing.T) {
	ctx := context.Background()
	repo := &accountFolderRepository{db: integrationDB}
	suffix := fmt.Sprint(time.Now().UnixNano())
	rootRules := &service.AccountFolderRules{Tags: []string{"root-" + suffix}}
	childRules := &service.AccountFolderRules{Tags: []string{"child-" + suffix}}
	root, err := repo.Save(ctx, 0, "folder-root-"+suffix, nil, rootRules)
	require.NoError(t, err)
	defer integrationDB.ExecContext(ctx, `DELETE FROM account_folders WHERE id=$1`, root.ID)
	child, err := repo.Save(ctx, 0, "folder-child-"+suffix, &root.ID, childRules)
	require.NoError(t, err)
	_, err = repo.Save(ctx, 0, "folder-grandchild-"+suffix, &child.ID, childRules)
	require.Error(t, err)
	_, err = repo.Save(ctx, root.ID, root.Name, &child.ID, rootRules)
	require.Error(t, err)
	_, err = repo.Save(ctx, 0, root.Name, nil, rootRules)
	require.Error(t, err)

	ids := make([]int64, 3)
	for i := range ids {
		err = integrationDB.QueryRowContext(ctx, `INSERT INTO accounts(name,platform,type,credentials,status) VALUES($1,'openai','apikey','{}',$2) RETURNING id`, "folder-account-"+suffix+fmt.Sprint(i), []string{"active", "error", "active"}[i]).Scan(&ids[i])
		require.NoError(t, err)
		defer integrationDB.ExecContext(ctx, `DELETE FROM accounts WHERE id=$1`, ids[i])
	}
	_, err = repo.UpdateTags(ctx, ids[:1], rootRules.Tags, "add")
	require.NoError(t, err)
	_, err = repo.UpdateTags(ctx, ids[1:2], childRules.Tags, "add")
	require.NoError(t, err)
	listRepo := newAccountRepositoryWithSQL(integrationEntClient, integrationDB, nil)
	scoped, err := service.WithAccountFolderFilter(ctx, fmt.Sprint(root.ID))
	require.NoError(t, err)
	accounts, page, err := listRepo.ListWithFilters(scoped, pagination.PaginationParams{Page: 1, PageSize: 1}, "", "", "", "", 0, "")
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, int64(2), page.Total)
	overview, err := repo.Overview(ctx)
	require.NoError(t, err)
	for _, folder := range overview.Folders {
		if folder.ID == root.ID {
			require.Equal(t, int64(2), folder.AccountCount)
			require.Equal(t, int64(1), folder.ErrorCount)
		}
	}
	// Overlapping child rules must not multiply matches in the parent scope.
	_, err = repo.Save(ctx, 0, "folder-second-child-"+suffix, &root.ID, childRules)
	require.NoError(t, err)
	overview, err = repo.Overview(ctx)
	require.NoError(t, err)
	for _, folder := range overview.Folders {
		if folder.ID == root.ID {
			require.Equal(t, int64(2), folder.AccountCount)
		}
	}

	// A matching account appears automatically in several folders without moving it.
	_, err = repo.UpdateTags(ctx, ids[:1], childRules.Tags, "add")
	require.NoError(t, err)
	childScope, err := service.WithAccountFolderFilter(ctx, fmt.Sprint(child.ID))
	require.NoError(t, err)
	_, childPage, err := listRepo.ListWithFilters(childScope, pagination.PaginationParams{Page: 1, PageSize: 10}, "", "", "", "", 0, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), childPage.Total)
	_, err = repo.UpdateTags(ctx, ids[:1], childRules.Tags, "remove")
	require.NoError(t, err)
	_, childPage, err = listRepo.ListWithFilters(childScope, pagination.PaginationParams{Page: 1, PageSize: 10}, "", "", "", "", 0, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), childPage.Total)

	require.NoError(t, repo.Delete(ctx, root.ID))
	var members int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM account_folder_members WHERE account_id=ANY($1::bigint[])`, fmt.Sprintf("{%d,%d}", ids[0], ids[1])).Scan(&members))
	require.Zero(t, members)
	var kept int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts WHERE name LIKE $1`, "folder-account-"+suffix+"%").Scan(&kept))
	require.Equal(t, 3, kept)
	var keptTags string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT extra->>'account_tags' FROM accounts WHERE id=$1`, ids[0]).Scan(&keptTags))
	require.Contains(t, keptTags, rootRules.Tags[0])
	accounts, page, err = listRepo.ListWithFilters(scoped, pagination.PaginationParams{Page: 1, PageSize: 10}, "", "", "", "", 0, "")
	require.NoError(t, err)
	require.Empty(t, accounts)
	require.Zero(t, page.Total)
}
