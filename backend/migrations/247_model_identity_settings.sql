-- Snapshot probe concurrency when a run is claimed, shared across replicas.
-- Existing runs keep their previous ten-probe limit until they finish.
ALTER TABLE account_identity_runs ADD COLUMN probe_concurrency INTEGER NOT NULL DEFAULT 10
    CHECK (probe_concurrency BETWEEN 1 AND 10);
ALTER TABLE account_identity_runs ALTER COLUMN probe_concurrency SET DEFAULT 4;
