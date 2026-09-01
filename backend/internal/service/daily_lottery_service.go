package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/google/uuid"
)

const (
	DailyLotteryConfigSettingKey = "daily_lottery_config"
	maxDailyLotteryPrizeCount    = 8
	maxDailyLotteryPrizeName     = 40
	maxDailyLotteryWeight        = int64(1_000_000_000)
	maxDailyLotteryReward        = 1_000_000.0
)

var (
	ErrDailyLotteryDisabled = infraerrors.Conflict(
		"DAILY_LOTTERY_DISABLED",
		"daily lottery is disabled",
	)
	ErrDailyLotteryNotCheckedIn = infraerrors.Conflict(
		"DAILY_LOTTERY_NOT_CHECKED_IN",
		"check in before drawing",
	)
	ErrDailyLotteryAlreadyDrawn = infraerrors.Conflict(
		"DAILY_LOTTERY_ALREADY_DRAWN",
		"today's lottery chance has already been used",
	)
)

type DailyLotteryPrize struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	RewardAmount float64 `json:"reward_amount"`
	Weight       int64   `json:"weight"`
	Enabled      bool    `json:"enabled"`
}

type DailyLotteryConfig struct {
	Enabled bool                `json:"enabled"`
	Prizes  []DailyLotteryPrize `json:"prizes"`
}

type DailyLotteryPrizeView struct {
	DailyLotteryPrize
	Probability float64 `json:"probability"`
}

type DailyLotteryEntry struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	CheckinDate  string     `json:"checkin_date"`
	CheckedInAt  time.Time  `json:"checked_in_at"`
	DrawnAt      *time.Time `json:"drawn_at,omitempty"`
	PrizeID      *string    `json:"prize_id,omitempty"`
	PrizeName    *string    `json:"prize_name,omitempty"`
	RewardAmount float64    `json:"reward_amount"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type DailyLotteryAdminEntry struct {
	DailyLotteryEntry
	UserEmail string `json:"user_email"`
	Username  string `json:"username"`
}

type DailyLotteryStatus struct {
	Enabled      bool                    `json:"enabled"`
	CheckedIn    bool                    `json:"checked_in"`
	CanDraw      bool                    `json:"can_draw"`
	AlreadyDrawn bool                    `json:"already_drawn"`
	Today        string                  `json:"today"`
	Timezone     string                  `json:"timezone"`
	NextResetAt  time.Time               `json:"next_reset_at"`
	Prizes       []DailyLotteryPrizeView `json:"prizes"`
	Entry        *DailyLotteryEntry      `json:"entry,omitempty"`
}

type DailyLotteryDrawResult struct {
	Entry      *DailyLotteryEntry `json:"entry"`
	OldBalance float64            `json:"old_balance"`
	NewBalance float64            `json:"new_balance"`
}

type DailyLotteryRepository interface {
	GetByDate(ctx context.Context, userID int64, checkinDate string) (*DailyLotteryEntry, error)
	CheckIn(ctx context.Context, userID int64, checkinDate string, now time.Time) (*DailyLotteryEntry, bool, error)
	MarkDrawn(ctx context.Context, userID int64, checkinDate string, prize DailyLotteryPrize, now time.Time) (*DailyLotteryEntry, bool, error)
	ListUserHistory(ctx context.Context, userID int64, limit int) ([]DailyLotteryEntry, error)
	ListAdminHistory(ctx context.Context, limit, offset int) ([]DailyLotteryAdminEntry, int64, error)
}

type dailyLotteryRandomInt func(maxExclusive int64) (int64, error)

type DailyLotteryService struct {
	repo        DailyLotteryRepository
	settingRepo SettingRepository
	userRepo    UserRepository
	entClient   *dbent.Client
	randomInt   dailyLotteryRandomInt
}

func NewDailyLotteryService(
	repo DailyLotteryRepository,
	settingRepo SettingRepository,
	userRepo UserRepository,
	entClient *dbent.Client,
) *DailyLotteryService {
	return &DailyLotteryService{
		repo:        repo,
		settingRepo: settingRepo,
		userRepo:    userRepo,
		entClient:   entClient,
		randomInt:   secureDailyLotteryRandomInt,
	}
}

func DefaultDailyLotteryConfig() DailyLotteryConfig {
	return DailyLotteryConfig{
		Enabled: false,
		Prizes: []DailyLotteryPrize{
			{ID: "grand-prize", Name: "特等奖", RewardAmount: 5, Weight: 5, Enabled: true},
			{ID: "first-prize", Name: "一等奖", RewardAmount: 1, Weight: 45, Enabled: true},
			{ID: "lucky-prize", Name: "幸运奖", RewardAmount: 0.1, Weight: 250, Enabled: true},
			{ID: "thanks", Name: "谢谢参与", RewardAmount: 0, Weight: 700, Enabled: true},
		},
	}
}

func (s *DailyLotteryService) GetConfig(ctx context.Context) (DailyLotteryConfig, error) {
	raw, err := s.settingRepo.GetValue(ctx, DailyLotteryConfigSettingKey)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultDailyLotteryConfig(), nil
		}
		return DailyLotteryConfig{}, fmt.Errorf("get daily lottery config: %w", err)
	}

	var cfg DailyLotteryConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DailyLotteryConfig{}, fmt.Errorf("decode daily lottery config: %w", err)
	}
	if err := validateDailyLotteryConfig(cfg); err != nil {
		return DailyLotteryConfig{}, fmt.Errorf("stored daily lottery config is invalid: %w", err)
	}
	return cfg, nil
}

func (s *DailyLotteryService) UpdateConfig(ctx context.Context, cfg DailyLotteryConfig) (DailyLotteryConfig, error) {
	for i := range cfg.Prizes {
		cfg.Prizes[i].ID = strings.TrimSpace(cfg.Prizes[i].ID)
		cfg.Prizes[i].Name = strings.TrimSpace(cfg.Prizes[i].Name)
		if cfg.Prizes[i].ID == "" {
			cfg.Prizes[i].ID = uuid.NewString()
		}
	}
	if err := validateDailyLotteryConfig(cfg); err != nil {
		return DailyLotteryConfig{}, err
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		return DailyLotteryConfig{}, fmt.Errorf("encode daily lottery config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, DailyLotteryConfigSettingKey, string(raw)); err != nil {
		return DailyLotteryConfig{}, fmt.Errorf("save daily lottery config: %w", err)
	}
	return cfg, nil
}

func (s *DailyLotteryService) GetStatus(ctx context.Context, userID int64) (*DailyLotteryStatus, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return s.buildStatus(ctx, userID, cfg)
}

func (s *DailyLotteryService) CheckIn(ctx context.Context, userID int64) (*DailyLotteryStatus, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrDailyLotteryDisabled
	}

	now := timezone.Now()
	date := now.Format("2006-01-02")
	if _, _, err := s.repo.CheckIn(ctx, userID, date, now); err != nil {
		return nil, fmt.Errorf("daily lottery check in: %w", err)
	}
	return s.buildStatus(ctx, userID, cfg)
}

func (s *DailyLotteryService) Draw(ctx context.Context, userID int64) (*DailyLotteryDrawResult, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrDailyLotteryDisabled
	}

	prize, err := s.selectPrize(cfg.Prizes)
	if err != nil {
		return nil, err
	}
	now := timezone.Now()
	date := now.Format("2006-01-02")

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin daily lottery transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	entry, updated, err := s.repo.MarkDrawn(txCtx, userID, date, prize, now)
	if err != nil {
		return nil, fmt.Errorf("record daily lottery draw: %w", err)
	}
	if !updated {
		current, getErr := s.repo.GetByDate(txCtx, userID, date)
		if getErr != nil {
			return nil, fmt.Errorf("read daily lottery state: %w", getErr)
		}
		if current == nil {
			return nil, ErrDailyLotteryNotCheckedIn
		}
		return nil, ErrDailyLotteryAlreadyDrawn
	}

	change, err := s.userRepo.AdjustBalance(txCtx, userID, prize.RewardAmount)
	if err != nil {
		return nil, fmt.Errorf("credit daily lottery reward: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit daily lottery transaction: %w", err)
	}

	return &DailyLotteryDrawResult{
		Entry:      entry,
		OldBalance: change.Old,
		NewBalance: change.New,
	}, nil
}

func (s *DailyLotteryService) ListUserHistory(ctx context.Context, userID int64, limit int) ([]DailyLotteryEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return s.repo.ListUserHistory(ctx, userID, limit)
}

func (s *DailyLotteryService) ListAdminHistory(ctx context.Context, page, pageSize int) ([]DailyLotteryAdminEntry, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListAdminHistory(ctx, pageSize, (page-1)*pageSize)
}

func (s *DailyLotteryService) buildStatus(ctx context.Context, userID int64, cfg DailyLotteryConfig) (*DailyLotteryStatus, error) {
	today := timezone.Today()
	date := today.Format("2006-01-02")
	entry, err := s.repo.GetByDate(ctx, userID, date)
	if err != nil {
		return nil, fmt.Errorf("get daily lottery status: %w", err)
	}
	drawn := entry != nil && entry.DrawnAt != nil
	checkedIn := entry != nil
	return &DailyLotteryStatus{
		Enabled:      cfg.Enabled,
		CheckedIn:    checkedIn,
		CanDraw:      cfg.Enabled && checkedIn && !drawn,
		AlreadyDrawn: drawn,
		Today:        date,
		Timezone:     timezone.Name(),
		NextResetAt:  today.AddDate(0, 0, 1),
		Prizes:       dailyLotteryPrizeViews(cfg.Prizes),
		Entry:        entry,
	}, nil
}

func (s *DailyLotteryService) selectPrize(prizes []DailyLotteryPrize) (DailyLotteryPrize, error) {
	enabled := make([]DailyLotteryPrize, 0, len(prizes))
	var total int64
	for _, prize := range prizes {
		if !prize.Enabled {
			continue
		}
		enabled = append(enabled, prize)
		total += prize.Weight
	}
	if total <= 0 || len(enabled) == 0 {
		return DailyLotteryPrize{}, infraerrors.BadRequest("DAILY_LOTTERY_NO_PRIZES", "no enabled daily lottery prizes")
	}

	pick, err := s.randomInt(total)
	if err != nil {
		return DailyLotteryPrize{}, fmt.Errorf("generate daily lottery random value: %w", err)
	}
	for _, prize := range enabled {
		if pick < prize.Weight {
			return prize, nil
		}
		pick -= prize.Weight
	}
	return DailyLotteryPrize{}, errors.New("daily lottery selection fell outside configured weights")
}

func secureDailyLotteryRandomInt(maxExclusive int64) (int64, error) {
	if maxExclusive <= 0 {
		return 0, errors.New("random upper bound must be positive")
	}
	v, err := rand.Int(rand.Reader, big.NewInt(maxExclusive))
	if err != nil {
		return 0, err
	}
	return v.Int64(), nil
}

func validateDailyLotteryConfig(cfg DailyLotteryConfig) error {
	if len(cfg.Prizes) < 2 || len(cfg.Prizes) > maxDailyLotteryPrizeCount {
		return infraerrors.BadRequest("DAILY_LOTTERY_INVALID_PRIZES", "configure between 2 and 8 prize levels")
	}

	seen := make(map[string]struct{}, len(cfg.Prizes))
	enabledCount := 0
	var totalWeight int64
	for _, prize := range cfg.Prizes {
		if prize.ID == "" || len(prize.ID) > 64 {
			return infraerrors.BadRequest("DAILY_LOTTERY_INVALID_PRIZE_ID", "each prize must have a valid id")
		}
		if _, exists := seen[prize.ID]; exists {
			return infraerrors.BadRequest("DAILY_LOTTERY_DUPLICATE_PRIZE_ID", "prize ids must be unique")
		}
		seen[prize.ID] = struct{}{}
		if prize.Name == "" || utf8.RuneCountInString(prize.Name) > maxDailyLotteryPrizeName {
			return infraerrors.BadRequest("DAILY_LOTTERY_INVALID_PRIZE_NAME", "prize names must contain 1 to 40 characters")
		}
		if math.IsNaN(prize.RewardAmount) || math.IsInf(prize.RewardAmount, 0) || prize.RewardAmount < 0 || prize.RewardAmount > maxDailyLotteryReward {
			return infraerrors.BadRequest("DAILY_LOTTERY_INVALID_REWARD", "prize reward amounts must be between 0 and 1000000")
		}
		if prize.Weight <= 0 || prize.Weight > maxDailyLotteryWeight {
			return infraerrors.BadRequest("DAILY_LOTTERY_INVALID_WEIGHT", "prize weights must be positive integers no greater than 1000000000")
		}
		if totalWeight > maxDailyLotteryWeight-prize.Weight {
			return infraerrors.BadRequest("DAILY_LOTTERY_WEIGHT_OVERFLOW", "total prize weight must not exceed 1000000000")
		}
		totalWeight += prize.Weight
		if prize.Enabled {
			enabledCount++
		}
	}
	if enabledCount == 0 {
		return infraerrors.BadRequest("DAILY_LOTTERY_NO_ENABLED_PRIZES", "at least one prize level must be enabled")
	}
	return nil
}

func dailyLotteryPrizeViews(prizes []DailyLotteryPrize) []DailyLotteryPrizeView {
	var total int64
	for _, prize := range prizes {
		if prize.Enabled {
			total += prize.Weight
		}
	}
	views := make([]DailyLotteryPrizeView, 0, len(prizes))
	for _, prize := range prizes {
		if !prize.Enabled {
			continue
		}
		probability := 0.0
		if total > 0 {
			probability = float64(prize.Weight) / float64(total)
		}
		views = append(views, DailyLotteryPrizeView{DailyLotteryPrize: prize, Probability: probability})
	}
	return views
}
