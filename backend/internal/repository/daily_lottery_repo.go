package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type dailyLotteryRepository struct {
	client *dbent.Client
}

func NewDailyLotteryRepository(client *dbent.Client) service.DailyLotteryRepository {
	return &dailyLotteryRepository{client: client}
}

func (r *dailyLotteryRepository) GetByDate(ctx context.Context, userID int64, checkinDate string) (*service.DailyLotteryEntry, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
		SELECT id, user_id, checkin_date::text, checked_in_at, drawn_at,
		       prize_id, prize_name, reward_amount, created_at, updated_at
		FROM daily_lottery_entries
		WHERE user_id = $1 AND checkin_date = $2::date
	`, userID, checkinDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	entry, err := scanDailyLotteryEntry(rows)
	if err != nil {
		return nil, err
	}
	return entry, rows.Err()
}

func (r *dailyLotteryRepository) CheckIn(ctx context.Context, userID int64, checkinDate string, now time.Time) (*service.DailyLotteryEntry, bool, error) {
	result, err := clientFromContext(ctx, r.client).ExecContext(ctx, `
		INSERT INTO daily_lottery_entries (
			user_id, checkin_date, checked_in_at, created_at, updated_at
		) VALUES ($1, $2::date, $3, $3, $3)
		ON CONFLICT (user_id, checkin_date) DO NOTHING
	`, userID, checkinDate, now)
	if err != nil {
		return nil, false, err
	}
	created := false
	if affected, affectedErr := result.RowsAffected(); affectedErr == nil {
		created = affected == 1
	}
	entry, err := r.GetByDate(ctx, userID, checkinDate)
	return entry, created, err
}

func (r *dailyLotteryRepository) MarkDrawn(ctx context.Context, userID int64, checkinDate string, prize service.DailyLotteryPrize, now time.Time) (*service.DailyLotteryEntry, bool, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
		UPDATE daily_lottery_entries
		SET drawn_at = $3,
		    prize_id = $4,
		    prize_name = $5,
		    reward_amount = $6,
		    updated_at = $3
		WHERE user_id = $1 AND checkin_date = $2::date AND drawn_at IS NULL
		RETURNING id, user_id, checkin_date::text, checked_in_at, drawn_at,
		          prize_id, prize_name, reward_amount, created_at, updated_at
	`, userID, checkinDate, now, prize.ID, prize.Name, prize.RewardAmount)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
	entry, err := scanDailyLotteryEntry(rows)
	if err != nil {
		return nil, false, err
	}
	return entry, true, rows.Err()
}

func (r *dailyLotteryRepository) ListUserHistory(ctx context.Context, userID int64, limit int) ([]service.DailyLotteryEntry, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
		SELECT id, user_id, checkin_date::text, checked_in_at, drawn_at,
		       prize_id, prize_name, reward_amount, created_at, updated_at
		FROM daily_lottery_entries
		WHERE user_id = $1
		ORDER BY checkin_date DESC, id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]service.DailyLotteryEntry, 0)
	for rows.Next() {
		entry, err := scanDailyLotteryEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, rows.Err()
}

func (r *dailyLotteryRepository) ListAdminHistory(ctx context.Context, limit, offset int) ([]service.DailyLotteryAdminEntry, int64, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
		SELECT e.id, e.user_id, e.checkin_date::text, e.checked_in_at, e.drawn_at,
		       e.prize_id, e.prize_name, e.reward_amount, e.created_at, e.updated_at,
		       u.email, u.username, COUNT(*) OVER ()
		FROM daily_lottery_entries e
		JOIN users u ON u.id = e.user_id
		ORDER BY e.checkin_date DESC, e.id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]service.DailyLotteryAdminEntry, 0)
	var total int64
	for rows.Next() {
		entry := service.DailyLotteryEntry{}
		var drawnAt sql.NullTime
		var prizeID, prizeName sql.NullString
		var adminEntry service.DailyLotteryAdminEntry
		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.CheckinDate, &entry.CheckedInAt, &drawnAt,
			&prizeID, &prizeName, &entry.RewardAmount, &entry.CreatedAt, &entry.UpdatedAt,
			&adminEntry.UserEmail, &adminEntry.Username, &total,
		); err != nil {
			return nil, 0, err
		}
		applyDailyLotteryNulls(&entry, drawnAt, prizeID, prizeName)
		adminEntry.DailyLotteryEntry = entry
		entries = append(entries, adminEntry)
	}
	return entries, total, rows.Err()
}

type dailyLotteryScanner interface {
	Scan(dest ...any) error
}

func scanDailyLotteryEntry(scanner dailyLotteryScanner) (*service.DailyLotteryEntry, error) {
	entry := &service.DailyLotteryEntry{}
	var drawnAt sql.NullTime
	var prizeID, prizeName sql.NullString
	if err := scanner.Scan(
		&entry.ID, &entry.UserID, &entry.CheckinDate, &entry.CheckedInAt, &drawnAt,
		&prizeID, &prizeName, &entry.RewardAmount, &entry.CreatedAt, &entry.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan daily lottery entry: %w", err)
	}
	applyDailyLotteryNulls(entry, drawnAt, prizeID, prizeName)
	return entry, nil
}

func applyDailyLotteryNulls(entry *service.DailyLotteryEntry, drawnAt sql.NullTime, prizeID, prizeName sql.NullString) {
	if drawnAt.Valid {
		entry.DrawnAt = &drawnAt.Time
	}
	if prizeID.Valid {
		entry.PrizeID = &prizeID.String
	}
	if prizeName.Valid {
		entry.PrizeName = &prizeName.String
	}
}
