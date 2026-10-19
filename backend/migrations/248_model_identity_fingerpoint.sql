-- Freeze engine/options at enqueue time. Legacy active work is interrupted;
-- the new engine must never continue or overwrite a BazaarLink report.
ALTER TABLE account_identity_plans ADD COLUMN options JSONB NOT NULL DEFAULT '{"protocol":"chat_completions","parallel":true,"relaxed":true,"stream":true,"tokenizer":false}';
ALTER TABLE account_identity_runs ADD COLUMN options JSONB NOT NULL DEFAULT '{}';
ALTER TABLE account_identity_runs ADD COLUMN engine_commit TEXT NOT NULL DEFAULT 'legacy-bazaarlink';
UPDATE account_identity_runs SET status='interrupted', finished_at=NOW(), token_hash=NULL, worker_slot=NULL, lease_until=NULL, probe_inflight=0
 WHERE status IN ('queued','running','cancelling');
ALTER TABLE account_identity_runs ALTER COLUMN engine_commit SET DEFAULT '654c8334bb53951aaf04cbd61252d163649e1369';
