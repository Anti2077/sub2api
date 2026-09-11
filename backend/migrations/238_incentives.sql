-- Independent incentive state. Existing billing and daily lottery history remain authoritative.
CREATE TABLE IF NOT EXISTS incentive_campaigns (
    kind TEXT PRIMARY KEY CHECK (kind IN ('global_rate','lottery')),
    version BIGINT NOT NULL DEFAULT 1,
    config JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE IF NOT EXISTS incentive_campaign_periods (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL REFERENCES incentive_campaigns(kind),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    generation INTEGER NOT NULL DEFAULT 0,
    activated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    config_version BIGINT NOT NULL,
    config JSONB NOT NULL,
    spend NUMERIC(20,10) NOT NULL DEFAULT 0,
    rule_spend NUMERIC(20,10) NOT NULL DEFAULT 0,
    decrease NUMERIC(20,10) NOT NULL DEFAULT 0,
    rule_decrease NUMERIC(20,10) NOT NULL DEFAULT 0,
    closed_at TIMESTAMPTZ,
    UNIQUE(kind, starts_at, generation)
);
CREATE UNIQUE INDEX incentive_current_period ON incentive_campaign_periods(kind, starts_at) WHERE closed_at IS NULL;
CREATE TABLE IF NOT EXISTS incentive_group_rate_snapshots (
    period_id BIGINT NOT NULL REFERENCES incentive_campaign_periods(id),
    group_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    base_rate NUMERIC(20,10) NOT NULL,
    PRIMARY KEY(period_id,group_id)
);
CREATE TABLE IF NOT EXISTS incentive_user_progress (
    period_id BIGINT NOT NULL REFERENCES incentive_campaign_periods(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    spend NUMERIC(20,10) NOT NULL DEFAULT 0,
    rule_spend NUMERIC(20,10) NOT NULL DEFAULT 0,
    rule_earned BIGINT NOT NULL DEFAULT 0,
    earned BIGINT NOT NULL DEFAULT 0,
    used BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY(period_id,user_id),
    CHECK (used >= 0 AND earned >= used)
);
CREATE TABLE IF NOT EXISTS incentive_usage_events (
    period_id BIGINT NOT NULL REFERENCES incentive_campaign_periods(id),
    usage_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    amount NUMERIC(20,10) NOT NULL,
    config_version BIGINT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY(period_id,usage_id)
);
CREATE TABLE IF NOT EXISTS incentive_lottery_chance_ledger (
    id BIGSERIAL PRIMARY KEY,
    period_id BIGINT NOT NULL REFERENCES incentive_campaign_periods(id),
    user_id BIGINT NOT NULL,
    usage_id BIGINT NOT NULL,
    chances BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(period_id,usage_id)
);
CREATE TABLE IF NOT EXISTS incentive_rate_history (
    id BIGSERIAL PRIMARY KEY,
    period_id BIGINT NOT NULL REFERENCES incentive_campaign_periods(id),
    decrease NUMERIC(20,10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX incentive_rate_at ON incentive_rate_history(period_id,created_at DESC);
CREATE TABLE IF NOT EXISTS incentive_reward_ledger (
    id BIGSERIAL PRIMARY KEY,
    period_id BIGINT NOT NULL REFERENCES incentive_campaign_periods(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    request_key TEXT NOT NULL,
    prize JSONB NOT NULL,
    reward_amount NUMERIC(20,8) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(user_id,request_key)
);
CREATE TABLE IF NOT EXISTS incentive_config_history (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL,
    version BIGINT NOT NULL,
    config JSONB NOT NULL,
    actor_id BIGINT NOT NULL,
    action TEXT NOT NULL DEFAULT 'config',
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

-- Called while holding the campaign lock, including by the usage INSERT trigger.
CREATE OR REPLACE FUNCTION incentive_ensure_period(k TEXT, at_time TIMESTAMPTZ) RETURNS BIGINT LANGUAGE plpgsql AS $$
DECLARE c incentive_campaigns; p BIGINT; start_time TIMESTAMPTZ; end_time TIMESTAMPTZ; tz TEXT;
BEGIN
    SELECT * INTO c FROM incentive_campaigns WHERE kind=k FOR UPDATE;
    IF NOT FOUND THEN RETURN NULL; END IF;
    tz := c.config->>'timezone';
    start_time := date_trunc('week', at_time AT TIME ZONE tz) AT TIME ZONE tz;
    end_time := (date_trunc('week', at_time AT TIME ZONE tz) + INTERVAL '1 week') AT TIME ZONE tz;
    SELECT id INTO p FROM incentive_campaign_periods WHERE kind=k AND starts_at=start_time AND closed_at IS NULL;
    IF p IS NULL THEN
        UPDATE incentive_campaign_periods SET closed_at=ends_at WHERE kind=k AND ends_at<=start_time AND closed_at IS NULL;
        INSERT INTO incentive_campaign_periods(kind,starts_at,ends_at,generation,config_version,config)
        SELECT k,start_time,end_time,COALESCE(MAX(generation)+1,0),c.version,c.config
        FROM incentive_campaign_periods WHERE kind=k AND starts_at=start_time RETURNING id INTO p;
        INSERT INTO incentive_group_rate_snapshots(period_id,group_id,name,base_rate)
        SELECT p,id,name,rate_multiplier FROM groups
        WHERE id IN (SELECT jsonb_array_elements_text(c.config->'group_ids')::BIGINT);
    END IF;
    RETURN p;
END $$;

CREATE OR REPLACE FUNCTION incentive_record_usage() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE c incentive_campaigns; p BIGINT; old_spend NUMERIC; new_spend NUMERIC;
    old_earned BIGINT; new_earned BIGINT; previous_decrease NUMERIC; threshold NUMERIC; cap BIGINT; inserted INTEGER;
BEGIN
    IF NEW.actual_cost <= 0 OR NEW.group_id IS NULL OR NEW.billing_type <> 0 THEN RETURN NEW; END IF;
    FOR c IN SELECT * FROM incentive_campaigns ORDER BY kind FOR UPDATE LOOP
        IF NOT COALESCE((c.config->>'enabled')::BOOLEAN,false) THEN CONTINUE; END IF;
        IF NOT (c.config->'group_ids' @> to_jsonb(ARRAY[NEW.group_id])) THEN CONTINUE; END IF;
        IF c.config->'excluded_user_ids' @> to_jsonb(ARRAY[NEW.user_id]) THEN CONTINUE; END IF;
        IF c.config->'excluded_models' ? COALESCE(NULLIF(NEW.requested_model,''),NEW.model) THEN CONTINUE; END IF;
        IF (c.config->>'exclude_admins')::BOOLEAN AND EXISTS(SELECT 1 FROM users WHERE id=NEW.user_id AND role='admin') THEN CONTINUE; END IF;
        -- Never backfill usage predating activation/config edits or revive expired periods.
        IF NEW.created_at < c.updated_at THEN CONTINUE; END IF;
        p := incentive_ensure_period(c.kind,clock_timestamp());
        IF NEW.created_at < (SELECT CASE WHEN generation>0 THEN activated_at ELSE starts_at END FROM incentive_campaign_periods WHERE id=p) THEN CONTINUE; END IF;
        INSERT INTO incentive_usage_events(period_id,usage_id,user_id,amount,config_version)
        VALUES(p,NEW.id,NEW.user_id,NEW.actual_cost,c.version) ON CONFLICT DO NOTHING;
        GET DIAGNOSTICS inserted=ROW_COUNT;
        IF inserted=0 THEN CONTINUE; END IF;
        SELECT spend-rule_spend INTO old_spend FROM incentive_campaign_periods WHERE id=p;
        UPDATE incentive_campaign_periods SET spend=spend+NEW.actual_cost WHERE id=p RETURNING spend-rule_spend INTO new_spend;
        threshold := (c.config->>'spend_threshold')::NUMERIC;
        IF c.kind='global_rate' THEN
            IF floor(old_spend/threshold) <> floor(new_spend/threshold) THEN
                UPDATE incentive_campaign_periods SET decrease=rule_decrease+floor(new_spend/threshold)*(c.config->>'rate_decrease')::NUMERIC WHERE id=p RETURNING decrease INTO previous_decrease;
                INSERT INTO incentive_rate_history(period_id,decrease) VALUES(p,previous_decrease);
            END IF;
        END IF;
        INSERT INTO incentive_user_progress(period_id,user_id) VALUES(p,NEW.user_id) ON CONFLICT DO NOTHING;
        SELECT spend,earned INTO old_spend,old_earned FROM incentive_user_progress WHERE period_id=p AND user_id=NEW.user_id;
        new_spend := old_spend+NEW.actual_cost;
        new_earned := old_earned;
        IF c.kind='lottery' THEN
            SELECT rule_earned+floor((new_spend-rule_spend)/threshold) INTO new_earned FROM incentive_user_progress WHERE period_id=p AND user_id=NEW.user_id;
            cap := (c.config->>'max_chances')::BIGINT;
            IF cap>0 THEN new_earned:=LEAST(new_earned,cap); END IF;
            new_earned:=GREATEST(old_earned,new_earned);
            IF new_earned>old_earned THEN
                INSERT INTO incentive_lottery_chance_ledger(period_id,user_id,usage_id,chances)
                VALUES(p,NEW.user_id,NEW.id,new_earned-old_earned);
            END IF;
        END IF;
        UPDATE incentive_user_progress SET spend=new_spend,earned=new_earned WHERE period_id=p AND user_id=NEW.user_id;
    END LOOP;
    RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS incentive_usage_insert ON usage_logs;
CREATE TRIGGER incentive_usage_insert AFTER INSERT ON usage_logs FOR EACH ROW EXECUTE FUNCTION incentive_record_usage();

CREATE OR REPLACE VIEW incentive_lottery_balance_history AS
SELECT id,user_id,checkin_date,checked_in_at,drawn_at,prize_id,prize_name,reward_amount,created_at,updated_at FROM daily_lottery_entries
UNION ALL
SELECT -id,user_id,created_at::date,created_at,created_at,prize->>'id',prize->>'name',reward_amount,created_at,created_at FROM incentive_reward_ledger;
