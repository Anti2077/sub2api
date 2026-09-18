package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

// The effective rate is the lower of the current daily group rate and the
// accumulated activity target. Personal rates remain higher priority.
func TestIncentiveRateConflictRecordUsage(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		daily, decrease, want float64
		personal              *float64
		excluded, disabled    bool
	}{
		{name: "unchanged_daily", daily: .8, decrease: .1, want: .7},
		{name: "daily_below_benefit", daily: .6, decrease: .1, want: .6},
		{name: "daily_equal_benefit", daily: .7, decrease: .1, want: .7},
		{name: "daily_above_benefit", daily: .9, decrease: .1, want: .7},
		{name: "daily_below_floor", daily: .2, decrease: .1, want: .2},
		{name: "further_progress_with_changed_daily", daily: .6, decrease: .3, want: .5},
		{name: "benefit_floor", daily: .8, decrease: .6, want: .35},
		{name: "excluded_user_receives_benefit", daily: .8, decrease: .1, want: .7, excluded: true},
		{name: "disabled", daily: .8, decrease: .1, want: .8, disabled: true},
		{name: "personal_lower", daily: .8, decrease: .1, want: .5, personal: rateConflictFloat(.5)},
		{name: "personal_higher", daily: .8, decrease: .1, want: .9, personal: rateConflictFloat(.9)},
		{name: "personal_equal_daily", daily: .8, decrease: .1, want: .8, personal: rateConflictFloat(.8)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			cfg := DefaultIncentiveConfig("global_rate")
			cfg.Enabled = !tc.disabled
			cfg.GroupIDs = []int64{2}
			if tc.excluded {
				cfg.ExcludedUserIDs = []int64{7}
			}
			raw, err := json.Marshal(cfg)
			require.NoError(t, err)
			at := time.Now()
			if tc.personal == nil || *tc.personal == tc.daily {
				rows := sqlmock.NewRows([]string{"base_rate", "config", "decrease"})
				if tc.personal == nil {
					rows.AddRow(.8, raw, tc.decrease)
				}
				mock.ExpectQuery("SELECT r.base_rate").WithArgs(int64(2), int64(7), at).WillReturnRows(rows)
			}
			logs := &openAIRecordUsageLogRepoStub{inserted: true}
			billing := &openAIRecordUsageBillingRepoStub{}
			svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs, billing, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, &openAIUserGroupRateRepoStub{rate: tc.personal})
			svc.settingService = &SettingService{incentives: &IncentiveService{client: client}}
			group := &Group{ID: 2, RateMultiplier: tc.daily}
			err = svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
				Result: &OpenAIForwardResult{RequestID: "rate-conflict-" + tc.name, Model: "gpt-5.1", Usage: OpenAIUsage{InputTokens: 1000, OutputTokens: 100}, Duration: time.Second},
				APIKey: &APIKey{ID: 10, GroupID: i64p(2), Group: group}, User: &User{ID: 7}, Account: &Account{ID: 20, Type: AccountTypeAPIKey}, PricingAt: at,
			})
			require.NoError(t, err)
			require.NotNil(t, logs.lastLog)
			require.NotNil(t, billing.lastCmd)
			require.Positive(t, logs.lastLog.TotalCost)
			require.InDelta(t, tc.want, logs.lastLog.RateMultiplier, 1e-10)
			require.InDelta(t, logs.lastLog.TotalCost*tc.want, logs.lastLog.ActualCost, 1e-10)
			require.InDelta(t, logs.lastLog.ActualCost, billing.lastCmd.BalanceCost, 1e-10)
			require.NoError(t, mock.ExpectationsWereMet())
			t.Logf("daily=%.2f benefit=%.2f applied=%.2f debit=%.8f", tc.daily, .8-tc.decrease, logs.lastLog.RateMultiplier, billing.lastCmd.BalanceCost)
		})
	}
}
func rateConflictFloat(v float64) *float64 { return &v }
