-- Track the source and consumption state of incentive lottery chances.
-- Existing usage based rows remain consumption chances.
ALTER TABLE incentive_lottery_chance_ledger
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'consumption',
    ADD COLUMN IF NOT EXISTS source_key TEXT,
    ADD COLUMN IF NOT EXISTS used BIGINT NOT NULL DEFAULT 0;

UPDATE incentive_lottery_chance_ledger
SET source = 'consumption'
WHERE source IS NULL OR source = '';

CREATE INDEX IF NOT EXISTS incentive_lottery_chance_available
    ON incentive_lottery_chance_ledger(user_id, period_id, id)
    WHERE used < chances;

CREATE UNIQUE INDEX IF NOT EXISTS incentive_lottery_chance_source_key
    ON incentive_lottery_chance_ledger(period_id, user_id, source, source_key)
    WHERE source_key IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'incentive_lottery_chance_source_check'
    ) THEN
        ALTER TABLE incentive_lottery_chance_ledger
            ADD CONSTRAINT incentive_lottery_chance_source_check
            CHECK (source IN ('consumption', 'checkin'));
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'incentive_lottery_chance_used_check'
    ) THEN
        ALTER TABLE incentive_lottery_chance_ledger
            ADD CONSTRAINT incentive_lottery_chance_used_check
            CHECK (used >= 0 AND used <= chances);
    END IF;
END $$;
