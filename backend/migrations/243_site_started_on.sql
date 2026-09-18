-- Persist a stable opening date: existing sites use their earliest account as
-- an initial estimate; administrators can correct it in site settings.
-- UTC calendar days keep all viewers on the same counter. Never reset on restart.
INSERT INTO settings (key, value, updated_at)
SELECT 'site_started_on',
       TO_CHAR(COALESCE(MIN(created_at), NOW()) AT TIME ZONE 'UTC', 'YYYY-MM-DD'),
       NOW()
FROM users
ON CONFLICT (key) DO NOTHING;
