# CLAUDE.md — PingWatch

> Full-stack website uptime & performance monitoring app. Users register URLs; the system pings them on a schedule, records results, detects outages, and shows a real-time dashboard.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | Next.js 14 (App Router, TypeScript) |
| Backend API | Go — REST, `net/http` or Chi/Fiber |
| Worker | Go — separate binary, goroutine-per-monitor |
| Database | PostgreSQL |
| Auth | JWT in httpOnly cookies, bcrypt passwords |
| Containers | Docker (multi-stage, one image per service) |
| Orchestration | Kubernetes (AWS EKS) |
| Registry | AWS ECR |
| CI/CD | GitHub Actions |
| Ingress | AWS ALB or NGINX Ingress |

---

## Architecture — Three Services

### 1. Next.js Frontend (port 3000)
- Pages: `/` (landing), `/login`, `/register`, `/dashboard`, `/monitors/new`, `/monitors/[id]`
- No business logic — purely UI + API calls (SWR or React Query for client fetches)

### 2. Go API (port 8080)
- Auth: register, login, logout (JWT issuance)
- Full CRUD for monitors
- Read endpoints for checks, incidents, stats
- Middleware: JWT validation, CORS, rate limiting
- **Does NOT run background jobs**

### 3. Go Worker (separate K8s Deployment)
- On startup: load all active monitors from DB
- Spawn one goroutine per monitor with a `time.Ticker`
- Each tick: HTTP GET → record status code + response time → write to `checks` table
- Detect `up→down` / `down→up` transitions → open/close `incidents` rows
- Send email (SMTP) or webhook alert on state change

```go
func StartScheduler(db *sql.DB) {
    monitors := loadActiveMonitors(db)
    for _, m := range monitors {
        go runMonitor(db, m)
    }
}

func runMonitor(db *sql.DB, m Monitor) {
    ticker := time.NewTicker(time.Duration(m.IntervalSeconds) * time.Second)
    for range ticker.C {
        result := pingURL(m.URL)
        saveCheck(db, m.ID, result)
        evaluateIncident(db, m, result)
    }
}
```

**Worker does NOT talk to the API — it reads/writes PostgreSQL directly.**

---

## Database Schema

```sql
-- users
id            uuid PRIMARY KEY DEFAULT gen_random_uuid()
email         varchar(255) UNIQUE NOT NULL
password_hash varchar(255) NOT NULL
created_at    timestamp NOT NULL DEFAULT now()

-- monitors
id               uuid PRIMARY KEY DEFAULT gen_random_uuid()
user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE
name             varchar(255) NOT NULL
url              varchar(2048) NOT NULL
interval_seconds int NOT NULL DEFAULT 60
is_active        boolean NOT NULL DEFAULT true
created_at       timestamp NOT NULL DEFAULT now()

-- checks  (heaviest table — index on monitor_id + checked_at DESC)
id               uuid PRIMARY KEY DEFAULT gen_random_uuid()
monitor_id       uuid NOT NULL REFERENCES monitors(id) ON DELETE CASCADE
checked_at       timestamp NOT NULL DEFAULT now()
status_code      int
response_time_ms int
is_up            boolean NOT NULL
error_message    varchar(500)

-- incidents
id               uuid PRIMARY KEY DEFAULT gen_random_uuid()
monitor_id       uuid NOT NULL REFERENCES monitors(id) ON DELETE CASCADE
started_at       timestamp NOT NULL
resolved_at      timestamp        -- NULL while ongoing
duration_seconds int
```

> Create index: `CREATE INDEX ON checks (monitor_id, checked_at DESC);`

---

## API Endpoints

```
POST   /api/auth/register
POST   /api/auth/login
POST   /api/auth/logout

GET    /api/monitors
POST   /api/monitors
GET    /api/monitors/:id
PUT    /api/monitors/:id
DELETE /api/monitors/:id

GET    /api/monitors/:id/checks?from=&to=&limit=
GET    /api/monitors/:id/incidents
GET    /api/monitors/:id/stats          # uptime %, avg response time, p95
```

---

## Frontend Pages

| Route | What it shows |
|---|---|
| `/` | Landing/marketing page |
| `/login` | Login form |
| `/register` | Register form |
| `/dashboard` | All monitors — green/red status dot, 30d uptime %, last response time |
| `/monitors/new` | Form: URL, name, interval |
| `/monitors/[id]` | Response time chart (24h), uptime % (1/7/30d), incident list |

---

## CI/CD — GitHub Actions (push to `main`)

1. `lint-and-test` — `go test ./...` + `npm run lint`
2. `build-api` — Docker build → push to ECR (tagged with commit SHA)
3. `build-worker` — Docker build → push to ECR
4. `build-frontend` — Docker build → push to ECR
5. `deploy` — update K8s manifests with new image tags → `kubectl apply`

---

## Kubernetes Resources (per service)

- `Deployment` — N replicas
- `Service` — ClusterIP internal; LoadBalancer/Ingress for frontend + API
- `ConfigMap` — non-secret env vars
- `Secret` — DB credentials, JWT secret, SMTP credentials
- `HorizontalPodAutoscaler` — optional

---

## AWS Infrastructure

| Service | Purpose |
|---|---|
| EKS | Managed Kubernetes (t3.small/medium nodes) |
| RDS | Managed PostgreSQL (db.t3.micro for dev) |
| ECR | Private container registry (one repo per service) |
| ALB | HTTPS termination via K8s Ingress |
| Route53 | DNS (optional, custom domain) |

---

## Build Order

1. **Local dev** — Docker Compose: Go API + worker + Next.js + PostgreSQL
2. **Backend core** — DB migrations, auth endpoints, monitor CRUD, worker scheduler
3. **Frontend core** — Auth pages, dashboard, monitor detail with charts
4. **Alerts** — Email (SMTP) on down/up transitions
5. **Dockerise** — Multi-stage Dockerfiles for all 3 services
6. **CI/CD** — GitHub Actions pipeline + ECR push
7. **Deploy** — K8s manifests, ingress, secrets, RDS connection

---

## Key Decisions & Gotchas

- **Worker isolation**: Worker and API are separate binaries/deployments. This lets you scale them independently and prevents a pinging bottleneck from blocking API responses.
- **checks table growth**: At 10 monitors × 1 min interval = 14,400 rows/day per user. Plan a retention/archival strategy early (e.g. delete checks older than 90 days via a cron).
- **Uptime % calculation**: `COUNT(*) FILTER (WHERE is_up) / COUNT(*)` over a time window from the `checks` table.
- **Incident detection**: Worker keeps an in-memory map of last known state per monitor. On `is_up` flip, insert or update `incidents`.
- **JWT**: Issue on login, validate in API middleware. Store in httpOnly cookie (not localStorage).
- **CORS**: API must allow requests from the Next.js origin.
- **Docker multi-stage**: Use `golang:alpine` builder → `scratch` or `alpine` final image to keep sizes tiny.


# Project Structure

PingWatch/
├── api/          ← Go API service
├── worker/       ← Go worker service
├── frontend/     ← Next.js app
├── infra/
│   ├── k8s/      ← Kubernetes manifests
│   └── docker-compose.yml
└── CLAUDE.md