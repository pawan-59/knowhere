# Configuration Reference

All configuration for Knowhere is supplied through **environment
variables**. The backend also reads a `backend/.env` file on startup (via
`godotenv`); real OS environment variables take precedence over `.env` entries.

- **Backend** vars are read in `backend/internal/config/config.go`.
- **Frontend** vars are read by Vite at build/dev time (must be prefixed `VITE_`).
- **Database** container vars live in `docker-compose.yml`.

Copy the template to get started:

```bash
cp backend/.env.example backend/.env
```

Legend: **Required** = the app/module will not function without it.
Blank cells under *Default* mean there is no default (the value is empty/unset).

---

## 1. Server

| Variable | Required | Default | Description | Possible values |
|----------|----------|---------|-------------|-----------------|
| `PORT` | No | `8080` | TCP port the backend HTTP server listens on. | Any free port, e.g. `8080`, `3000`, `9090`. |
| `ALLOW_ORIGIN` | No | `http://localhost:5173` | The **single** browser origin allowed to call the API (CORS). Must exactly match the frontend origin, scheme + host + port. Never a wildcard. | A full origin URL, e.g. `http://localhost:5173`, `https://central.devtron.ai`. |
| `CACHE_TTL_SECONDS` | No | `60` | Intended TTL for caching upstream (Zoho/Devtron) responses. **Reserved — currently parsed but not yet wired into the clients.** | Whole number of seconds, e.g. `30`, `60`, `300`. |

---

## 2. Database (embedded SQLite — powers Licenses, Onboarding & Users)

The app uses an embedded, pure-Go SQLite database — no separate database server.
Data lives in a single file; the schema is applied automatically on startup.

| Variable | Required | Default | Description | Possible values |
|----------|----------|---------|-------------|-----------------|
| `DB_PATH` | No | `central.db` | Filesystem path to the SQLite database file. **In a container/pod, point this at a mounted volume so data survives pod rotation.** | Any writable path, e.g. `central.db` (local dev), `/data/central.db` (pod, on a PersistentVolume). |

> **Persistence:** the file (plus its `-wal`/`-shm` companions from WAL mode) must
> live on durable storage. Locally that's your disk; in Kubernetes it's a
> PersistentVolumeClaim mounted at `/data` (see `k8s/pvc.yaml`). Because SQLite
> has a single writer, run **one replica** (the Deployment uses `strategy:
> Recreate` and `ReadWriteOnce`).

---

## 3. Authentication

Controls login, session cookies, and the seeded admin account.

| Variable | Required | Default | Description | Possible values |
|----------|----------|---------|-------------|-----------------|
| `AUTH_SECRET` | **Yes (production)** | *(auto-generated, ephemeral)* | HMAC-SHA256 key that signs session cookies. If unset, a random key is generated at startup — **sessions then break on every restart**, so always set it in production. | A long, random string (≥ 32 chars). Generate with `openssl rand -base64 48`. |
| `AUTH_TOKEN_TTL_SECONDS` | No | `43200` (12 h) | Session lifetime before the user must log in again. | Whole seconds, e.g. `3600` (1 h), `28800` (8 h), `86400` (24 h). |
| `COOKIE_SECURE` | No | `false` | Marks the session cookie `Secure` so browsers only send it over HTTPS. **Set `true` in production.** Keep `false` for local `http://` dev, or the cookie won't be stored. | `true`, `false` (also accepts `1`/`0`). |
| `ADMIN_EMAIL` | Yes* | `admin@devtron.ai` *(in `.env.example`)* | Email of the admin user created/updated on every startup. Without a user, no one can log in. | Any valid email address. |
| `ADMIN_PASSWORD` | Yes* | *(none)* | Password for the seeded admin. Validated to be **at least 8 characters**; startup fails if shorter. | Any string ≥ 8 chars. Use a strong, unique password. |

\* At least one user must exist to log in. `ADMIN_EMAIL` + `ADMIN_PASSWORD`
are the built-in way to create/reset that user. If both are empty, no admin is
seeded and the backend logs a warning that login will fail until a user exists.

> Security notes: passwords are stored bcrypt-hashed; login is rate-limited to
> 5 attempts / 15 min per IP; the session cookie is `HttpOnly` + `SameSite=Strict`.

---

## 4. Zoho Desk (ticket monitoring — India data center)

Create a **Self Client** at <https://api-console.zoho.in> with the
`Desk.tickets.READ` scope to obtain the client credentials and a refresh token.
If any of the four required values are missing, all `/api/zoho/*` endpoints
return `503` and the rest of the dashboard still works.

| Variable | Required | Default | Description | Possible values |
|----------|----------|---------|-------------|-----------------|
| `ZOHO_ACCOUNTS_BASE` | No | `https://accounts.zoho.in` | OAuth token endpoint host for your Zoho data center. | Per DC: `https://accounts.zoho.in` (IN), `https://accounts.zoho.com` (US), `https://accounts.zoho.eu` (EU), `https://accounts.zoho.com.au` (AU), `https://accounts.zoho.jp` (JP). |
| `ZOHO_API_BASE` | No | `https://desk.zoho.in` | Zoho Desk API host for your data center. | `https://desk.zoho.in` (IN), `https://desk.zoho.com` (US), `https://desk.zoho.eu` (EU), `https://desk.zoho.com.au` (AU), `https://desk.zoho.jp` (JP). |
| `ZOHO_CLIENT_ID` | Yes | *(none)* | OAuth client ID from the Zoho API console. | Zoho-issued ID, e.g. `1000.XXXXXXXX...`. |
| `ZOHO_CLIENT_SECRET` | Yes | *(none)* | OAuth client secret. Keep secret; never commit. | Zoho-issued secret string. |
| `ZOHO_REFRESH_TOKEN` | Yes | *(none)* | Long-lived OAuth refresh token; exchanged for short-lived access tokens automatically. | Zoho-issued token, e.g. `1000.XXXX...`. |
| `ZOHO_ORG_ID` | Yes | *(none)* | Zoho Desk organization ID. Found under **Zoho Desk → Setup → Developer Space → API**. | Numeric org ID, e.g. `60012345678`. |

---

## 5. Devtron (release & deployment tracking)

Generate a token at **Devtron → Global Configurations → API Tokens**. If either
value is missing, all `/api/devtron/*` endpoints return `503`.

| Variable | Required | Default | Description | Possible values |
|----------|----------|---------|-------------|-----------------|
| `DEVTRON_BASE_URL` | Yes | *(none)* | Base URL of your Devtron dashboard/orchestrator (no trailing path). | e.g. `https://devtron.yourco.com`. |
| `DEVTRON_API_TOKEN` | Yes | *(none)* | Devtron API token, sent as the `token` request header. Keep secret. | Devtron-issued token string. |

---

## 6. Frontend (Vite)

Frontend variables are read at build/dev time and **must** be prefixed `VITE_`
to be exposed to the browser bundle. Set them in `frontend/.env` or the shell.

| Variable | Required | Default | Description | Possible values |
|----------|----------|---------|-------------|-----------------|
| `VITE_API_BASE` | No | `` (empty) | Base URL prefix for API calls. Leave empty in dev — Vite proxies `/api` to the backend (see `vite.config.js`). Set it only if the API is served from a different origin than the frontend. | Empty string, or a full origin like `https://api.central.devtron.ai`. |

> Do **not** put secrets (Zoho/Devtron/DB credentials, `AUTH_SECRET`) in any
> `VITE_`-prefixed variable — those are compiled into the public browser bundle.

---

## Example: production `.env`

```bash
# backend/.env
PORT=8080
ALLOW_ORIGIN=https://central.devtron.ai

DB_PATH=/data/central.db          # on a mounted PersistentVolume in k8s

AUTH_SECRET=REPLACE_WITH_openssl_rand_base64_48
AUTH_TOKEN_TTL_SECONDS=28800
COOKIE_SECURE=true
ADMIN_EMAIL=ops@devtron.ai
ADMIN_PASSWORD=REPLACE_WITH_STRONG_PASSWORD

ZOHO_ACCOUNTS_BASE=https://accounts.zoho.in
ZOHO_API_BASE=https://desk.zoho.in
ZOHO_CLIENT_ID=1000.xxxxxxxx
ZOHO_CLIENT_SECRET=xxxxxxxx
ZOHO_REFRESH_TOKEN=1000.xxxxxxxx
ZOHO_ORG_ID=60012345678

DEVTRON_BASE_URL=https://devtron.yourco.com
DEVTRON_API_TOKEN=xxxxxxxx
```

## Behavior when unconfigured

| Missing | Effect |
|---------|--------|
| `DB_PATH` not writable | Backend fails fast on startup (the SQLite file must be creatable). |
| Zoho vars | `/api/zoho/*` → `503`; dashboard keeps working. |
| Devtron vars | `/api/devtron/*` → `503`; dashboard keeps working. |
| `AUTH_SECRET` | Ephemeral key generated; sessions drop on restart. |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | No admin seeded; login impossible until a user exists (warning logged). |

Authenticated `GET /api/status` reports which integrations are currently
unconfigured.
