-- Expand the original default field without rewriting planted crop snapshots.
UPDATE reward_settings
SET config = jsonb_set(config, '{plots}', '12'::jsonb), version = version + 1
WHERE config->>'plots' = '6';
