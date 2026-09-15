-- Fix first-period creation for incentive campaigns.
-- The original function selected from the current-period rows while inserting,
-- which produced no row when a campaign had never created a period yet.
CREATE OR REPLACE FUNCTION incentive_ensure_period(k TEXT, at_time TIMESTAMPTZ) RETURNS BIGINT LANGUAGE plpgsql AS $$
DECLARE
    c incentive_campaigns;
    p BIGINT;
    start_time TIMESTAMPTZ;
    end_time TIMESTAMPTZ;
    tz TEXT;
    next_generation INTEGER;
BEGIN
    SELECT * INTO c FROM incentive_campaigns WHERE kind=k FOR UPDATE;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    tz := NULLIF(c.config->>'timezone', '');
    IF tz IS NULL THEN
        tz := 'UTC';
    END IF;
    start_time := date_trunc('week', at_time AT TIME ZONE tz) AT TIME ZONE tz;
    end_time := (date_trunc('week', at_time AT TIME ZONE tz) + INTERVAL '1 week') AT TIME ZONE tz;

    SELECT id INTO p
    FROM incentive_campaign_periods
    WHERE kind=k AND starts_at=start_time AND closed_at IS NULL;

    IF p IS NULL THEN
        UPDATE incentive_campaign_periods
        SET closed_at=ends_at
        WHERE kind=k AND ends_at<=start_time AND closed_at IS NULL;

        SELECT COALESCE(MAX(generation)+1, 0)
        INTO next_generation
        FROM incentive_campaign_periods
        WHERE kind=k;

        INSERT INTO incentive_campaign_periods(
            kind, starts_at, ends_at, generation, config_version, config
        )
        VALUES(k, start_time, end_time, next_generation, c.version, c.config)
        RETURNING id INTO p;

        INSERT INTO incentive_group_rate_snapshots(period_id, group_id, name, base_rate)
        SELECT p, id, name, rate_multiplier
        FROM groups
        WHERE id IN (SELECT jsonb_array_elements_text(c.config->'group_ids')::BIGINT);
    END IF;

    RETURN p;
END $$;
