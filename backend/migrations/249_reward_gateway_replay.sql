-- A claimed key is never forwarded twice, including after process restart.
-- Pending/oversize receipts deliberately remain non-replayable; never expire a
-- claim automatically, since a crashed process may already have paid upstream.
CREATE TABLE reward_gateway_requests (
 api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
 idempotency_key VARCHAR(128) NOT NULL,
 fingerprint TEXT NOT NULL,
 request_id TEXT NOT NULL,
 state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','complete','unavailable')),
 status INTEGER NOT NULL DEFAULT 0,
 content_type TEXT NOT NULL DEFAULT '',
 body BYTEA,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 completed_at TIMESTAMPTZ,
 PRIMARY KEY(api_key_id,idempotency_key)
);
