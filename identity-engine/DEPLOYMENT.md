# Account model identity

## One image, hosted detection

Use the custom Sub2API image built from this repository, or
`ghcr.io/anti2077/sub2api:custom`. There is one application container and no
extra detection image or public worker port. Node 24 and the legacy AGPL
worker remain bundled for compatibility; the default container detection
mode now calls BazaarLink's hosted Probe API from Go. The frontend never
receives or sends the permanent test Key.

Pass these variables to the application container (putting them in a Compose
`.env` alone does not pass them to the container):

```dotenv
MODEL_IDENTITY_REMOTE_API_URL=https://bazaarlink.ai/api/probe/run
```

The remote API default is injected by the single-image supervisor. Set the public
Base URL in **Model identity → Detection connection settings**. It is stored in the
database, not read from an environment variable, and must be your public HTTPS site origin. A path prefix
is supported when the reverse proxy preserves it. Do not include credentials,
query parameters, or a fragment. BazaarLink rejects private destinations.
For Compose, add this variable to `services.sub2api.environment`, or apply
`deploy/docker-compose.model-identity.yml` as an overlay to the existing
Compose file and project. That overlay does not create a second service.
Keep existing production data paths and image tags when upgrading.

Allow **POST** requests to
`/internal/model-identity/<run-id>/remote/v1/chat/completions` through the public
reverse proxy. Keep all other `/internal/model-identity/` paths private. The
public callback requires the exact dedicated test Key bound to a running,
unexpired detection. It fixes the account, group and request model on the
server; neither headers nor request fields can select another account. It
validates the current binding before ordinary gateway authentication, billing,
limits, model mapping and forwarding. A failed target never switches accounts
or groups. Disabling/deleting the Key or moving the account out of its group
stops further probes. In-flight probes check cancellation every second.

BazaarLink receives the **permanent dedicated test Key** in its HTTPS request
body. This is the chosen integration mode, not a short-lived credential.
Responses redact credential fields and embedded occurrences of that Key before
storage; headers such as Authorization, Cookie and X-API-Key are redacted.
The original upstream credentials stay in Sub2API. Do not log request bodies
sent to the remote API. Tests of real upstreams incur normal gateway charges.

## Administrator workflow

Open **Model identity** in the admin sidebar, at `/admin/model-identity`.
The matrix lists only accounts with detection plans, the latest expected/recognized model and status,
recent five result/time chips, next schedule and dedicated Key binding.
Click a result chip to read its stored report. Expand history to see request
models, expected models, recognized models, full timestamps and errors.
The mobile view uses account cards. Summary counts cover the current page.
API failures show errors and never substitute simulated accounts or results.

Choose **Add plan** in the upper-right corner, select an upstream account, then
search for an existing test
user, choose one compatible account group, and save. The user's existing
permissions, balance, subscriptions and limits apply. No permission grant or
balance top-up happens automatically. The dedicated Key is reused and named
`模型身份测试专用｜分组名称｜账号名称 (#账号ID)`, visible in normal Usage records.
Deleted/disabled/expired/exhausted/rebound Keys must be repaired; no silent
replacement is created.

Add one or more request-model / expected-model plans. Expected models come
from the hosted `/api/probe/baselines` catalog merged with fingerprint-supported
entries in `/api/probe/suggested-models` (V3/V3H), including GPT-6-sol,
GPT-6.1-sol and GPT-6-luna when supported by that current catalog. Schedules start disabled with a default of
120 minutes; 15–10080 minutes is supported. Manual runs do not shift cadence.
Go scans due plans every minute, with database leases/atomic claims for two
global concurrent accounts. Each run permits ten concurrent probe requests,
180 seconds per request and 20 minutes in total. Full probe pools wait up to
30 seconds. Target accounts use the ordinary gateway wait queue without
account failover; user and account concurrency limits still apply. Detection
requests also appear in the normal routing monitor with their test user. Restart skips missed slots
with at most one catch-up per plan. Each plan retains 50 terminal reports.

Only a completed remote `clean_match` with a recognized model equal to the
expected model is an identity match. Family-only, ambiguous, insufficient or
off-baseline results without a specific recognized model stay inconclusive.
Confirmed different models are mismatches. Request failures, configuration
errors, cancellations, timeouts and remote service failures are separate
lifecycle states, not model mismatches. Tests do not automatically disable or
recover upstream accounts.

The hosted API's legacy `identityOnly` flag is ignored by its documented
contract. We request fingerprint analysis and disable the optional context-size
check; the remaining probe phases are managed by BazaarLink. No composite IQ
score is displayed. Stored reports retain the upstream assessment, warnings,
per-question responses, usage and classifier diagnostics. The hosted API does
not promise a fixed engine commit or baseline version; absent versions are
shown as unavailable, not copied from the legacy bundled worker.

BazaarLink currently documents 30 run requests per IP per hour. Plan volume
accordingly; a 429 or remote outage is recorded as a service error rather than
retried indefinitely or interpreted as identity mismatch. Contract reference:
<https://bazaarlink.ai/probe-api-skill.md>.

## Build and verify

```sh
docker compose --env-file deploy/.env -f deploy/docker-compose.dev.yml build sub2api
```

This builds the source image and does not change a running deployment. Startup
applies the existing identity database migration. The supervisor binds the
legacy worker to loopback only; normal CLI `--help`, `--version` and `--setup`
bypass it. Preserve the bundled engine's AGPL license, NOTICE and corresponding
source when distributing the image; see `NOTICE.md`.

For the legacy source build, use Node 24 and pnpm 9.15.9:

```sh
cd identity-engine
pnpm install --frozen-lockfile
pnpm build
pnpm test
pnpm test:engine
```

Mock gateway tests verify target-account billing, no failover, callback
credential/expiry checks, concurrent probe limits, exact-model verdict mapping
and redaction without real paid probes. Verify one real paid detection after
separately deploying/configuring the public callback.
