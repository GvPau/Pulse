# Pulse Roadmap

Status: build the backend as a real product first, client/admin/landing later.
Single-process Go service + PostgreSQL. Keep architecture ready to scale
horizontally (one Docker pod now; external queue once warranted).

## Where we are (baseline, September 2026)

- **Auth**: email/password + JWT, user-scoped resources.
- **Monitors CRUD** -> scheduler via event callbacks (`add` / `update` / `remove`).
- **Scheduler**: min-heap priority queue (wake-on-next, one-shot timer),
  N=3 workers sharing a buffered `jobs` channel.
- **Option B (hot path without DB)**: scheduler reschedules in memory using the
  interval stored in the queue entry; a batched flusher persists `next_run`
  once per second (deduped map, final flush with `context.WithoutCancel`).
- **Worker fallback**: on `monitor.ErrNotFound` the worker emits a `remove`
  event so ghost monitors disappear from the queue.
- **Checker**: HTTP method + expected status + timeout; incidents open at the
  failure threshold and resolve on recovery.
- **Graceful shutdown** wired (scheduler waits for in-flight work, closes jobs).
- **Pricing doc** (`docs/PRICING.md`) is direction, NOT final. The minimum
  interval validation (reject < 60s) is deferred until pricing is decided.

## Next steps

### Phase 1 — API foundations ✅ (06 Sep 2026)
- **Error contract**: `{"error":{"code","message","details"}}`, codes `invalid_request`, `unauthorized`, `not_found`, `conflict`, `internal_error`. Validations → 400 with field-level `details`.
- **Pagination**: `?page=&limit=` (default 1/20, max 100), envelope `{data, pagination:{page,limit,total,has_more}}`.
- **Filters/sorts (monitors)**: `?q=` (ILIKE), `?active=true|false`, `?sort=name|created_at|interval_seconds&order=asc|desc`. Whitelist enforced.
- **`healthz`/`readyz`**: liveness (always 200) + readiness (pool.Ping with 2s timeout, 503 on failure).
- **Contract-first doc** (`docs/API.md`): *pendiente* — se añadirá como cierre de fase.
- **Incidents/checks**: paginación y filtros pendientes para cuando se retomen esos endpoints.

### Phase 2 — Notifications
- `notification_channels` (webhook first, email later via a provider such as Resend).
- `monitor_channels` (many-to-many: monitor -> channels to notify).
- `internal/notifier` with a `Dispatcher` interface (webhook + email impls).
- Hooks in the incident lifecycle: opened -> DOWN alert, resolved -> RECOVERED.
- Delivery log and retries.

### Phase 3 — Availability metrics
- Uptime %, average response time over time windows, computed from `monitor_checks`.
- Endpoints for dashboards and status pages.

### Phase 4 — Check types + retention
- SSL/TLS certificate expiry, keyword match, latency threshold.
- Retention policy for `monitor_checks` (age-based delete or daily aggregation).

### Phase 5 — Auth completeness
- Google SSO.
- Email verification.
- User profile (change password, delete account).

### Phase 6 — Public + admin surfaces
- Public status pages (read-only uptime for a monitor/user).
- Admin endpoints (users, stats) for the admin client.

### Phase 7 — Hardening & scale
- Tests.
- Observability.
- Docker single-pod deployment.
- Later: Redis / external queue for horizontal scaling, only when the channel-based
  queue becomes the bottleneck.

## Frontends (later, separate repos)
- Client app (dashboard, monitor management, incidents, notifications).
- Admin app.
- Landing page.