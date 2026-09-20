# Official main synchronization — 2026-09-20

## Scope

- Target: `Anti2077/custom`, starting at `f867e4f7e48a4a063e168ca80f3649adf77429f4`.
- Official repository: `Wei-Shaw/sub2api`, branch `main`.
- First tested upstream snapshot: `bbdcfbac0a3c0982a33639fba372d55d25724e2f` (32 new commits).
- A pre-publication remote check found two additional commits. Final target: `fbb9006adef852c46f0c7f18b0a8a740722cfac7` (34 upstream commits total).
- Regular merges preserve custom history; both merges resolved automatically without conflicts.

The update includes native Seedance tasks, plugin host services and read-only account metadata, independent OpenAI/TypeSafe moderation profiles, Anthropic tool-schema compatibility, DeepSeek reasoning pass-through, Gemini thinking/SSE compatibility, quota handling, response-model aliases, and OpenAI HTTP/2 keepalive tolerance.

## Regression repairs

Full regression testing reproduced four failures on the original custom baseline:

- Clearing an HTTP proxy username submitted `null`, which the backend treats as no change. Submit an empty string so the username is actually removed; preserve the Hysteria2 branch.
- Registration confirmation tests omitted the existing required username. Supply it and check that it survives both registration paths.

Subsequent full runs also exposed two timing-dependent test fixtures:

- Account selection tests retained mounted components and pending debounce timers. Unmount automatically, control timers, and finish the filter reload before teardown.
- The late-snapshot routing test created its supposedly stale `started` event after the completion event. Assign explicitly ordered timestamps to test the intended scenario deterministically.

## Validation

Frontend (unchanged by the final HTTP/2 increment):

- `pnpm install --frozen-lockfile`.
- `pnpm test:run`: 315 files, 2,321 tests passed; successful exit without unhandled errors.
- `pnpm lint:check`, `pnpm typecheck`.
- `pnpm contrast:check`: 70 checks passed.
- Production frontend build and embedded Go binary built through the repository Dockerfile.

Backend and deployment:

- Full `go test -tags=unit ./...` and `go test -tags=integration ./...` passed for the first merge. Integration used actual OrbStack PostgreSQL/Redis containers with `CI=true` so missing Docker cannot silently pass.
- `golangci-lint run --timeout=30m`: zero issues for the first merge.
- Deployment shell syntax and Apple-container, auto-update, Compose security, gateway environment, runtime resource, Caddy cache, and GitHub-token script tests passed.
- Final HTTP/2 increment: full unit and PostgreSQL/Redis integration suites passed again; golangci-lint reported zero issues. HTTP/2 tests exercise real frame exchange with virtual time, including delayed PING acknowledgements, missing acknowledgements, active response bodies, and the unchanged long-stream policy.

## Local running-service acceptance

Isolated image `sub2api-sync-20260920:local`, application bound to `127.0.0.1:18090`, independent database/Redis and generated test credentials.

The final image (`sha256:e3cc90a4566b9587a0ac38eddd3bbb53d97586450097e751aa9f2db123ed5b70`) was rebuilt after the HTTP/2 increment, recreated successfully, and its running container image ID matched. All seventeen API checks passed again, including health, embedded frontend, administrator login and acknowledgement in the test instance, settings/groups/accounts/plugins/risk-control/prompt-audit reads, custom site-start-date persistence, proxy username clearing, independent moderation profile switching, ordinary-user login, and admin-access denial.

Native browser interaction verified administrator login, the content-audit dialog switching from OpenAI (`3200 ms`) to TypeSafe (`jev-latest`, `4500 ms`), and ordinary-user login displaying `已运营 42 天` without the admin navigation.

An existing local QA database was copied read-only into a separate `upgrade_test` database. The upgraded application retained its two users and `2026-08-07` site date, accepted the existing administrator login, exposed both moderation profiles, and added `content_moderation_logs.engine_meta`. Both fresh and upgraded instances passed restart/persistence checks.

Local evidence is under `/tmp/sub2api-sync-20260920/`; credentials and database contents are excluded from Git. The test scope is local running services, real database/cache integration, and provider protocol fixtures. No paid live provider requests were exercised. Image publication and production deployment are separate from local acceptance and Git push.
