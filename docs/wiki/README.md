# Sub2API Wiki

This directory is the source of truth for the planned public user Wiki.
The Wiki is intended to help a new user go from receiving an API key to making
their first successful request without exposing deployment secrets or requiring
administrator access.

The implementation plan lives in [PLAN.md](./PLAN.md). The initial content map
is tracked in [CONTENT_MAP.md](./CONTENT_MAP.md).

## Audience

- Users who have received a Sub2API API key but have not configured a client.
- Users moving between supported clients and needing exact field mappings.
- Support staff who need one canonical link instead of repeating setup steps.

Administrator deployment and upstream account management remain in the existing
repository documentation. They are not part of the public Wiki unless a later
scope decision explicitly includes them.

## Editorial rules

1. Use screenshots only as supporting evidence. Every procedure must remain
   understandable as text because client interfaces change.
2. Use placeholders such as `https://your-sub2api.example/v1` and
   `sk-your-key` in committed examples. Never commit a real host, API key,
   account identifier, or private routing configuration.
3. Explain what each field changes, not only what value to paste.
4. Separate verified behavior from version-sensitive notes. Record the client
   version and verification date for screenshots and UI-specific instructions.
5. Include a success check and common failure modes in every configuration
   guide.
6. Warn before operations that can overwrite an existing client configuration.
   Backup and diff instructions must precede CC Switch routing changes.
7. Keep product claims factual. Do not promise model availability, pricing,
   uptime, or compatibility that the live service does not expose.

## Definition of a publishable article

A Wiki article is publishable when it has:

- a clear audience and expected outcome;
- prerequisites and supported client versions;
- numbered steps with copy-safe placeholders;
- a verification request that does not consume avoidable quota;
- troubleshooting for authentication, endpoint, model, and network failures;
- a last-verified date and evidence source;
- desktop and mobile layout review if the article contains screenshots or
  tables.
