# API Monitor

> Simple API and endpoint monitoring for developers.

API Monitor is a lightweight SaaS that allows developers and small teams to monitor their APIs and web services without having to configure complex observability infrastructure.

The goal is simple:

**Add an endpoint → monitor it automatically → know when it breaks.**

---

## 🎯 Product Vision

Developers should be able to monitor the health of their APIs in a few minutes.

The product should answer questions like:

* Is my API currently working?
* Is an endpoint returning the expected status code?
* Is it getting slower?
* When did it fail?
* How long was it down?
* Did it recover?
* Can I receive an alert when something breaks?
* Can I periodically test a complete API flow?

The product should prioritize **simplicity over complexity**.

It is not intended to replace advanced observability platforms such as Grafana, Prometheus, Datadog or New Relic.

Instead:

> **Simple monitoring for developers who just want to know when their API breaks.**

---

# 🧩 Core Features

## 1. Monitor URLs

Users can create monitors for their APIs and websites.

Example:

```text
Name:
Production API

URL:
https://api.example.com/health

Method:
GET

Interval:
5 minutes

Timeout:
10 seconds

Expected status:
200
```

The system periodically executes the request and stores the result.

Example:

```text
10:00 → 200 → 124ms → ✓
10:05 → 200 → 137ms → ✓
10:10 → 200 → 119ms → ✓
10:15 → 500 → 231ms → ✗
10:20 → 500 → 215ms → ✗
10:25 → 200 → 141ms → ✓
```

---

# 2. Dashboard

The main dashboard should provide an immediate overview of all monitors.

Example:

```text
┌──────────────────────────────────────────────┐
│ My Monitors                         + Add     │
├──────────────────────────────────────────────┤
│                                              │
│ 🟢 Production API                 99.98%      │
│    /health                         142ms       │
│                                              │
│ 🟢 Users API                      100.00%      │
│    GET /users                      187ms       │
│                                              │
│ 🔴 Orders API                      97.20%      │
│    POST /orders                    842ms       │
│                                              │
└──────────────────────────────────────────────┘
```

The dashboard should show:

* Current status
* Uptime
* Response time
* Last check
* Number of incidents
* Current incidents

---

# 3. Monitor Details

Each monitor should have a dedicated page.

Example:

```text
Production API

Status
🟢 Operational

Uptime
99.98%

Average response time
143ms

Last 24 hours
█████████████████████████████████

Recent checks

✓ 11:45   142ms
✓ 11:40   138ms
✓ 11:35   151ms
✗ 11:30   500
✓ 11:25   139ms
```

Users should be able to inspect historical checks.

---

# 4. Incidents

A single failed request should not necessarily create an incident.

The system should detect consecutive failures.

Example:

```text
Check #1 → 500
Check #2 → 500
Check #3 → 500
```

After a configurable number of failures:

```text
🔴 INCIDENT STARTED
```

When the endpoint starts working again:

```text
🟢 INCIDENT RESOLVED
```

The incident should contain:

* Start time
* End time
* Duration
* Number of failed checks
* Error/status code
* Monitor affected

Example:

```text
Production API

Incident
────────────────────────

Started:
26 Aug 2026 10:15

Resolved:
26 Aug 2026 10:32

Duration:
17 minutes

Failed checks:
4

Error:
HTTP 500
```

---

# 5. Email Notifications

Users can configure email notifications.

## Incident notification

```text
🔴 Production API is down

GET https://api.example.com/health

Expected:
200

Received:
500

Failed checks:
3 consecutive failures

Started:
10:15
```

## Recovery notification

```text
🟢 Production API has recovered

Downtime:
17 minutes

Failed checks:
4
```

---

# 6. Weekly Reports

The system can send a weekly summary.

Example:

```text
Your weekly monitoring report

────────────────────────────

Overall uptime
99.97%

Checks performed
20,431

Incidents
2

Total downtime
12m 32s

Average latency
143ms

────────────────────────────

Production API
99.99%

Users API
100.00%

Orders API
99.91%
```

This provides value even when there are no incidents.

---

# 7. Webhooks

Users can configure a webhook URL.

When an incident occurs, API Monitor sends:

```http
POST https://api.example.com/monitoring-webhook
```

Example payload:

```json
{
  "event": "monitor.failed",
  "monitor": {
    "id": "mon_123",
    "name": "Production API"
  },
  "status_code": 500,
  "timestamp": "2026-08-26T10:15:00Z"
}
```

Supported events could include:

```text
monitor.failed
monitor.recovered
incident.started
incident.resolved
```

This allows customers to integrate API Monitor with their own systems.

---

# 8. API Monitoring

The first version will support simple HTTP monitoring:

```text
GET
POST
PUT
PATCH
DELETE
```

Users should be able to configure:

* HTTP method
* URL
* Headers
* Request body
* Expected status code
* Timeout
* Check interval

Example:

```text
POST /api/users

Headers:
Authorization: Bearer xxx
Content-Type: application/json

Body:

{
  "name": "monitor-test"
}

Expected status:
201
```

---

# 9. API Assertions

Eventually, monitors should be able to verify more than the HTTP status code.

For example:

```text
HTTP status == 200

Response time < 500ms

JSON.status == "ok"

JSON.users.length > 0
```

This transforms the product from simple uptime monitoring into **API health monitoring**.

---

# 10. CRUD / Flow Monitoring

A future feature will allow users to define complete API flows.

Example:

```text
Create user
    ↓
GET user
    ↓
Update user
    ↓
DELETE user
```

The system can execute the complete flow periodically against a staging/test environment.

Example:

```text
POST /users
      ↓
201 ✓

GET /users/{id}
      ↓
200 ✓

PUT /users/{id}
      ↓
200 ✓

DELETE /users/{id}
      ↓
204 ✓
```

This allows developers to detect broken integrations rather than simply checking whether a server responds.

---

# 💰 Pricing

The initial goal is to make the product cheap enough that developers can try it without thinking too much about the cost.

### Free

```text
5 monitors
15 minute interval
7 days history
Email alerts
```

### Basic — €5/month

```text
20 monitors
5 minute interval
30 days history
Incident alerts
Recovery alerts
Weekly reports
Webhooks
API monitoring
```

### Pro — €10/month

Potential future plan:

```text
100 monitors
1 minute interval
90 days history
Advanced assertions
API flows
Multiple environments
Priority support
```

Pricing is experimental and should be validated with real users.

---

# 🏗️ Technical Architecture

The project will initially be a monorepo.

```text
                    ┌──────────────┐
                    │   React UI   │
                    └──────┬───────┘
                           │
                          HTTP
                           │
                           ▼
                    ┌──────────────┐
                    │    Go API    │
                    │              │
                    │ Auth         │
                    │ Monitors     │
                    │ Incidents    │
                    │ Dashboard    │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ PostgreSQL   │
                    └──────────────┘


                    ┌──────────────┐
                    │ Go Worker    │
                    │              │
                    │ Scheduler    │
                    │ HTTP checks  │
                    │ Incidents    │
                    │ Notifications│
                    └──────┬───────┘
                           │
                           ▼
                       Internet
```

Initially, we should avoid unnecessary infrastructure.

No Kubernetes.

No Kafka.

No microservice architecture.

No complicated distributed system.

Start simple and scale only when there is a reason.

---

# 🦫 Why Go?

Go is the main reason for building this project.

The project should be used to learn Go through a real application rather than tutorials.

Areas we want to learn:

### Go fundamentals

* Structs
* Interfaces
* Error handling
* Packages
* Modules
* Generics where appropriate
* Testing

### HTTP

* `net/http`
* HTTP clients
* HTTP servers
* Request handling
* Timeouts
* Context

### Concurrency

* Goroutines
* Channels
* Worker pools
* Synchronization
* Race conditions
* Cancellation

### Backend

* PostgreSQL
* SQL
* Transactions
* Connection pools
* Migrations

### Distributed systems

Eventually:

* Job queues
* Retry strategies
* Exponential backoff
* Idempotency
* Scheduling
* Multiple workers

### Observability

Eventually:

* Structured logging
* Metrics
* OpenTelemetry
* Prometheus
* Tracing

---

# 🗺️ Development Roadmap

## Phase 1 — Project foundation

Goal: Have a working Go backend.

* [ ] Create Go project
* [ ] Configure modules
* [ ] Create HTTP server
* [ ] Health endpoint
* [ ] PostgreSQL
* [ ] Database migrations
* [ ] Basic project structure
* [ ] Docker Compose
* [ ] Basic tests

---

## Phase 2 — Authentication

Goal: Users can create accounts.

* [ ] User model
* [ ] Registration
* [ ] Login
* [ ] Password hashing
* [ ] Authentication middleware
* [ ] Sessions/JWT
* [ ] User isolation

---

## Phase 3 — Monitors

Goal: Users can create and manage monitors.

* [ ] Create monitor
* [ ] List monitors
* [ ] Get monitor
* [ ] Update monitor
* [ ] Delete monitor
* [ ] Enable/disable monitor
* [ ] Configure interval
* [ ] Configure timeout
* [ ] Configure expected status

---

## Phase 4 — Monitoring engine

Goal: Automatically execute checks.

* [ ] Scheduler
* [ ] HTTP checker
* [ ] Store check results
* [ ] Measure latency
* [ ] Handle timeouts
* [ ] Handle connection errors
* [ ] Worker pool
* [ ] Concurrent checks
* [ ] Context cancellation

At the end of this phase:

```text
User creates monitor
        ↓
Scheduler detects it
        ↓
Worker executes HTTP request
        ↓
Result stored in PostgreSQL
        ↓
Dashboard displays result
```

---

## Phase 5 — Incidents

Goal: Detect actual outages.

* [ ] Consecutive failure detection
* [ ] Incident creation
* [ ] Incident resolution
* [ ] Downtime calculation
* [ ] Incident history
* [ ] Recovery detection

---

## Phase 6 — Notifications

Goal: Notify users when something happens.

* [ ] Email provider integration
* [ ] Incident email
* [ ] Recovery email
* [ ] Notification preferences
* [ ] Weekly reports
* [ ] Webhook notifications

---

## Phase 7 — Advanced API Monitoring

Goal: Monitor APIs rather than only URLs.

* [ ] GET
* [ ] POST
* [ ] PUT
* [ ] PATCH
* [ ] DELETE
* [ ] Custom headers
* [ ] Request body
* [ ] Expected status
* [ ] Response time assertions
* [ ] JSON assertions

---

## Phase 8 — API Flows

Goal: Test complete API workflows.

* [ ] Multiple requests per monitor
* [ ] Variables
* [ ] Request dependencies
* [ ] Extract values from responses
* [ ] CRUD flows
* [ ] Test environments

Example:

```text
POST /users
      ↓
extract user.id
      ↓
GET /users/{{user.id}}
      ↓
PUT /users/{{user.id}}
      ↓
DELETE /users/{{user.id}}
```

---

## Phase 9 — Billing

Goal: Turn the project into a real SaaS.

* [ ] Stripe integration
* [ ] Free plan
* [ ] €5 Basic plan
* [ ] Usage limits
* [ ] Subscription management
* [ ] Billing portal
* [ ] Payment webhooks

---

## Phase 10 — Production

Goal: Run the actual service reliably.

* [ ] Production deployment
* [ ] HTTPS
* [ ] Domain
* [ ] Database backups
* [ ] Logging
* [ ] Metrics
* [ ] Monitoring the monitoring service
* [ ] Error tracking
* [ ] Rate limiting
* [ ] Security review

---

# 🧠 Product Principles

### 1. Simple

The user should be able to create their first monitor in less than a minute.

### 2. Cheap

The initial paid plan should be cheap enough to remove purchase friction.

### 3. Developer-focused

The product is primarily for developers and small technical teams.

### 4. No unnecessary complexity

We should not build enterprise functionality before having customers.

### 5. Build for real users

Features should be driven by feedback rather than assumptions.

---

# 🎯 MVP Definition

The first version is considered complete when a user can:

```text
1. Create an account
        ↓
2. Add an endpoint
        ↓
3. Select an interval
        ↓
4. API Monitor checks it automatically
        ↓
5. Results are stored
        ↓
6. Dashboard shows the status
        ↓
7. User receives an email if it goes down
        ↓
8. User receives an email when it recovers
```

Everything else comes afterwards.

---

# 🚫 Things We Will NOT Build Initially

To avoid overengineering:

* Kubernetes
* Microservices
* Kafka
* Multiple regions
* Complex RBAC
* Mobile app
* Native desktop app
* AI features
* Advanced analytics
* Enterprise SSO
* Custom dashboards
* Hundreds of configuration options

The first goal is simply:

> **Can we reliably detect when an API is broken and tell the developer?**

If yes, we have the foundation of the product.

---

# 📈 Long-Term Vision

If the product gets real users, it could evolve from:

```text
URL monitoring
```

into:

```text
API monitoring
        ↓
API health checks
        ↓
API assertions
        ↓
API workflows
        ↓
Synthetic monitoring
        ↓
Developer observability platform
```

The long-term objective is not to compete with large observability platforms.

It is to become the tool that a developer can configure in **5 minutes** and forget about until something breaks.

---

## First Milestone

The first technical milestone is intentionally tiny:

```text
Go server
    ↓
PostgreSQL
    ↓
Create monitor
    ↓
Background scheduler
    ↓
HTTP request every N minutes
    ↓
Store result
    ↓
GET /monitors/:id/checks
```

Once this works reliably, we build the UI.
