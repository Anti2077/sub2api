-- 每日签到抽奖记录。每个用户按服务器时区每天最多一条记录，
-- 抽奖结果保存奖项快照，后续修改配置不会改写历史。
CREATE TABLE IF NOT EXISTS daily_lottery_entries (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date DATE NOT NULL,
    checked_in_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    drawn_at TIMESTAMPTZ,
    prize_id VARCHAR(64),
    prize_name VARCHAR(100),
    reward_amount DECIMAL(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_daily_lottery_user_date UNIQUE (user_id, checkin_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_lottery_entries_user_history
    ON daily_lottery_entries (user_id, checkin_date DESC);

CREATE INDEX IF NOT EXISTS idx_daily_lottery_entries_draw_history
    ON daily_lottery_entries (drawn_at DESC)
    WHERE drawn_at IS NOT NULL;
