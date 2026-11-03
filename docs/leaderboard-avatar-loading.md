# Leaderboard images load independently

The leaderboard previously embedded each uploaded avatar's Base64 data URI in
`GET /api/v1/usage/leaderboard`. Its JSON could contain several megabytes of
images, delaying the entire table. The `/auth/me` avatar negotiation never
applied to this endpoint.

## Protocol and caching

- Ranking JSON stays authenticated and `Cache-Control: private, no-store`.
  Inline images are replaced with a small `avatar_version`; external image URLs
  keep the existing `avatar_url` behavior. Internal user IDs remain hidden.
- The repository's batch projection returns metadata only, without transferring
  inline image strings to the application. The version is PostgreSQL `md5(url)`;
  this is a change validator, never a password, signature or access credential.
- The client renders usernames, ranks and metrics immediately, then retrieves
  `GET /api/v1/usage/leaderboard/avatar/:username/:version` through the normal
  authenticated API client as binary blobs. No JWT is placed in an image URL.
- At most three binary requests run concurrently. Desktop and mobile share one
  object URL per image. Switching ranges reuses loaded matching versions and
  aborts obsolete downloads. Removed/replaced images and page teardown revoke
  object URLs. Image errors retain initials without hiding the table.
- Image successes return their image MIME type, `X-Content-Type-Options: nosniff`,
  `ETag: "<version>"`, `Cache-Control: private, no-cache` and `Vary: Authorization`.
  Browsers may retain bytes, but must revalidate before reuse. After current
  authentication and visibility checks, a matching validator returns HTTP 304.
  This permits browser caching, **not shared CDN caching**.
- Image retrieval checks the exact username/version and current privacy state:
  deleted users, unconfirmed usernames, anonymous users, absent images and stale
  versions do not return image bytes or 304. All errors remain `private, no-store`.
  The caller must still satisfy the leaderboard's confirmed-username requirement.
- Images use the panel's global limiter, outside its heavy aggregation limiter.
  Only PNG, JPEG, GIF and WebP data URIs are served. No external URL is fetched by
  the server and no schema/data migration is required.

## Verification

Handler tests cover small ranking responses with a synthetic 800 KiB avatar,
image bytes/MIME, ETag revalidation, invalid images/versions and error caching.
Repository tests check the metadata-only projection and current visibility/version
predicates. Route tests exercise unauthenticated access. Frontend tests hold image
promises pending while verifying that the ranking is already visible; they also
cover concurrency, deduplication, cancellation, changed versions, anonymous rows,
image failures and object URL cleanup.

Deploy frontend and backend together. Older clients still render the ranking but
cannot display inline avatars until their frontend assets refresh. Remote avatar
URLs continue to depend on the remote host's own cache policy. Origin headers
were inspected; CDN provider console rules were not changed.
