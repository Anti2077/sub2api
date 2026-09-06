ALTER TABLE users
    ADD COLUMN IF NOT EXISTS username_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS leaderboard_anonymous BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users
SET username_confirmed = TRUE
WHERE BTRIM(username) <> ''
  AND username_confirmed = FALSE;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        WHERE deleted_at IS NULL
          AND BTRIM(username) <> ''
        GROUP BY LOWER(BTRIM(username))
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate active usernames must be resolved before migration 235';
    END IF;
END
$$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active_ci_unique
    ON users (LOWER(BTRIM(username)))
    WHERE deleted_at IS NULL AND BTRIM(username) <> '';
