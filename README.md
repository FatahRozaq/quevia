# Quevia — Smart Queue & Appointment Management Platform

> **A multi-tenant B2B SaaS platform that seamlessly unifies online appointment scheduling and on-site queue management into a single, high-performance operational workflow.**

---

## 🌟 Executive Overview

In service-based businesses (clinics, automotive service centers, salons, public service offices), operations are frequently disrupted by the disconnection between **scheduled appointments** and **walk-in customers**:
- Customers face long, unpredictable waiting times on-site.
- Manual scheduling causes catastrophic **double booking**.
- Walk-ins and appointment holders clash without fair, transparent queue prioritization.
- High no-show rates drain business revenue due to a lack of automated omnichannel reminders.

**Quevia** resolves these challenges by integrating appointment booking and on-site queue execution into a unified, concurrency-safe **Modular Monolith**.

---

## 🚀 Key Engineering Highlights

1. **Guaranteed Zero Double-Bookings**: Enforced directly at the kernel database level via PostgreSQL **GiST Exclusion Constraints** (`btree_gist`) over overlapping timestamp ranges.
2. **Deterministic Daily Queue Numbering**: Concurrency-safe counter increments utilizing PostgreSQL transactional row-locking (`ON CONFLICT DO UPDATE RETURNING last_number`).
3. **Multi-Layered Multi-Tenant Isolation**: Application middleware tenant context paired with strict database-level **Row-Level Security (`FORCE ROW LEVEL SECURITY`)**. Cross-tenant data leaks are mathematically impossible.
4. **Sub-Second Real-Time Display Sync**: Real-time queue call events broadcast to public kiosk display screens and customer smartphone trackers in **p95 < 1,000 ms** via WebSocket & Redis Pub/Sub.
5. **Reliable Asynchronous Background Processing**: Transactional Outbox Pattern combined with **River** (PostgreSQL-backed transactional job worker) for resilient email dispatch with exponential backoff retries.
6. **Guest-First Frictionless Experience**: Customers can book, check-in via dynamic QR code, and track queue progress without mandatory account registration.

---

## 🛠️ Technology Stack

| Layer | Technologies |
| --- | --- |
| **Frontend** | [Next.js](https://nextjs.org/) (App Router), [TypeScript](https://www.typescriptlang.org/), [Tailwind CSS](https://tailwindcss.com/), [shadcn/ui](https://ui.shadcn.com/), [TanStack Query](https://tanstack.com/query), [Zod](https://zod.dev/) |
| **Backend API** | [Golang](https://go.dev/) + [Hertz](https://github.com/cloudwego/hertz), [sqlc](https://sqlc.dev/) + [pgx](https://github.com/jackc/pgx), OpenAPI 3.0 / Swagger |
| **Real-Time** | WebSockets (`github.com/coder/websocket`) + [Redis](https://redis.io/) Pub/Sub multiplexing |
| **Database** | [PostgreSQL 16+](https://www.postgresql.org/) (26 tables, `btree_gist`, Row-Level Security) |
| **Background Jobs**| [River](https://riverqueue.com/) (PostgreSQL transactional job engine) + Transactional Outbox Relay |
| **DevOps & Infra** | [Docker](https://www.docker.com/), Docker Compose, [Caddy](https://caddyserver.com/) (Automatic HTTPS), GitHub Actions CI/CD |
| **Observability** | Prometheus metrics, Grafana dashboards, OpenTelemetry distributed tracing |

---

## 📐 System Architecture

```
                    [ Customer Browser / Mobile / Display Kiosk ]
                                         │
                         ┌───────────────┴───────────────┐
                 HTTPS / REST                      WSS / WebSocket
                         │                               │
                         ▼                               ▼
           ┌──────────────────────────────────────────────────────────┐
           │                   Reverse Proxy (TLS)                    │
           └─────────────────────────────┬────────────────────────────┘
                                         │
                         ┌───────────────┴───────────────┐
                         ▼                               ▼
          ┌─────────────────────────────┐  ┌─────────────────────────────┐
          │   Frontend (Next.js App)    │  │    Backend API (Go Hertz)   │
          └─────────────────────────────┘  │   - REST Endpoint Handlers  │
                                           │   - WebSocket Gateway       │
                                           └──────────────┬──────────────┘
                                                          │
                         ┌────────────────────────────────┼────────────────────────────────┐
                         ▼                                ▼                                ▼
          ┌─────────────────────────────┐  ┌─────────────────────────────┐  ┌─────────────────────────────┐
          │    PostgreSQL (Data Inti)   │  │   Redis (Cache & Pub/Sub)   │  │    Worker (Go + River)      │
          │  - 26 Tabel & RLS Multi-    │  │  - Reference Cache          │  │  - Outbox Event Relay       │
          │    Tenant Policies          │  │  - Rate Limiter Token       │  │  - Email Sender (SMTP)      │
          │  - GiST Exclusion Constraint│  │  - Real-Time Pub/Sub        │  │  - Proximity Alert Evaluator│
          │  - River Job Queue Tables   │  │  - Refresh Token Allow-List │  │  - Scheduled Reminders      │
          └─────────────────────────────┘  └─────────────────────────────┘  └─────────────────────────────┘
```

---

## 📁 Repository Structure (Monorepo)

```
quevia/
├── backend/                       # Go Hertz API & River Background Worker
│   ├── cmd/
│   │   ├── api/main.go            # HTTP & WebSocket Gateway Server
│   │   └── worker/main.go         # River Worker & Outbox Relay
│   ├── internal/                  # Domain modules: application/ + service/
│   │   ├── auth/                  # application/ (routing/handlers) + service/
│   │   ├── branch/                # application/ + service/
│   │   ├── service/               # application/ + service/
│   │   ├── staff/                 # application/ + service/
│   │   ├── appointment/           # application/ + service/
│   │   ├── repository/payload/    # DTOs & request validation
│   │   └── platform/              # database (WithTx), cache (redis), security
│   ├── migrations/                # Database Migrations (28 modular files)
│   └── sql/queries/               # sqlc queries per domain
│
├── frontend/                      # Next.js App Router (TypeScript)
│   ├── app/                       # Route groups: (public), (customer), (dashboard), (operator), display
│   ├── components/                # Reusable UI & queue display components
│   └── hooks/                     # Custom hooks (useQueueSocket, etc.)
│
├── deploy/                        # Production deployment (Docker, Caddy/Nginx)
├── docs/                          # Complete Specifications & Engineering Reports
│   ├── prd.md                     # Product Requirements Document (PRD v1.2)
│   ├── architecture.md            # System Architecture & API Contract (v1.1)
│   ├── erd.md                     # Database Design & ERD (v1.1)
│   ├── implementation-plan.md     # 8-Week Engineering Implementation Plan
│   └── reports/                   # Audit & Validation Reports (Phases 1 - 5)
│
├── docker-compose.yml             # Local development services (Postgres, Redis, Mailpit)
├── Makefile                       # Developer shortcuts (make dev, make test, etc.)
├── IMPLEMENTATION_PLAN.md         # Symlink to implementation plan
└── README.md                      # Project documentation
```

---

## 📚 Official Documentation Index

The platform has been audited, refined, and validated to an **Engineering-Ready Quality Score of 97/100**:

- 📄 [**Product Requirements Document (PRD v1.2)**](docs/prd.md) — Comprehensive user stories (US-01 to US-10), business rules (BR-01 to BR-21), and quantitative NFRs (WCAG 2.2 AA, 99.5% SLO).
- 🏛️ [**System Architecture & API Contract (v1.1)**](docs/architecture.md) — Runtime flows, WebSocket event catalog, and REST endpoint specifications.
- 🗄️ [**Database Design & ERD (v1.1)**](docs/erd.md) — 26 tables, DDL constraints, indexing strategy, and PostgreSQL Row-Level Security policies.
- 🗺️ [**Engineering Implementation Plan (v1.0)**](IMPLEMENTATION_PLAN.md) — 8-week delivery timeline (4 sprints), Gantt charts, critical path, and WBS task checklist.
- 📑 **Audit & Verification Reports**:
  - [PRD Assessment Report](docs/01-prd-assessment-report.md)
  - [Requirement Audit Report](docs/02-requirement-audit-report.md)
  - [Product Improvement Proposal](docs/03-product-improvement-proposal.md)
  - [PRD Change Log](docs/04-prd-change-log.md)
  - [Engineering Validation Report](docs/05-engineering-validation-report.md)

---

## ⚡ Quickstart (Local Development)

### 1. Prerequisites
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [Go 1.22+](https://golang.org/dl/)
- [Node.js 20+](https://nodejs.org/) & pnpm / npm

### 2. Start Core Infrastructure Services
Clone the repository and start the development backing services:

```bash
# Clone the repository
git clone https://github.com/FatahRozaq/quevia.git
cd quevia

# Start PostgreSQL, Redis, and Mailpit
docker compose up -d
```

Service endpoints will be available at:
- **PostgreSQL**: `localhost:5432` (User: `postgres`, Password: `postgres_password`, DB: `quevia_dev`)
- **Redis**: `localhost:6379`
- **Mailpit Web UI (Mock SMTP)**: [http://localhost:8025](http://localhost:8025)

---

## ⚖️ License
This project is licensed under the [MIT License](LICENSE).
