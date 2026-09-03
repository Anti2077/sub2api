# Public Wiki Foundation Plan

Status: foundation implemented on `Anti2077/wiki-foundation`; broader content remains proposed

The current branch includes the public shell, article and search routes, typed
registry, safe Markdown rendering, responsive navigation, homepage/sidebar
entry points, three Chinese seed articles, and focused tests. Code-block copy
controls and additional verified client guides remain follow-up work.

## Outcome

Add a public, read-only Wiki to the existing Sub2API site at `/wiki`. The first
release should let a new user configure CC Switch and confirm that their key and
endpoint work. The Wiki must ship inside the existing frontend artifact so the
current Docker and embedded-Go deployment paths continue to produce one
deployable application.

## Why this shape

The repository already has Vue 3, Vite, Vue Router, `marked`, DOMPurify, dark
mode, and an embedded frontend build. Reusing those pieces avoids a second
documentation framework, package lockfile, deployment target, theme, and
release pipeline.

The content remains Markdown, so it is easy to review in Git and can be moved to
a dedicated documentation generator later if the Wiki grows beyond the needs
of the application shell.

## Product boundary

### MVP

- Public `/wiki` landing page with task-oriented entry points.
- Public `/wiki/:section/:slug` article routes.
- Responsive article layout with a desktop table of contents and compact mobile
  navigation.
- Typed article registry containing title, summary, section, order, tags, and
  last-verified date.
- Markdown rendering through `marked`, sanitized with DOMPurify.
- Search over article titles, summaries, headings, and tags.
- Copy buttons for code blocks, visible language labels, and horizontal overflow
  for long commands and tables.
- A CC Switch quick-start article and a generic OpenAI-compatible endpoint test.
- Links from the public home navigation and authenticated application sidebar.
- Not-found and unavailable-content states.

### Later

- Additional client guides for Codex, Claude Code, Cherry Studio, and other
  confirmed clients.
- English content after the Chinese structure and terminology stabilize.
- Full-text search index splitting if the content bundle becomes large.
- Version banners for materially different client releases.
- Feedback or issue-report links.

### Not in the foundation

- A browser-based Wiki editor, database-backed content, comments, or accounts.
- Public administrator, deployment, payment-provider, or upstream credential
  documentation.
- Copying real service keys or private production configuration into examples.
- Automatically switching CC Switch providers or editing local user files.
- Maintaining an independent Docusaurus, VitePress, or other documentation
  deployment.

## Architecture

### Routes

| Route | Purpose | Authentication |
| --- | --- | --- |
| `/wiki` | Wiki home, categories, and search | Public |
| `/wiki/:section/:slug` | Canonical article URL | Public |
| `/wiki/search?q=...` | Shareable search results | Public |

The router must explicitly mark these routes with `requiresAuth: false`. If the
application is running in backend-only mode, `/wiki` must be added to the public
allowlist so documentation does not redirect anonymous readers to `/login`.

### Content source

Keep authored pages under:

```text
frontend/src/content/wiki/
  registry.ts
  zh-CN/
    getting-started/
    clients/
    troubleshooting/
```

Use Vite raw Markdown imports and a typed registry. Avoid an ad hoc frontmatter
parser. The registry should be the single place for navigation and search
metadata; each Markdown file should contain only article content.

Suggested metadata shape:

```ts
type WikiArticle = {
  slug: string
  section: 'getting-started' | 'clients' | 'troubleshooting'
  title: string
  summary: string
  order: number
  tags: string[]
  lastVerified: string
  source: string
}
```

`source` is a raw Markdown import. Route slugs must be unique within a section,
and the registry should fail tests when duplicate canonical paths exist.

### UI modules

```text
frontend/src/features/wiki/
  WikiHomeView.vue
  WikiArticleView.vue
  WikiSearchView.vue
  components/
    WikiHeader.vue
    WikiSidebar.vue
    WikiTableOfContents.vue
    WikiSearchInput.vue
    WikiArticleContent.vue
  content.ts
  markdown.ts
```

Keep Wiki-specific rendering and styles inside the feature directory. Extract a
shared safe-Markdown utility only if the Wiki and current legal/announcement
renderers can use one API without changing their behavior.

### Rendering and safety

- Parse GitHub-flavored Markdown with `marked`.
- Sanitize every rendered document with DOMPurify before using `v-html`.
- Generate heading IDs from plain text with deterministic collision suffixes.
- Rewrite internal Wiki links through Vue Router and identify external links.
- Do not allow scripts, inline event handlers, iframes, or arbitrary embedded
  HTML from authored Markdown in the MVP.
- Use text-based copy controls; never place real secrets in rendered examples or
  automated tests.

### Search

Start with a small client-side index generated from the typed registry and
Markdown headings. Normalize case and whitespace and rank exact title matches
before title prefixes, tags, headings, summaries, and body matches. Do not add a
search service until article count or bundle size demonstrates the need.

## Initial CC Switch guide

The first client article should be titled `使用 CC Switch 连接 Sub2API` and cover:

1. Prerequisites: a Sub2API endpoint, API key, supported model name, and a noted
   CC Switch version.
2. Safety first: export or back up existing client configuration before changing
   providers or enabling local routing.
3. Provider creation: map the display name, OpenAI-compatible base URL, API key,
   and model fields using placeholders.
4. Endpoint semantics: explain whether the field expects the origin or a `/v1`
   base path, based on the verified CC Switch version.
5. Switching scope: identify which client configuration CC Switch will update
   and warn that unrelated Codex settings may be replaced in some versions.
6. Verification: issue one minimal request, confirm the HTTP result and model,
   then inspect the Sub2API usage record.
7. Troubleshooting: distinguish `401` authentication, `404` endpoint, model-not-
   found, timeout/network, and overwritten-local-settings cases.
8. Recovery: restore the backup, compare the before/after configuration, and
   retain the newly required routing fields only after review.

UI labels and screenshots must be verified against a current CC Switch release
before publication. The article must not claim that Common Config or provider
switching preserves unrelated Codex settings unless that behavior is tested in
the documented version.

## Delivery phases

### Phase 1: Readable foundation

- Add public routes, feature layout, typed registry, and two seed articles.
- Add home/sidebar navigation entries and localized UI labels.
- Add Markdown sanitization, heading anchors, article navigation, and 404 state.
- Verify responsive light/dark layouts and direct URL refresh through the
  embedded frontend server.

### Phase 2: Findability and authoring quality

- Add search, section filters, code-copy controls, and active table of contents.
- Add registry/content validation tests and an author checklist.
- Add metadata for last verification and version-sensitive notices.

### Phase 3: Broader client coverage

- Publish only guides verified against real client versions.
- Add client-specific troubleshooting and a shared endpoint test article.
- Evaluate English localization and search-index splitting using measured demand.

## Verification gates

Focused tests should cover:

- route access without authentication, including backend-only mode;
- canonical route generation and duplicate-slug rejection;
- Markdown sanitization against script and event-handler payloads;
- deterministic heading IDs and duplicate headings;
- internal/external link handling;
- search ordering and empty/no-result states;
- code copy success/failure behavior;
- missing article and malformed registry entries.

Before merging the feature, run the repository's standard frontend lint,
typecheck, and Vitest suite, then the complete official regression suite required
for custom features. Finish with the normal production build so the Wiki content
is proven to be included in the embedded frontend artifact.

Manual acceptance should cover desktop and mobile widths, light and dark modes,
keyboard navigation, visible focus, direct refresh of nested Wiki URLs, and one
end-to-end CC Switch setup performed with disposable placeholder credentials or
a deliberately scoped test key.

## Decisions needed before Phase 1 content is published

- Public display name and permanent production endpoint format.
- Whether anonymous readers can see the Wiki when backend-only mode is enabled.
  This plan recommends yes.
- Which CC Switch release and operating systems the first guide supports.
- Whether screenshots should use the real site brand or a redacted demo account.
- Where users should report documentation errors.
