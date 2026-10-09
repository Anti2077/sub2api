# Account model identity

Build both Sub2API and this worker from the task branch. Never replace the
custom backend with an upstream image: the internal capability route and
strict account selection are required.

The worker requires Node 24. Its locked dependencies use pnpm 9.15.9:

```sh
cd identity-engine
corepack prepare pnpm@9.15.9 --activate
pnpm install --frozen-lockfile
pnpm build
pnpm test
pnpm exec vitest run --root vendor/bazaarlink
```

The source-build Compose overlay is `deploy/docker-compose.model-identity.yml`.
Review it alongside the chosen base Compose file before deployment. The
overlay expects the existing `sub2api-network` and backend port 8080. Set
`MODEL_IDENTITY_ENGINE_URL` and `MODEL_IDENTITY_CALLBACK_URL` on the Go backend;
the worker's `MODEL_IDENTITY_CALLBACK_ORIGIN` must match the callback origin.
No worker host port is published. Keep both services on a private network.
Reverse proxies should deny `/internal/model-identity/` from public ingress;
the route additionally requires a random, expiring run capability.

```sh
docker compose --env-file deploy/.env \
  -f deploy/docker-compose.dev.yml \
  -f deploy/docker-compose.model-identity.yml build sub2api model-identity
```

This command builds source images only. Starting or updating the production
deployment is a separate operation. The migration creates three new tables.
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
