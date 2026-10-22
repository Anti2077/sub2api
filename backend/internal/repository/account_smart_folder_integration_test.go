//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestSmartFolderConditionsAndAtomicTagsIntegration(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprint(time.Now().UnixNano())
	repo := &accountFolderRepository{db: integrationDB}
	listRepo := newAccountRepositoryWithSQL(integrationEntClient, integrationDB, nil)
	type fixture struct {
		platform, typ, status, name, extra, expiry string
		schedulable                                bool
	}
	fixtures := []fixture{
		{"openai", "apikey", "active", "Prod_%_" + suffix, `{"account_tags":["team A","batch"],"keep":true}`, "2 days", true},
		{"grok", "oauth", "error", "prod-" + suffix, `{"account_tags":["batch"]}`, "-1 day", false},
		{"openai", "oauth", "inactive", "staging-" + suffix, `{}`, "", true},
	}
	ids := make([]int64, len(fixtures))
	for i, f := range fixtures {
		require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO accounts(name,platform,type,credentials,status,extra,schedulable,expires_at) VALUES($1,$2,$3,'{}',$4,$5,$6,CASE WHEN $7='' THEN NULL ELSE NOW()+NULLIF($7,'')::interval END) RETURNING id`, f.name, f.platform, f.typ, f.status, f.extra, f.schedulable, f.expiry).Scan(&ids[i]))
		t.Cleanup(func() { _, _ = integrationDB.ExecContext(ctx, `DELETE FROM accounts WHERE id=$1`, ids[i]) })
	}
	enabled, disabled := true, false
	cases := []struct {
		name  string
		rules service.AccountFolderRules
		want  []int64
	}{
		{"platform OR", service.AccountFolderRules{Platforms: []string{"grok", "openai"}}, ids},
		{"all tags", service.AccountFolderRules{Tags: []string{"team A", "batch"}, TagMatch: "all"}, ids[:1]},
		{"any tags", service.AccountFolderRules{Tags: []string{"team A", "batch"}, TagMatch: "any"}, ids[:2]},
		{"case insensitive name", service.AccountFolderRules{Name: "PROD"}, ids[:2]},
		{"literal percent underscore", service.AccountFolderRules{Name: "_%_"}, ids[:1]},
		{"all fields AND", service.AccountFolderRules{Platforms: []string{"openai"}, Tags: []string{"batch"}, Types: []string{"apikey"}, Statuses: []string{"active"}, Name: "prod", Schedulable: &enabled, Expiry: "expiring_7d"}, ids[:1]},
		{"expired", service.AccountFolderRules{Expiry: "expired"}, ids[1:2]},
		{"no expiry", service.AccountFolderRules{Expiry: "never"}, ids[2:]},
		{"untagged", service.AccountFolderRules{TagState: "untagged"}, ids[2:]},
		{"tagged", service.AccountFolderRules{TagState: "tagged"}, ids[:2]},
		{"scheduling off", service.AccountFolderRules{Schedulable: &disabled}, ids[1:2]},
		{"no match", service.AccountFolderRules{Tags: []string{"missing-" + suffix}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			folder, err := repo.Save(ctx, 0, "rules-"+suffix+tc.name, nil, &tc.rules)
			require.NoError(t, err)
			defer repo.Delete(ctx, folder.ID)
			scope, err := service.WithAccountFolderFilter(ctx, fmt.Sprint(folder.ID))
			require.NoError(t, err)
			accounts, page, err := listRepo.ListWithFilters(scope, pagination.PaginationParams{Page: 1, PageSize: 100}, "", "", "", suffix, 0, "")
			require.NoError(t, err)
			require.Equal(t, int64(len(tc.want)), page.Total)
			got := []int64{}
			for _, a := range accounts {
				got = append(got, a.ID)
			}
			require.ElementsMatch(t, tc.want, got)
			// A separate ordinary filter intersects the saved rule before pagination.
			_, page, err = listRepo.ListWithFilters(scope, pagination.PaginationParams{Page: 1, PageSize: 1}, "grok", "", "", suffix, 0, "")
			require.NoError(t, err)
			var grokCount int64
			for _, id := range tc.want {
				if id == ids[1] {
					grokCount++
				}
			}
			require.Equal(t, grokCount, page.Total)
		})
	}

	_, err := repo.UpdateTags(ctx, ids[:2], []string{"added"}, "add")
	require.NoError(t, err)
	tags, err := repo.Tags(ctx)
	require.NoError(t, err)
	require.Contains(t, tags, service.AccountTag{Name: "added", AccountCount: 2})
	var stored map[string]any
	var raw []byte
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT extra FROM accounts WHERE id=$1`, ids[0]).Scan(&raw))
	require.NoError(t, json.Unmarshal(raw, &stored))
	require.Equal(t, true, stored["keep"])
	require.Contains(t, stored["account_tags"], "team A")
	_, err = repo.UpdateTags(ctx, ids[:2], []string{"batch"}, "remove")
	require.NoError(t, err)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT extra FROM accounts WHERE id=$1`, ids[0]).Scan(&raw))
	require.NoError(t, json.Unmarshal(raw, &stored))
	require.NotContains(t, stored["account_tags"], "batch")
	require.Contains(t, stored["account_tags"], "added")

	// One overflow rolls back changes to every target, including otherwise valid rows.
	full := []string{}
	for i := 0; i < 32; i++ {
		full = append(full, fmt.Sprint("full-", i))
	}
	payload, err := json.Marshal(full)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `UPDATE accounts SET extra=jsonb_set(extra,'{account_tags}',$2::jsonb) WHERE id=$1`, ids[1], string(payload))
	require.NoError(t, err)
	_, err = repo.UpdateTags(ctx, ids[:2], []string{"overflow"}, "add")
	require.Error(t, err)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT extra FROM accounts WHERE id=$1`, ids[0]).Scan(&raw))
	require.NotContains(t, string(raw), "overflow")
	_, err = repo.UpdateTags(ctx, []int64{ids[0], -1}, []string{"missing-target"}, "add")
	require.Error(t, err)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT extra FROM accounts WHERE id=$1`, ids[0]).Scan(&raw))
	require.NotContains(t, string(raw), "missing-target")

	// A newly saved rule immediately changes unmatched membership.
	unmatched, err := service.WithAccountFolderFilter(ctx, "unfiled")
	require.NoError(t, err)
	_, before, err := listRepo.ListWithFilters(unmatched, pagination.PaginationParams{Page: 1, PageSize: 100}, "", "", "", suffix, 0, "")
	require.NoError(t, err)
	folder, err := repo.Save(ctx, 0, "cover-"+suffix, nil, &service.AccountFolderRules{Name: suffix})
	require.NoError(t, err)
	defer repo.Delete(ctx, folder.ID)
	_, after, err := listRepo.ListWithFilters(unmatched, pagination.PaginationParams{Page: 1, PageSize: 100}, "", "", "", suffix, 0, "")
	require.NoError(t, err)
	require.Greater(t, before.Total, int64(0))
	require.Zero(t, after.Total)
}

func TestSmartFolderMigrationPreservesLegacyAssignmentsIntegration(t *testing.T) {
	ctx := context.Background()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	schema := fmt.Sprintf("smart_folder_migration_%d", time.Now().UnixNano())
	_, err = tx.ExecContext(ctx, `CREATE SCHEMA `+schema+`; SET LOCAL search_path TO `+schema)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `CREATE TABLE accounts(id BIGINT PRIMARY KEY,name TEXT,extra JSONB,deleted_at TIMESTAMPTZ);
 INSERT INTO accounts VALUES(1,'one','{"account_tags":["existing"],"keep":true}',NULL),(2,'two','{}',NULL),(3,'unfiled','{}',NULL);`)
	require.NoError(t, err)
	legacy, err := migrations.FS.ReadFile("246_account_folders.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(legacy))
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO account_folders(id,name,parent_id) VALUES(10,'Root',NULL),(11,'Same',10),(20,'Other',NULL),(21,'Same',20);
 INSERT INTO account_folder_members VALUES(1,11),(2,21);`)
	require.NoError(t, err)
	smart, err := migrations.FS.ReadFile("252_smart_account_folders.sql")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err = tx.ExecContext(ctx, string(smart))
		require.NoError(t, err)
	}
	var first, second string
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT extra::text FROM accounts WHERE id=1`).Scan(&first))
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT extra::text FROM accounts WHERE id=2`).Scan(&second))
	require.Contains(t, first, "existing")
	require.Contains(t, first, `"keep": true`)
	require.Equal(t, 1, strings.Count(first, "Same (#11)"))
	require.NotContains(t, first, "Same (#21)")
	require.Contains(t, second, "Same (#21)")
	var matches bool
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT account_matches_smart_folder('openai','apikey','active',a.name,a.extra,true,NULL,f.rules) FROM accounts a CROSS JOIN account_folders f WHERE a.id=1 AND f.id=11`).Scan(&matches))
	require.True(t, matches)
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT account_matches_smart_folder('openai','apikey','active',a.name,a.extra,true,NULL,f.rules) FROM accounts a CROSS JOIN account_folders f WHERE a.id=1 AND f.id=21`).Scan(&matches))
	require.False(t, matches)
}
