# Knowhere

A unified operations dashboard that brings four Devtron workstreams into one place:

| Module | Source | Status |
|--------|--------|--------|
| **Zoho Desk** | Zoho Desk API (India DC) — ticket monitoring | Live integration |
| **Devtron Releases** | Devtron orchestrator API — app deployments/rollouts + version | Live integration |
| **Onboarding** | Embedded SQLite — customer onboarding status | Local data |
| **Licenses** | Embedded SQLite — Devtron installation license monitoring | Local data |

## Architecture

**Local dev** (two processes, hot reload):
```
frontend/  React + Vite + Tailwind (:5173)  ──proxy /api──►  backend/ (:8080)
backend/   Go API  ──►  Zoho Desk · Devtron orchestrator · SQLite file
```

**Production** (one self-contained container / pod):
```
┌─ pod ─────────────────────────────────────────────┐
│  Go binary  ──serves──►  embedded React frontend    │
│      │        ──►  Zoho Desk · Devtron orchestrator │
│      └──►  SQLite  ──►  /data/central.db  ┐         │
└───────────────────────────────────────────┼─────────┘
                                             ▼
                              PersistentVolume (survives pod rotation)
```

The backend is a single Go service that aggregates all four sources behind one
REST API and, in the container build, also serves the built frontend — so the
whole app is one ~20 MB image. Data lives in an embedded SQLite database (pure
Go, no CGO); in Kubernetes the file sits on a PersistentVolume so it survives
pod rotation. Unconfigured live integrations return `503` so the dashboard still
boots with a subset of modules working.

## Security

The dashboard is **fully authenticated** — no data endpoint responds without a
valid session.

- **Login required for everything.** Only `POST /api/auth/login` and a minimal
  `GET /api/health` are public. Every other `/api/*` route returns `401` without
  a session.
- **Session in an `httpOnly`, `SameSite=Strict` cookie** — unreadable by
  JavaScript (immune to XSS token theft) and not sent cross-site (immune to
  CSRF). Signed with HMAC-SHA256 (`AUTH_SECRET`).
- **Passwords** are bcrypt-hashed; login uses constant-time comparison and a
  generic error, so it never reveals whether an email exists.
- **Brute-force protection**: login is rate-limited to 5 attempts / 15 min per IP.
- **CORS** is locked to the single configured origin with credentials — never a
  wildcard.
- **Hardening**: security headers (`X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy`, `COOP`), 1 MiB request-body cap, and full server timeouts.

> In production: set a strong `AUTH_SECRET` (`openssl rand -base64 48`), set
> `COOKIE_SECURE=true`, and serve over HTTPS.

## Quick start (local dev)

No database server to run — SQLite is embedded (the schema auto-applies on boot,
writing to `./central.db`).

```bash
# 1. Configure the backend
cp backend/.env.example backend/.env
#   → set ADMIN_EMAIL / ADMIN_PASSWORD (your login) and AUTH_SECRET
#   → fill in Zoho + Devtron credentials (see below)

# 2. Run the backend  (terminal 1)
make backend

# 3. Install + run the frontend  (terminal 2)
make install
make frontend
```

Then open http://localhost:5173 and **log in** with the `ADMIN_EMAIL` /
`ADMIN_PASSWORD` you configured. The admin user is created/updated from those
env vars on every startup.

## Deployment (single container / pod)

The whole app builds into one ~20 MB image (Go binary serving the API + embedded
frontend, with SQLite on a mounted volume).

**Docker (locally):**
```bash
make docker-run          # docker compose up --build
open http://localhost:8080
```
Data persists in the `knowhere_data` volume across restarts.

**Kubernetes:**
```bash
# 1. Build & push the image to your registry, update the image ref in
#    k8s/deployment.yaml (knowhere:latest → your-registry/knowhere:tag)
# 2. Put real values in k8s/secret.yaml (or use a sealed-secret / external-secrets)
#    and k8s/configmap.yaml (ALLOW_ORIGIN, ZOHO_ORG_ID, DEVTRON_BASE_URL)
# 3. Apply everything:
kubectl apply -k k8s/
```

The manifests (`k8s/`) provision a **PersistentVolumeClaim** mounted at `/data`,
so the SQLite database — and all your license/onboarding/user data — survives
pod rotation. Because SQLite is single-writer, the Deployment runs **1 replica**
with `strategy: Recreate` on a `ReadWriteOnce` volume. Includes health probes,
a non-root read-only-rootfs security context, and an optional Ingress for HTTPS.

> Scaling note: for multiple replicas / very high write volume, switch the store
> to a networked database (the store interfaces in `internal/license` and
> `internal/onboarding` are the only code that would change).

## Configuration (`backend/.env`)

### Zoho Desk (India data center)
Create a **Self Client** at https://api-console.zoho.in, grant the
`Desk.tickets.READ` scope, and generate a refresh token.

- `ZOHO_CLIENT_ID`, `ZOHO_CLIENT_SECRET`, `ZOHO_REFRESH_TOKEN`
- `ZOHO_ORG_ID` — found under Zoho Desk → Setup → Developer Space → API
- Base URLs default to `accounts.zoho.in` / `desk.zoho.in`.

### Devtron orchestrator
- `DEVTRON_BASE_URL` — e.g. `https://devtron.yourco.com`
- `DEVTRON_API_TOKEN` — Devtron → Global Configurations → API Tokens

> Devtron deployment/version endpoints live in `backend/internal/devtron/client.go`.
> A couple of paths vary by Devtron version — they are centralized there so
> they're easy to adjust to your installation.

### Database
Embedded SQLite — nothing to run. Set `DB_PATH` to control where the file lives
(`central.db` locally, `/data/central.db` on a mounted volume in a pod).

## API surface

All routes except `/api/health` and `/api/auth/login` require a valid session.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/health` | Liveness (public) |
| POST | `/api/auth/login` | Log in, sets session cookie (public) |
| POST | `/api/auth/logout` | Clear session |
| GET | `/api/auth/me` | Current user |
| GET | `/api/status` | Unconfigured integrations (authenticated) |
| GET | `/api/zoho/summary` | Ticket counts by status |
| GET | `/api/zoho/tickets?limit=&status=` | Recent tickets |
| GET | `/api/devtron/summary` | Version + deployment health |
| GET | `/api/devtron/deployments` | Deployment/rollout list |
| GET | `/api/devtron/version` | Devtron server version |
| GET/POST | `/api/licenses` | List / upsert license |
| GET/DELETE | `/api/licenses/{id}` | Get / delete license |
| GET | `/api/licenses/summary` | License health aggregate |
| GET/POST | `/api/onboarding` | List / upsert onboarding |
| GET/DELETE | `/api/onboarding/{id}` | Get / delete onboarding |
| GET | `/api/onboarding/summary` | Onboarding health aggregate |

## Layout

```
backend/
  cmd/server/main.go        entrypoint + graceful shutdown
  internal/config           env-based configuration
  internal/db               SQLite connection + embedded schema.sql + time helpers
  internal/httpx            JSON / CORS / logging helpers
  internal/auth             login, session cookies, RequireAuth middleware
  internal/zoho             Zoho Desk client + handler
  internal/devtron          Devtron orchestrator client + handler
  internal/license          SQLite store + handler
  internal/onboarding       SQLite store + handler
  internal/web              embeds + serves the built frontend (single-container)
  internal/server           router wiring
frontend/
  src/pages                 Overview, Zoho, Devtron, Onboarding, License, Login
  src/components/ui.jsx      shared primitives
  src/lib/api.js             API client
  src/lib/auth.jsx           auth context / session gate
k8s/                        Deployment, Service, PVC, ConfigMap, Secret, Ingress
Dockerfile                  multi-stage single-container build
```
