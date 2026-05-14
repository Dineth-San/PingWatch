# PingWatch 🚀

A full-stack website uptime and performance monitoring application deployed to production on AWS EKS. Monitor your URLs in real-time, detect outages instantly, and receive email alerts when services go down or recover.

## Features

- **User Authentication** – Secure registration and login with JWT tokens (httpOnly cookies, bcrypt hashing)
- **Monitor Management** – Create, edit, and manage URL monitors with configurable ping intervals
- **Automated Pinging** – Background scheduler with 50+ concurrent goroutines, one per monitored URL
- **Outage Detection** – Automatic detection of up↔down state transitions with email alerts via AWS SES
- **Real-Time Dashboard** – Live status indicators, uptime percentages (1d/7d/30d), and response time charts
- **Incident History** – Track outage periods with start/end times and computed duration
- **Performance Tracking** – Store 1000+ ping results with response times, status codes, and error messages

## Tech Stack

### Frontend
- **Next.js 14** (App Router, TypeScript)
- **React** with Server-Side Rendering
- **Recharts** for response time visualization
- **SWR** for client-side data polling

### Backend
- **Go API** (net/http) – User auth, monitor CRUD, read endpoints
- **Go Worker** – Concurrent scheduler with goroutine-based pinging and reconciliation
- **PostgreSQL 16** – Relational database with optimized queries

### Infrastructure
- **Docker** – Multi-stage builds for all 3 services
- **Kubernetes** – EKS cluster with automated deployments
- **AWS Services** – EKS, RDS, ECR, SES, ALB, IAM
- **CI/CD** – GitHub Actions (5-job pipeline: lint → build → deploy)

## Architecture

The project uses a **3-service microarchitecture**, each independently deployable:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster (EKS)                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐  │
│  │  Frontend    │    │  Go API      │    │  Go Worker       │  │
│  │  Next.js     │───▶│  (port 8080) │    │  (no inbound)    │  │
│  │  (port 3000) │    │  ClusterIP   │◀───│  ClusterIP       │  │
│  └──────────────┘    └──────────────┘    └──────────────────┘  │
│         │                                           │             │
│         │ (proxies via                              │             │
│         │  internal                                 │             │
│         │  route handler)                           │             │
│         │                                           │             │
│         └───────────────────────────────────────────┘             │
│                           │                                       │
└───────────────────────────┼───────────────────────────────────────┘
                            │
                     ┌──────▼──────┐
                     │ PostgreSQL  │
                     │  (RDS)      │
                     └─────────────┘
```

### Service Breakdown

1. **Go API** (ClusterIP, 2 replicas)
   - Handles HTTP requests: registration, login, monitor CRUD
   - JWT middleware for authentication
   - CORS handled by Next.js proxy (no public exposure)

2. **Go Worker** (ClusterIP, 1 replica)
   - Runs independently with 50+ concurrent goroutines
   - One goroutine per monitor with time.Ticker
   - Performs HTTP GET requests on schedule
   - Records results in `checks` table
   - Detects state transitions and fires email alerts
   - 60-second reconciliation loop to detect monitor changes

3. **Next.js Frontend** (LoadBalancer, 2 replicas)
   - Server-side rendered React dashboard
   - Catch-all route handler proxies `/api/*` to Go API
   - Real-time polling with SWR (30-second refresh)
   - Live status, uptime bars, response time charts

## Database Schema

### Users
```sql
- id (PK)
- email (UNIQUE)
- password_hash (bcrypt)
- created_at
```

### Monitors
```sql
- id (PK)
- user_id (FK)
- url
- ping_interval (seconds)
- is_active
- created_at, updated_at
```

### Checks
```sql
- id (PK)
- monitor_id (FK)
- status_code
- response_time_ms
- is_up
- error_message
- checked_at
INDEX: (monitor_id, checked_at DESC)
```

### Incidents
```sql
- id (PK)
- monitor_id (FK)
- started_at
- resolved_at
- duration_seconds
```

## Deployment

### Prerequisites
- AWS Account with EKS cluster, RDS PostgreSQL, ECR registries
- kubectl configured for cluster access
- Docker installed locally
- GitHub Actions secrets configured

### Environment Variables
```
DATABASE_URL=postgresql://user:password@rds-endpoint:5432/pingwatch
JWT_SECRET=your-secret-key
API_URL=http://api (internal Kubernetes DNS)
SMTP_HOST=email-smtp.region.amazonaws.com
SMTP_PORT=587
SMTP_USER=ses-smtp-user
SMTP_PASSWORD=ses-password
SENDER_EMAIL=verified-sender@domain.com
```

### Deploy with Kubernetes
```bash
# Apply secrets (create manually in cluster)
kubectl create secret generic pingwatch-secrets \
  --from-literal=database-url=$DATABASE_URL \
  --from-literal=jwt-secret=$JWT_SECRET \
  --from-literal=smtp-password=$SMTP_PASSWORD

# Deploy
kubectl apply -f infra/k8s/
```

### CI/CD Pipeline
GitHub Actions triggered on every push to `main`:
1. **Lint & Test** – go test, npm lint
2. **Build Services** (parallel) – Docker build → ECR push (tagged with commit SHA)
3. **Deploy** – kubectl apply with image tag substitution

## Key Engineering Decisions

### ✅ Worker Isolation
The Worker and API are separate binaries and Kubernetes Deployments. This ensures:
- Pinging activity cannot block API response times
- Each service scales independently
- Clear separation of concerns

### ✅ API Not Publicly Exposed
The Go API uses ClusterIP (no public URL). All browser traffic proxies through Next.js:
- Eliminates CORS configuration
- Reduces attack surface
- Cleaner networking model

### ✅ Runtime Environment Variables
API_URL is read from pod environment at request time, not baked into bundle:
- Avoids build-time hardcoding issues
- Environment-agnostic frontend image

### ✅ Worker Reconciliation Loop
The Worker maintains a map of active monitors and reconciles every 60 seconds:
- Detects new monitors (spawn goroutines)
- Detects deleted/deactivated monitors (cancel goroutines)
- Detects configuration changes (restart goroutines)

### ✅ State Transition Tracking
In-memory sync.Map tracks last known state per monitor:
- Email alerts fire only on up↔down transitions
- Prevents alert spam on every check

### ✅ Optimized Queries
PostgreSQL indices on `(monitor_id, checked_at DESC)`:
- Fast time-range lookups for uptime calculations
- Efficient dashboard queries

## Project Stats

| Metric | Value |
|--------|-------|
| Concurrent Goroutines | 50+ |
| Database Tables | 4 |
| Kubernetes Deployments | 3 |
| API Endpoints | 10+ |
| CI/CD Jobs | 5 |
| Replicas (Total) | 5 |
| Lines of Code (Backend) | ~2000 |
| Lines of Code (Frontend) | ~1500 |

## Development Setup (Local)

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL 16
- Docker

### Running Locally

**API:**
```bash
cd api
go run main.go
```

**Worker:**
```bash
cd worker
go run main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

Visit `http://localhost:3000` to access the dashboard.



**Built as a solo end-to-end project covering backend API development, concurrent systems programming, frontend development, containerisation, cloud infrastructure (AWS), Kubernetes, and CI/CD.**
