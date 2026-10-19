-- Preserve an existing shared test user. Ambiguous installations must choose
-- one explicitly; neither account IDs nor production user IDs are hardcoded.
INSERT INTO settings(key,value,updated_at)
SELECT 'model_identity_settings',jsonb_build_object(
  'public_base_url',COALESCE((SELECT value FROM settings WHERE key='model_identity_public_base_url'),''),
  'concurrent_accounts',1,'probe_concurrency',4)::text,NOW()
ON CONFLICT(key) DO NOTHING;

UPDATE settings SET value=(value::jsonb || jsonb_build_object('test_user_id',
  COALESCE((SELECT MIN(user_id) FROM account_identity_configs HAVING COUNT(DISTINCT user_id)=1),0)))::text,
  updated_at=NOW()
WHERE key='model_identity_settings' AND NOT (value::jsonb ? 'test_user_id');
