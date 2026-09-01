package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type dailyLotterySettingRepoStub struct {
	mu     sync.Mutex
	values map[string]string
}

func newDailyLotterySettingRepoStub() *dailyLotterySettingRepoStub {
	return &dailyLotterySettingRepoStub{values: make(map[string]string)}
}

func (s *dailyLotterySettingRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	value, err := s.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *dailyLotterySettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *dailyLotterySettingRepoStub) Set(_ context.Context, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *dailyLotterySettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, err := s.GetValue(context.Background(), key); err == nil {
			result[key] = value
		}
	}
	return result, nil
}

func (s *dailyLotterySettingRepoStub) SetMultiple(ctx context.Context, values map[string]string) error {
	for key, value := range values {
		if err := s.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *dailyLotterySettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *dailyLotterySettingRepoStub) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}

type dailyLotteryRepoStub struct {
	mu      sync.Mutex
	entries map[string]*DailyLotteryEntry
}

func newDailyLotteryRepoStub() *dailyLotteryRepoStub {
	return &dailyLotteryRepoStub{entries: make(map[string]*DailyLotteryEntry)}
}

func dailyLotteryEntryKey(userID int64, date string) string {
	return date + "/" + time.Unix(userID, 0).UTC().Format(time.RFC3339Nano)
}

func cloneDailyLotteryEntry(entry *DailyLotteryEntry) *DailyLotteryEntry {
	if entry == nil {
		return nil
	}
	clone := *entry
	return &clone
}

func (r *dailyLotteryRepoStub) GetByDate(_ context.Context, userID int64, date string) (*DailyLotteryEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneDailyLotteryEntry(r.entries[dailyLotteryEntryKey(userID, date)]), nil
}

func (r *dailyLotteryRepoStub) CheckIn(_ context.Context, userID int64, date string, now time.Time) (*DailyLotteryEntry, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := dailyLotteryEntryKey(userID, date)
	if current := r.entries[key]; current != nil {
		return cloneDailyLotteryEntry(current), false, nil
	}
	entry := &DailyLotteryEntry{ID: int64(len(r.entries) + 1), UserID: userID, CheckinDate: date, CheckedInAt: now, CreatedAt: now, UpdatedAt: now}
	r.entries[key] = entry
	return cloneDailyLotteryEntry(entry), true, nil
}

func (r *dailyLotteryRepoStub) MarkDrawn(_ context.Context, _ int64, _ string, _ DailyLotteryPrize, _ time.Time) (*DailyLotteryEntry, bool, error) {
	panic("not used")
}

func (r *dailyLotteryRepoStub) ListUserHistory(_ context.Context, _ int64, _ int) ([]DailyLotteryEntry, error) {
	return nil, nil
}

func (r *dailyLotteryRepoStub) ListAdminHistory(_ context.Context, _, _ int) ([]DailyLotteryAdminEntry, int64, error) {
	return nil, 0, nil
}

func TestDefaultDailyLotteryConfigIsSafeAndValid(t *testing.T) {
	cfg := DefaultDailyLotteryConfig()
	require.False(t, cfg.Enabled, "existing installations must opt in before rewards can be issued")
	require.NoError(t, validateDailyLotteryConfig(cfg))
	require.Len(t, cfg.Prizes, 4)
}

func TestDailyLotteryConfigValidation(t *testing.T) {
	base := DefaultDailyLotteryConfig()

	tests := []struct {
		name   string
		mutate func(*DailyLotteryConfig)
	}{
		{name: "too few prizes", mutate: func(cfg *DailyLotteryConfig) { cfg.Prizes = cfg.Prizes[:1] }},
		{name: "duplicate ids", mutate: func(cfg *DailyLotteryConfig) { cfg.Prizes[1].ID = cfg.Prizes[0].ID }},
		{name: "empty name", mutate: func(cfg *DailyLotteryConfig) { cfg.Prizes[0].Name = "" }},
		{name: "negative reward", mutate: func(cfg *DailyLotteryConfig) { cfg.Prizes[0].RewardAmount = -1 }},
		{name: "zero weight", mutate: func(cfg *DailyLotteryConfig) { cfg.Prizes[0].Weight = 0 }},
		{name: "all disabled", mutate: func(cfg *DailyLotteryConfig) {
			for i := range cfg.Prizes {
				cfg.Prizes[i].Enabled = false
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.Prizes = append([]DailyLotteryPrize(nil), base.Prizes...)
			tt.mutate(&cfg)
			require.Error(t, validateDailyLotteryConfig(cfg))
		})
	}
}

func TestDailyLotterySelectPrizeUsesWeightedBoundaries(t *testing.T) {
	service := &DailyLotteryService{}
	prizes := []DailyLotteryPrize{
		{ID: "a", Name: "A", Weight: 1, Enabled: true},
		{ID: "disabled", Name: "Disabled", Weight: 1000, Enabled: false},
		{ID: "b", Name: "B", Weight: 3, Enabled: true},
	}

	for _, tc := range []struct {
		pick int64
		want string
	}{{0, "a"}, {1, "b"}, {3, "b"}} {
		service.randomInt = func(maxExclusive int64) (int64, error) {
			require.Equal(t, int64(4), maxExclusive)
			return tc.pick, nil
		}
		got, err := service.selectPrize(prizes)
		require.NoError(t, err)
		require.Equal(t, tc.want, got.ID)
	}
}

func TestDailyLotteryUpdateConfigFillsIDsAndPersists(t *testing.T) {
	settings := newDailyLotterySettingRepoStub()
	service := &DailyLotteryService{settingRepo: settings}
	cfg := DefaultDailyLotteryConfig()
	cfg.Prizes[0].ID = ""
	cfg.Prizes[0].Name = "  特等奖  "

	updated, err := service.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)
	require.NotEmpty(t, updated.Prizes[0].ID)
	require.Equal(t, "特等奖", updated.Prizes[0].Name)

	loaded, err := service.GetConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, updated, loaded)
}

func TestDailyLotteryCheckInIsIdempotentUnderConcurrency(t *testing.T) {
	settings := newDailyLotterySettingRepoStub()
	repo := newDailyLotteryRepoStub()
	service := &DailyLotteryService{repo: repo, settingRepo: settings}
	cfg := DefaultDailyLotteryConfig()
	cfg.Enabled = true
	_, err := service.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	const workers = 32
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, checkErr := service.CheckIn(context.Background(), 42)
			if checkErr == nil && (!status.CheckedIn || !status.CanDraw) {
				checkErr = context.Canceled
			}
			errs <- checkErr
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Len(t, repo.entries, 1)
}

func TestSecureDailyLotteryRandomIntStaysInRange(t *testing.T) {
	for range 100 {
		value, err := secureDailyLotteryRandomInt(7)
		require.NoError(t, err)
		require.GreaterOrEqual(t, value, int64(0))
		require.Less(t, value, int64(7))
	}
}
