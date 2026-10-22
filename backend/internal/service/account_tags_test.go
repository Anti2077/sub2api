package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountTagsNormalization(t *testing.T) {
	extra := map[string]any{AccountTagsExtraKey: []any{" Batch A ", "中文 标签", "Batch A", "", "batch a"}, "keep": true}
	require.NoError(t, NormalizeAccountTagsExtra(extra))
	require.Equal(t, []string{"Batch A", "中文 标签", "batch a"}, extra[AccountTagsExtraKey])
	require.Equal(t, true, extra["keep"])
	for _, raw := range []any{nil, "label", []any{"ok", 1}, []string{strings.Repeat("字", 129)}, []string{"line\nbreak"}, []string{"bad\x00"}} {
		require.Error(t, NormalizeAccountTagsExtra(map[string]any{AccountTagsExtraKey: raw}))
	}
	require.NoError(t, NormalizeAccountTagsExtra(nil))
	require.NoError(t, NormalizeAccountTagsExtra(map[string]any{AccountTagsExtraKey: []any{}}))
	tooMany := make([]string, 33)
	for i := range tooMany {
		tooMany[i] = strings.Repeat("x", i+1)
	}
	_, err := NormalizeAccountTags(tooMany)
	require.Error(t, err)
}

type accountTagRepoSpy struct {
	AccountFolderRepository
	ids       []int64
	tags      []string
	operation string
}

func (r *accountTagRepoSpy) UpdateTags(_ context.Context, ids []int64, tags []string, operation string) (int64, error) {
	r.ids, r.tags, r.operation = ids, tags, operation
	return int64(len(ids)), nil
}
func TestAccountTagsBulkValidation(t *testing.T) {
	repo := &accountTagRepoSpy{}
	svc := NewAccountFolderService(repo)
	count, err := svc.UpdateTags(context.Background(), []int64{1, 1, 2}, []string{" label ", "label"}, "add")
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Equal(t, []int64{1, 2}, repo.ids)
	require.Equal(t, []string{"label"}, repo.tags)
	for _, input := range []struct {
		ids       []int64
		tags      []string
		operation string
	}{
		{nil, []string{"label"}, "add"}, {[]int64{-1}, []string{"label"}, "remove"},
		{[]int64{1}, nil, "add"}, {[]int64{1}, []string{"label"}, "replace"},
	} {
		_, err = svc.UpdateTags(context.Background(), input.ids, input.tags, input.operation)
		require.Error(t, err)
	}
}

func TestAccountTagsValidatedByCreationAndExtraUpdates(t *testing.T) {
	_, err := buildAccountForCreate(&CreateAccountInput{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}, map[string]any{AccountTagsExtraKey: "bad"})
	require.Error(t, err)
	account, err := buildAccountForCreate(&CreateAccountInput{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}, map[string]any{AccountTagsExtraKey: []any{" batch ", "batch"}})
	require.NoError(t, err)
	require.Equal(t, []string{"batch"}, account.Extra[AccountTagsExtraKey])
	svc := &adminServiceImpl{}
	require.Error(t, svc.UpdateAccountExtra(context.Background(), 1, map[string]any{AccountTagsExtraKey: "bad"}))
	_, err = svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{AccountIDs: []int64{1}, Extra: map[string]any{AccountTagsExtraKey: "bad"}})
	require.Error(t, err)
}

func TestAccountTagsUpdateKeepsLabelsUnlessExplicitlyProvided(t *testing.T) {
	for _, clear := range []bool{false, true} {
		repo := &upstreamBillingProbeAdminRepo{upstreamBillingProbeAccountRepo: &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{
			10: {ID: 10, Name: "Tagged", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "test"}, Extra: map[string]any{AccountTagsExtraKey: []string{"keep"}}},
		}}}
		extra := map[string]any{"other": true}
		if clear {
			extra[AccountTagsExtraKey] = []any{}
		}
		account, err := (&adminServiceImpl{accountRepo: repo}).UpdateAccount(context.Background(), 10, &UpdateAccountInput{Extra: extra})
		require.NoError(t, err)
		if clear {
			require.Empty(t, account.Extra[AccountTagsExtraKey])
		} else {
			require.Equal(t, []string{"keep"}, account.Extra[AccountTagsExtraKey])
		}
	}
}

func TestAccountFolderRulesValidation(t *testing.T) {
	for _, rules := range []*AccountFolderRules{
		nil, {}, {TagMatch: "any"}, {Tags: []string{"one"}, TagMatch: "invalid"},
		{TagState: "untagged", Tags: []string{"contradiction"}}, {Types: []string{"bad"}},
		{Statuses: []string{"rate_limited"}}, {Expiry: "tomorrow"}, {Platforms: []string{""}},
	} {
		require.Error(t, rules.Validate())
	}
	disabled := false
	for _, rules := range []*AccountFolderRules{
		{Platforms: []string{"openai", "grok"}}, {Tags: []string{" a ", "a"}, TagMatch: "all"},
		{Name: " test "}, {TagState: "untagged"}, {Schedulable: &disabled},
		{Expiry: "expiring_7d"}, {Types: []string{"oauth"}, Statuses: []string{"active"}},
	} {
		require.NoError(t, rules.Validate())
	}
}
