# Routing monitor and daily spending verification

## Features

- `/admin/routing` is an administrator-only page reached directly from the sidebar. The operations overview remains at `/admin/ops`.
- Graph edges use the rendered node coordinates. Shorter columns are vertically centered within the tallest column; the SVG grows to contain every visible node.
- Shared connections aggregate request counts. User/account IDs keep same-name entities distinct. The final selected account is the graph endpoint; ordered attempts remain available per request.
- A route list provides a readable alternative to the graph, including on narrow screens. Nodes/edges support keyboard selection. The view can be paused while the subscription continues.
- Completed/failed events expire after 30 seconds. Periodic snapshots reconcile missing requests without replacing newer in-flight WebSocket updates. Unmount cleans up subscriptions and timers.
- The administrator dashboard has a dedicated daily spending card. `today_actual_cost` is the sum of user charges; `today_account_cost` is upstream cost based on recorded account costs and account multipliers. The two values are not added together. The existing server-day boundaries are retained, independent of the trend date filter.

## Standalone frontend scenarios

```sh
cd frontend
pnpm exec vite --config vite.config.ts --host 127.0.0.1 --port 4177
```

Open `/dev/routing-monitor.html` on this server. The controls generate synthetic routes (12 or 35), empty state, reconnect state, periodic events, and light/dark themes. These fixtures make no production API calls and are excluded from the default production build.

`--config` selects the TypeScript configuration directly, `--host` binds the test server to loopback, and `--port` selects the local port. For a full local frontend/backend check, set `VITE_DEV_PROXY_TARGET` to an isolated local backend; the `/api` proxy supports WebSocket upgrades.

## Regression checks

The root `Makefile` critical frontend suite includes graph/reducer/component regressions and the dashboard spending tests. Coverage includes:

- exact node-edge endpoints and nodes beyond the old fixed-height viewport;
- centered short columns and stable positions;
- shared models, duplicate names, final accounts and failover history;
- terminal events with HTTP reason-phrase statuses;
- expiry, pause/resume, keyboard selection and subscription disposal;
- late HTTP snapshots, continuous live updates and equal-timestamp out-of-order events;
- separate monetary fields, full currency precision, and unavailable versus zero cost.

Backend routing tests cover user IDs and cross-instance Redis events. The cross-instance test waits for **both** service subscriptions before publishing an event.

## Local integration procedure

Restore a database backup only into a separate PostgreSQL instance, with a separate Redis instance. Keep the application and database on an internal Docker network; expose a reverse proxy only on loopback. Use local-only test credentials, acknowledgement fixtures and a mock upstream. Do not commit backup data, credentials, generated JWTs, or private screenshots.

Verify group/account listing through authenticated endpoints. Compare daily spending fields to direct SQL over the same server-day window:

- actual charges: `SUM(actual_cost)`;
- account costs: `SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1))`.

Use a dedicated local group/account/key pointing at the mock upstream. A successful gateway request must return 200 and emit `started` followed by `completed`, with the correct user and account IDs. Confirm the route also appears in the browser through the real WebSocket connection. Existing backup accounts must not be allowed to make outbound production requests during testing.
