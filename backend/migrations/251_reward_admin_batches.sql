-- Persist administrative code-generation receipts for uncertain network retries.
CREATE TABLE reward_admin_batches (
 actor_id BIGINT NOT NULL REFERENCES users(id),
 request_key VARCHAR(128) NOT NULL,
 template_id BIGINT NOT NULL,
 count INTEGER NOT NULL,
 days INTEGER NOT NULL,
 codes JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY(actor_id,request_key)
);
