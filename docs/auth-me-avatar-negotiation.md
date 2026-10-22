# Current-user avatar negotiation

`GET /api/v1/auth/me` continues to return fresh balance, role, status, run mode
and identity summaries on every authenticated request. The frontend still polls
every 60 seconds and refreshes immediately when restoring a session.

## Protocol

- Profile responses include `avatar_version`, an opaque `v1-` SHA-256 digest of
  the user ID, avatar storage source and exact avatar URL/data URI.
- A client without a complete saved avatar/version pair requests `/auth/me`
  normally. It receives the full avatar, preserving older-client behavior.
- A client with that pair may send `?avatar_version=<saved version>`. Only an
  exact match for the authenticated user's current avatar omits `avatar_url`.
- A changed avatar or source returns a different version and the full value.
  A deletion returns a different version with `avatar_url` omitted, matching the
  existing empty-avatar representation. The frontend records this as `null`.
- Omission means reuse **only** when the requested and returned versions and
  user IDs match. Otherwise it means no avatar; it must never keep an old image.
- Profile reads and mutations share the version builder. A profile update
  therefore cannot persist new bytes paired with the previous avatar's version.
- Missing, unknown or malformed versions fall back to a full response. An old
  server without `avatar_version` retains full replacement semantics in the new
  frontend. A missing cached avatar value also forces a full request.

This is application-level negotiation, not a conditional HTTP response.
Successful requests always return HTTP 200 with current user fields, including
when `If-None-Match` is supplied. There is no new avatar endpoint, ETag, HTTP 304,
public resource URL, shared cache entry or credential in an avatar URL. The
digest is a validator, not authorization, and never bypasses JWT validation,
user status/revocation checks, backend-mode guards or rate limits.

## Privacy and session boundaries

Auth routes and authenticated user routes install `Cache-Control: private,
no-store` and append `Vary: Authorization` before authentication and other guards.
Existing `Vary: Origin` is preserved. These rules cover successes and errors,
including 401/403/429/500; personal JSON must not be stored by browsers or CDNs.
The existing per-origin localStorage user record holds the avatar/version pair;
this change does not introduce a separate browser or CDN avatar cache.

The client reuses an image only for the response owner and saved version. It
discards responses from a previous local login session or from a request whose
stored account changed in another tab, and does not let a late 401 clear a
replacement session. A profile update completed during polling takes precedence
over that poll's older snapshot. Logout removes the saved user and pair. Token
rotation for the same account continues to work.

## Scope and evidence

Inline avatars remain supported, including the compressed animated upload
implementation already present in the starting branch. The first fetch, avatar
changes, and older clients still transfer the full image. The improvement avoids
retransmitting it during unchanged read-only polling; it does not remove the
database hydration/hash cost or change upstream caching of remote image URLs.

An in-process handler test uses a synthetic 800 KiB data URI, never production
user data. The full JSON response is about 823.5 kB; the unchanged-avatar response
is about 4.3 kB, a 99.48% reduction, while changed balance and role remain present.
Small byte-count variation comes from serialized timestamps. Historical 809 kB
responses are a motivation, not proof of which field a real user's response held
or a measurement of production latency after this change.

## Verification

Use Go 1.27.2, Node 24 and pnpm 9.15.9. Focused checks from the relevant package
directories:

```sh
# backend
go test -tags=unit ./internal/handler ./internal/server/middleware ./internal/server/routes \
  -run 'TestAuthHandlerGetCurrentUser|TestUserHandlerUpdateProfileReturnsAvatarURL|TestPrivateNoStore|TestAuthAndProfileRoutes' -count=1 -v

# frontend
pnpm exec vitest run src/stores/__tests__/auth.spec.ts src/api/__tests__/auth-avatar-version.spec.ts
pnpm exec vue-tsc --noEmit
```

The tests cover first fetch/startup, unchanged polls, changed bytes/source,
deletion/no avatar, account scoping, late responses after logout/account switch,
profile-update races, old clients/servers, authentication and non-cacheable errors.
No production response, credentials, production services or deployment is needed.
