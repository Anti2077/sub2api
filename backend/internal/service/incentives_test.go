package service

import (
	"context"
	"encoding/json"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
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
func TestIncentiveRatePreservesExclusionsAndFloor(t *testing.T) {
	for _, tt := range []struct {
		name    string
		drop    float64
		admin   bool
		exclude bool
		enabled bool
		want    float64
	}{{"tier", .02, false, false, true, .43}, {"floor", .5, false, false, true, .35}, {"admin", .02, true, false, true, .45}, {"excluded", .02, false, true, true, .45}, {"disabled", .02, false, false, false, .45}} {
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
