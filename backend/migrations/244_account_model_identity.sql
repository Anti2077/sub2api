CREATE TABLE account_identity_configs (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    group_id BIGINT NOT NULL REFERENCES groups(id),
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE account_identity_plans (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES account_identity_configs(account_id) ON DELETE CASCADE,
    request_model TEXT NOT NULL,
    expected_model TEXT NOT NULL,
    interval_minutes INTEGER NOT NULL DEFAULT 120 CHECK (interval_minutes BETWEEN 15 AND 10080),
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX account_identity_due ON account_identity_plans(next_run_at) WHERE enabled;
CREATE TABLE account_identity_runs (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES account_identity_plans(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    api_key_id BIGINT NOT NULL,
    request_model TEXT NOT NULL,
    expected_model TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    token_hash TEXT,
    worker_slot INTEGER CHECK (worker_slot IN (1, 2)),
    lease_until TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    report JSONB NOT NULL DEFAULT '{}',
    probes JSONB NOT NULL DEFAULT '[]',
    probe_inflight INTEGER NOT NULL DEFAULT 0 CHECK (probe_inflight BETWEEN 0 AND 4)
);
CREATE UNIQUE INDEX account_identity_one_active_account ON account_identity_runs(account_id) WHERE status IN ('running', 'cancelling');
CREATE UNIQUE INDEX account_identity_one_pending_plan ON account_identity_runs(plan_id) WHERE status IN ('queued', 'running', 'cancelling');
CREATE UNIQUE INDEX account_identity_worker_slot ON account_identity_runs(worker_slot) WHERE status IN ('running', 'cancelling');
CREATE INDEX account_identity_history ON account_identity_runs(plan_id, id DESC);
