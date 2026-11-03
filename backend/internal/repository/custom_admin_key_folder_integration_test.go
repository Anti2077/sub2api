//go:build integration

package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestScopedAdminListIntersectsFolderGroupAndIdentityPermission(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprint(time.Now().UnixNano())
	email := "folder-" + suffix + "@example.com"
	groups := newGroupRepositoryWithSQL(integrationEntClient, integrationDB)
	groupIDs := make([]int64, 2)
	for i := range groupIDs {
		group := &service.Group{Name: fmt.Sprintf("scoped-folder-%s-%d", suffix, i), Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard}
		require.NoError(t, groups.Create(ctx, group))
		groupIDs[i] = group.ID
		t.Cleanup(func() { _, _ = integrationDB.ExecContext(ctx, `DELETE FROM groups WHERE id=$1`, group.ID) })
	}
	ids := make([]int64, 4)
	for i := range ids {
		require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO accounts(name,platform,type,credentials,status) VALUES($1,'openai','oauth',jsonb_build_object('email',$2::text,'access_token','fixture-token'),'active') RETURNING id`, fmt.Sprintf("scoped-folder-%s-%d", suffix, i), email).Scan(&ids[i]))
		t.Cleanup(func() { _, _ = integrationDB.ExecContext(ctx, `DELETE FROM accounts WHERE id=$1`, ids[i]) })
		groupID := groupIDs[0]
		if i == 2 {
			groupID = groupIDs[1]
		}
		_, err := integrationDB.ExecContext(ctx, `INSERT INTO account_groups(account_id,group_id,priority) VALUES($1,$2,50)`, ids[i], groupID)
		require.NoError(t, err)
	}
	folders := &accountFolderRepository{db: integrationDB}
	folder, err := folders.Save(ctx, 0, "scoped-folder-"+suffix, nil, &service.AccountFolderRules{Tags: []string{suffix}})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = integrationDB.ExecContext(ctx, `DELETE FROM account_folders WHERE id=$1`, folder.ID) })
	// Two matching rows, one in another group, and one outside the folder.
	_, err = folders.UpdateTags(ctx, ids[:3], []string{suffix}, "add")
	require.NoError(t, err)
	repo := newAccountRepositoryWithSQL(integrationEntClient, integrationDB, nil)
	for _, identity := range []bool{false, true} {
		t.Run(fmt.Sprintf("identity=%t", identity), func(t *testing.T) {
			permissions := []string{"accounts.read"}
			if identity {
				permissions = append(permissions, "accounts.identity.read")
			}
			principal := &adminaccess.Principal{Policy: adminaccess.Policy{Rules: []adminaccess.Rule{{Permissions: permissions, GroupIDs: groupIDs, AccountTypes: []string{"oauth"}, Platforms: []string{"openai"}}}}}
			scope, err := service.WithAccountFolderFilter(adminaccess.WithPrincipal(ctx, principal), fmt.Sprint(folder.ID))
			require.NoError(t, err)
			for page := 1; page <= 2; page++ {
				rows, total, err := repo.ScopedAdminList(scope, pagination.PaginationParams{Page: page, PageSize: 1}, "openai", "oauth", "", strings.ToUpper(email), groupIDs[0])
				require.NoError(t, err)
				if !identity {
					require.Zero(t, total, "email must not be searchable without identity permission")
					require.Empty(t, rows)
					continue
				}
				require.EqualValues(t, 2, total, "folder and selected group must intersect before pagination")
				require.Len(t, rows, 1)
				require.Equal(t, ids[page-1], rows[0].ID)
				require.Equal(t, email, rows[0].Credentials["email"])
				require.NotContains(t, rows[0].Credentials, "access_token")
			}
			// Name search remains available with either permission policy.
			rows, total, err := repo.ScopedAdminList(scope, pagination.PaginationParams{Page: 1, PageSize: 10}, "openai", "oauth", "", "scoped-folder-"+suffix, groupIDs[0])
			require.NoError(t, err)
			require.EqualValues(t, 2, total)
			require.Len(t, rows, 2)
		})
	}
}
