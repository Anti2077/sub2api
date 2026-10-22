ALTER TABLE account_folders ADD COLUMN IF NOT EXISTS rules JSONB;

-- Keep each historical assignment as a unique, editable label. Including the
-- folder ID prevents identically named subfolders from merging accidentally.
UPDATE accounts a SET extra = COALESCE(a.extra, '{}'::jsonb) || jsonb_build_object(
    'account_tags', CASE WHEN jsonb_typeof(a.extra->'account_tags')='array' THEN a.extra->'account_tags' ELSE '[]'::jsonb END || jsonb_build_array(f.name || ' (#' || f.id || ')'))
FROM account_folder_members m JOIN account_folders f ON f.id=m.folder_id
WHERE a.id=m.account_id AND f.rules IS NULL;
UPDATE account_folders SET rules = jsonb_build_object('tags', jsonb_build_array(name || ' (#' || id || ')'), 'tag_match', 'all') WHERE rules IS NULL;
ALTER TABLE account_folders ALTER COLUMN rules SET NOT NULL;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='account_folders'::regclass AND conname='account_folders_rules_object') THEN
        ALTER TABLE account_folders ADD CONSTRAINT account_folders_rules_object CHECK (jsonb_typeof(rules) = 'object');
    END IF;
END $$;

-- Shared by list pagination, filtered bulk operations and overview counts.
-- Literal substring matching avoids treating '%' and '_' as wildcards.
CREATE OR REPLACE FUNCTION account_matches_smart_folder(
    a_platform TEXT, a_type TEXT, a_status TEXT, a_name TEXT, a_extra JSONB,
    a_schedulable BOOLEAN, a_expires_at TIMESTAMPTZ, rules JSONB
) RETURNS BOOLEAN LANGUAGE SQL STABLE PARALLEL SAFE AS $$
WITH labels AS (
    SELECT CASE WHEN jsonb_typeof(a_extra->'account_tags')='array'
                THEN a_extra->'account_tags' ELSE '[]'::jsonb END AS tags
)
SELECT
    (COALESCE(jsonb_array_length(rules->'platforms'),0)=0 OR (rules->'platforms') ? a_platform)
    AND (COALESCE(jsonb_array_length(rules->'types'),0)=0 OR (rules->'types') ? a_type)
    AND (COALESCE(jsonb_array_length(rules->'statuses'),0)=0 OR (rules->'statuses') ? a_status)
    AND (COALESCE(rules->>'name','')='' OR STRPOS(LOWER(a_name),LOWER(rules->>'name'))>0)
    AND (NOT (rules ? 'schedulable') OR a_schedulable=(rules->>'schedulable')::boolean)
    AND (CASE COALESCE(rules->>'tag_state','')
        WHEN 'tagged' THEN jsonb_array_length(labels.tags)>0
        WHEN 'untagged' THEN jsonb_array_length(labels.tags)=0 ELSE TRUE END)
    AND (COALESCE(jsonb_array_length(rules->'tags'),0)=0 OR CASE WHEN rules->>'tag_match'='any'
        THEN labels.tags ?| ARRAY(SELECT jsonb_array_elements_text(rules->'tags'))
        ELSE labels.tags @> (rules->'tags') END)
    AND (CASE COALESCE(rules->>'expiry','')
        WHEN 'expired' THEN a_expires_at <= NOW()
        WHEN 'expiring_7d' THEN a_expires_at > NOW() AND a_expires_at <= NOW()+INTERVAL '7 days'
        WHEN 'never' THEN a_expires_at IS NULL ELSE TRUE END)
FROM labels;
$$;

CREATE INDEX IF NOT EXISTS accounts_account_tags_gin ON accounts USING gin ((extra->'account_tags')) WHERE deleted_at IS NULL;
