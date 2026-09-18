package service

import (
	"context"
	"encoding/json"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

func TestIncentiveConfigValidation(t *testing.T) {
	c := DefaultIncentiveConfig("lottery")
	require.NoError(t, validateIncentiveConfig(c))
	for _, mutate := range []func(*IncentiveConfig){func(c *IncentiveConfig) { c.SpendThreshold = 0 }, func(c *IncentiveConfig) { c.MinimumRate = math.NaN() }, func(c *IncentiveConfig) { c.Enabled = true }, func(c *IncentiveConfig) { c.MaxChances = -1 }, func(c *IncentiveConfig) { c.GroupIDs = []int64{1, 1} }, func(c *IncentiveConfig) { c.Prizes = nil }} {
		v := c
		mutate(&v)
		require.Error(t, validateIncentiveConfig(v))
	}
}
func TestIncentiveRatePreservesContributionExclusionsAndFloor(t *testing.T) {
	for _, tt := range []struct {
		name    string
		drop    float64
		admin   bool
		exclude bool
		enabled bool
		want    float64
	}{{"tier", .02, false, false, true, .43}, {"floor", .5, false, false, true, .35}, {"admin", .02, true, false, true, .43}, {"excluded", .02, false, true, true, .43}, {"disabled", .02, false, false, false, .45}} {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			c := DefaultIncentiveConfig("global_rate")
			c.Enabled = tt.enabled
			c.GroupIDs = []int64{2}
			if tt.exclude {
				c.ExcludedUserIDs = []int64{1}
			}
			raw, err := json.Marshal(c)
			require.NoError(t, err)
			mock.ExpectQuery("SELECT r.base_rate").WillReturnRows(sqlmock.NewRows([]string{"base_rate", "config", "decrease"}).AddRow(.45, raw, tt.drop))
			svc := &IncentiveService{client: client}
			user := &User{ID: 1}
			if tt.admin {
				user.Role = "admin"
			}
			require.InDelta(t, tt.want, svc.ApplyRate(context.Background(), user, &Group{ID: 2, RateMultiplier: .45}, "gpt", .45, time.Now()), 1e-9)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
func TestIncentiveDrawRetryReturnsReceiptWithoutDebit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	c := DefaultIncentiveConfig("lottery")
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT config FROM incentive_campaigns").WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(raw))
	mock.ExpectQuery("SELECT prize,reward_amount").WithArgs(int64(1), "same-request").WillReturnRows(sqlmock.NewRows([]string{"prize", "reward_amount"}).AddRow([]byte(`{"id":"won","name":"Prize"}`), 1))
	mock.ExpectRollback()
	svc := &IncentiveService{client: client}
	v, err := svc.Draw(context.Background(), 1, "same-request")
	require.NoError(t, err)
	require.Equal(t, float64(1), v["reward_amount"])
	require.NoError(t, mock.ExpectationsWereMet())
}

// The real PostgreSQL reproduction covers parameter type resolution. This test
// covers crediting exactly once and retaining the independent daily-check-in
// switch when consumption rewards are disabled.
func TestIncentiveCheckInChanceCredit(t *testing.T) {
	for _, inserted := range []bool{true, false} {
		t.Run(fmt.Sprint("inserted=", inserted), func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			settings := newDailyLotterySettingRepoStub()
			lottery := NewDailyLotteryService(newDailyLotteryRepoStub(), settings, nil, client)
			legacy := DefaultDailyLotteryConfig()
			legacy.Enabled = true
			_, err = lottery.UpdateConfig(context.Background(), legacy)
			require.NoError(t, err)
			config := DefaultIncentiveConfig("lottery") // Consumption rewards disabled.
			raw, err := json.Marshal(config)
			require.NoError(t, err)
			mock.ExpectBegin()
			mock.ExpectExec("SELECT pg_advisory_xact_lock").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery("SELECT config FROM incentive_campaigns").WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(raw))
			mock.ExpectQuery("SELECT incentive_ensure_period").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectExec("INSERT INTO incentive_user_progress").WithArgs(int64(1), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
			rows := sqlmock.NewRows([]string{"id"})
			if inserted {
				rows.AddRow(1)
			}
			mock.ExpectQuery("INSERT INTO incentive_lottery_chance_ledger").WithArgs(int64(1), int64(7), sqlmock.AnyArg(), sqlmock.AnyArg(), int64(0)).WillReturnRows(rows)
			if inserted {
				mock.ExpectExec("UPDATE incentive_user_progress SET earned=earned\\+1").WithArgs(int64(1), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectCommit()
			svc := &IncentiveService{client: client, lottery: lottery}
			status, err := svc.CheckIn(context.Background(), 7)
			require.NoError(t, err)
			require.True(t, status.Enabled)
			require.True(t, status.CheckedIn)
			require.Equal(t, inserted, status.ChanceAwarded)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
