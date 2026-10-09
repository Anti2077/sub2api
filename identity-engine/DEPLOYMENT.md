# Account model identity

The custom Sub2API image includes this worker and Node 24. Updating the one
Sub2API container starts both services automatically; no separate worker
image, container or environment variables are required. Build from the custom
source or use `ghcr.io/anti2077/sub2api:custom`, not an upstream image.

The worker requires Node 24. Its locked dependencies use pnpm 9.15.9:

```sh
cd identity-engine
corepack prepare pnpm@9.15.9 --activate
pnpm install --frozen-lockfile
pnpm build
pnpm test
pnpm exec vitest run --root vendor/bazaarlink
```

The container supervisor starts the worker, waits for its supported-model
catalog, and then starts Go. Both run as the existing non-root UID 1000.
The worker listens only on `127.0.0.1:8081`; Go defaults to that engine URL
and the callback defaults to `http://127.0.0.1:${SERVER_PORT:-8080}`. Do not
publish 8081. Remove any old `http://model-identity:8081` override when updating.
Normal CLI commands such as `--version`, `--help` and `--setup` bypass the
supervisor. On SIGTERM both services receive a graceful shutdown; after ten
seconds remaining children are killed. If either service unexpectedly exits,
the container exits with a failure so its existing restart policy restarts
both. Tini reaps child processes and forwards container signals.

The old `deploy/docker-compose.model-identity.yml` is now an optional
compatibility overlay with loopback addresses and no additional service.
Use the existing Compose project, file and data paths when upgrading. Existing
image auto-update scripts continue updating the same Sub2API image tag.
Reverse proxies should deny `/internal/model-identity/` from public ingress;
the route additionally requires a random, expiring run capability.

```sh
docker compose --env-file deploy/.env \
  -f deploy/docker-compose.dev.yml build sub2api
```

This command builds the single source image only. Starting or updating the
production deployment is a separate operation. The normal backend startup
applies the database migration, which creates three new tables.
No existing account is recovered or disabled by identity tests.

In Accounts, open Model identity, select an existing test user and a compatible
billing group, and save the configuration. The user's normal permissions,
balance, subscriptions and request limits apply. One permanent dedicated Key
is retained per account, with a descriptive name visible in Usage. Repair
that same Key if deleted/disabled/expired/exhausted/rebound; tests never replace
it. Multiple plans may use distinct request and expected model names. Schedules
start disabled; the default interval is 120 minutes (15–10080 supported).

The Go backend scans schedules every minute and dispatches queued tasks within
two seconds. Database locks and two globally shared slots serialize workers;
each account has at most one active task. Leases expire after 20 minutes.
Missed schedule slots advance to the next original cadence, with at most one
queued catch-up per plan. Manual runs do not shift the schedule. Probes have
a 180-second timeout and four concurrent slots per run. Cancellation invalidates
the capability immediately and interrupts active probes on their next poll.

Reports retain the last 50 terminal runs per plan, including textual probe
responses, requested/upstream models, request IDs, token usage and versions.
Behavioural fingerprints are statistical evidence, not cryptographic proof.
Unsupported, weak, conflicting or stale evidence produces an inconclusive
verdict. The public engine lacks the IKP raw corpus; that layer abstains.
Publish the corresponding source and AGPL license with the deployed service;
see NOTICE.md. Real paid production probes must be verified after deployment.
