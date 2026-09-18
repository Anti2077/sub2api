# Sidebar site status

The top-left sidebar displays the current running version for administrators,
including the existing update menu. Regular users see the site's operating age
instead of its software version.

In **Admin → Settings → Site**, configure **Site opening date** (`site_started_on`).
The date is stored in the database as `YYYY-MM-DD`. The counter measures completed
UTC calendar days: the opening day is day 0, and each subsequent UTC midnight adds
one day. An open page refreshes its count within one minute. Clearing the date
hides the user-facing counter. Invalid dates and future dates are rejected by the
settings API.

Migration `243_site_started_on.sql` initializes existing installations from the
earliest account creation date, including deleted accounts. This is an initial
estimate, not a claim about the actual public launch; administrators should
correct it when their launch date differs. A fresh database uses the migration
date. The migration preserves an existing setting, and server/container restarts
do not reset it. Database restores preserve the saved opening date.

Both the public settings API and the initial HTML settings injection include the
date, so the badge does not need an additional request or administrator API access.

## Local verification

Validated with Node 24 / pnpm 9 in Docker and Go 1.27.0:

- 49 frontend tests: role-specific badge, UTC/leap-day boundaries, invalid dates,
  sidebar regressions and public settings store.
- Focused Go settings mutation and public-settings tests passed.
- Production frontend typecheck/build, focused ESLint, 70 contrast checks passed.
- OrbStack project `sub2api-sidebar-qa`, loopback port 18089, independent volumes:
  real admin/user login, settings persistence/clear/validation, HTML injection,
  migration backfill and idempotency, container recreation persistence passed.
- Browser inspection: admin version/menu; user counter in Chinese/English,
  light/dark themes and 390px viewport. Opening date input renders saved value.

The local fixture used an opening date 42 UTC days before the test. This date is
only test data, not a production launch date. A synthetic compliance acknowledgement
was seeded only in the fresh test database to exercise authenticated admin APIs.
