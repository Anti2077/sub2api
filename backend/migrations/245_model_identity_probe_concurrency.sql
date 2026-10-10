-- Keep the already-applied migration 244 checksum unchanged.
ALTER TABLE account_identity_runs DROP CONSTRAINT IF EXISTS account_identity_runs_probe_inflight_check;
ALTER TABLE account_identity_runs ADD CONSTRAINT account_identity_runs_probe_inflight_check
    CHECK (probe_inflight BETWEEN 0 AND 10);
