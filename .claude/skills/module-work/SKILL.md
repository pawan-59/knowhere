---
name: module-work
description: >-
  Work on a single Central-Devtron module in isolation — Zoho Desk, Devtron
  releases, Onboarding, Licenses, or Auth. Use this skill whenever the user asks
  to add, change, fix, extend, or debug a specific Central-Devtron module (e.g.
  "add a filter to the license module", "fix the zoho ticket handler", "work on
  onboarding", "the devtron page is broken", "add an endpoint to auth"). It maps
  the exact backend + frontend files for that module, enforces the project's
  authentication and security rules on every change, and runs a build/test
  verification loop (max 5 iterations) before reporting done. Trigger it even
  when the user names a module without saying "module".
---

# Central-Devtron: Module Work

Central-Devtron is a Go backend (`backend/`) + React/Vite frontend (`frontend/`)
that aggregates four operational surfaces behind one authenticated API. This
skill keeps changes **scoped to one module**, **secure by construction**, and
**verified** before you call it done.

## Core rules (do not violate)

1. **Stay in scope.** Touch only the target module's files (see the map below)
   plus, when strictly necessary, the shared wiring points also listed. If a
   change seems to require editing another module, stop and tell the user why
   instead of silently widening the blast radius.
2. **Never weaken auth.** Every data endpoint lives behind authentication. New
   endpoints go on the **protected** mux, never the public one. See
   `references/security-checklist.md` and apply it on every change — this is not
   optional and is the reason this project exists in a locked-down form.
3. **Always verify before finishing.** Run the test loop below (max 5
   iterations). Never report a module as done without a green verification pass.

## Module map

Each module = a backend package (client/store + handler) + one frontend page,
wired through `backend/internal/server/router.go`. Read the existing files
before editing so your changes match the established style.

| Module | Type | Backend files | Frontend | Routes |
|--------|------|---------------|----------|--------|
| **zoho** | Live API (Zoho Desk, India DC) | `backend/internal/zoho/client.go`, `handler.go` | `frontend/src/pages/Zoho.jsx` | `GET /api/zoho/summary`, `GET /api/zoho/tickets` |
| **devtron** | Live API (orchestrator) | `backend/internal/devtron/client.go`, `handler.go` | `frontend/src/pages/Devtron.jsx` | `GET /api/devtron/summary`, `/deployments`, `/version` |
| **onboarding** | SQLite CRUD | `backend/internal/onboarding/onboarding.go`, `handler.go` | `frontend/src/pages/Onboarding.jsx` | `GET/POST /api/onboarding`, `GET/DELETE /api/onboarding/{id}`, `GET /api/onboarding/summary` |
| **license** | SQLite CRUD | `backend/internal/license/license.go`, `handler.go` | `frontend/src/pages/License.jsx` | `GET/POST /api/licenses`, `GET/DELETE /api/licenses/{id}`, `GET /api/licenses/summary` |
| **auth** | Security-critical | `backend/internal/auth/token.go`, `store.go`, `middleware.go`, `handler.go` | `frontend/src/pages/Login.jsx`, `frontend/src/lib/auth.jsx` | `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/me` |

**Shared wiring (edit only when the change needs it):**
- `backend/internal/server/router.go` — register new routes here. Data routes on the `protected` mux; only health/login are public.
- `backend/internal/db/schema.sql` — SQLite schema for `onboarding`/`license`/`auth` (users). Applied idempotently on startup; keep statements `IF NOT EXISTS`. Timestamps are RFC3339 TEXT — use the `db.ParseTime*`/`db.TimeArg`/`db.StrArg` helpers when scanning/binding.
- `backend/internal/config/config.go` + `backend/.env.example` + `config.md` — when a module gains a new setting, add it in all three and document possible values.
- `backend/internal/httpx/httpx.go` — shared JSON/error/CORS/security helpers. Reuse `httpx.JSON` and `httpx.Error`; don't hand-roll responses.
- `frontend/src/lib/api.js` — add the client call here; it already sends the session cookie (`credentials: 'include'`) and handles 401.
- `frontend/src/components/ui.jsx` — shared UI primitives (Card, Stat, Badge, Progress, Spinner, ErrorBox, useAsync). Reuse them.

## Workflow

1. **Identify the module.** Match the user's request to a row above. If it's
   ambiguous (e.g. "releases" → devtron, "tickets" → zoho), state your
   interpretation and proceed. If it genuinely spans two modules, ask.
2. **Read before writing.** Open the module's backend + frontend files first.
   Mirror the existing patterns (error envelope, `useAsync`, `Badge`, etc.).
3. **Make the change**, honoring Core rules 1 & 2.
4. **Run the security checklist** in `references/security-checklist.md` against
   your diff. For `auth` changes, treat every item as blocking.
5. **Verify** with the test loop below.
6. **Report**: what changed, which files, and the final verification result.

## Verification / test loop (max 5 iterations)

Use the bundled harness — it builds and (re)starts the backend against a
throwaway SQLite file (no external services needed) with test credentials, and
runs auth + module endpoint checks:

```bash
bash .claude/skills/module-work/scripts/test-module.sh <module>
# <module> ∈ zoho | devtron | onboarding | license | auth | all
```

The loop protocol:

- Run the harness for the target module.
- **If it exits 0**, verification passed — stop and report success with the summary it printed.
- **If it exits non-zero**, read the failure it reports, fix the root cause in the module, and run again.
- **Repeat at most 5 times.** If it is still failing after the 5th attempt, stop
  and report to the user: the module, what still fails, the harness output, and
  your best hypothesis. Do not loop past 5 — a persistent failure usually means
  a wrong assumption that needs the user, not another blind retry.

What the harness checks (see the script for specifics):

- `go build ./...` and `go vet ./...` — compile + static analysis.
- **Auth gating (always enforced):** the module's endpoints return **401 without
  a session**. This is the security backbone; a data route that answers
  unauthenticated is an automatic failure regardless of anything else.
- **Authenticated behavior:** after logging in, endpoints respond correctly.
  - SQLite modules (`onboarding`, `license`): full CRUD round-trip returns 2xx.
  - Live-API modules (`zoho`, `devtron`) **without credentials**: a clean **503**
    counts as passing (the module is wired correctly, just unconfigured). A
    `500`/crash/hang does **not** pass.
- `auth` module: additionally verifies login rejects bad credentials (401),
  accepts good ones (sets `HttpOnly` cookie), and that logout invalidates access.
- Frontend: `npm run build` succeeds (only run when the frontend page changed).

If you changed only backend or only frontend, you may skip the other half — but
when in doubt, run the full harness.

## Notes

- The harness uses throwaway test admin credentials (`admin@devtron.ai` /
  `testpassword123`) and an ephemeral `AUTH_SECRET`; these are for verification
  only and never touch the user's real `backend/.env`.
- Adding a config value? Update `config.go`, `.env.example`, **and** `config.md`
  together — the docs are part of the contract.
- Keep error messages returned to clients generic for anything auth-related (no
  user enumeration, no internal detail leakage).
