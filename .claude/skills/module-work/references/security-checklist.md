# Security Checklist

Apply this to every module change. For the **auth** module, treat every item as
blocking — a mistake here compromises the whole dashboard. The goal isn't
box-ticking; it's preserving the property that **nothing data-bearing is
reachable without a valid session, and nothing leaks that helps an attacker.**

## 1. Authentication gating (the backbone)

- [ ] **New data endpoints are registered on the `protected` mux** in
  `backend/internal/server/router.go`, never on the public `root` mux. Only
  `GET /api/health` and `POST /api/auth/login` are public by design.
- [ ] The endpoint returns **401 without a session** (the harness checks this).
  If a new route answers unauthenticated, that's a security regression.
- [ ] Public endpoints reveal nothing sensitive. `GET /api/health` returns only
  liveness; integration/config status lives behind auth at `GET /api/status`.

## 2. Session & cookie integrity (auth module)

- [ ] Session cookie stays `HttpOnly` + `SameSite=Strict` (set in
  `auth/middleware.go`). `HttpOnly` blocks XSS token theft; `SameSite=Strict`
  blocks CSRF. Don't downgrade either.
- [ ] `Secure` flag is driven by `COOKIE_SECURE` (on in production/HTTPS).
- [ ] Tokens are verified with a **constant-time** comparison (`hmac.Equal`) and
  expiry is checked. Don't introduce string `==` on signatures or tokens.
- [ ] `AUTH_SECRET` is read from config only — never hard-code or log it.

## 3. Credentials & error hygiene

- [ ] Passwords are bcrypt-hashed (`auth/store.go`); never store or log plaintext.
- [ ] Login failures return a **generic** message ("invalid email or password")
  for both wrong password and unknown user — no account enumeration. Keep the
  dummy-hash compare path so timing doesn't leak existence either.
- [ ] Error responses to clients don't leak internal detail (stack traces, DSNs,
  upstream tokens). Use `httpx.Error` with a safe message; keep specifics in
  server logs.
- [ ] Login/sensitive endpoints stay rate-limited (`auth/middleware.go`
  limiter). Don't remove it.

## 4. Input handling

- [ ] Request bodies are parsed with `encoding/json` into typed structs;
  required fields validated before use (see existing handlers).
- [ ] The global 1 MiB body cap (`httpx.LimitBody` in the router) stays in place;
  add tighter per-endpoint limits with `http.MaxBytesReader` if a route warrants.
- [ ] All SQL uses **parameterized queries** (`?` placeholders via database/sql).
  Never build SQL by string concatenation with user input.
- [ ] Path/query params are validated (e.g. `strconv.ParseInt` on `{id}` with a
  400 on failure), as the existing handlers do.

## 5. Secrets & frontend boundary

- [ ] No secret (DB creds, `AUTH_SECRET`, Zoho/Devtron tokens) is exposed to the
  browser. Anything in a `VITE_`-prefixed var is compiled into the public bundle
  — keep integration credentials strictly server-side.
- [ ] Frontend API calls go through `frontend/src/lib/api.js`, which sends the
  cookie via `credentials: 'include'` and bounces to login on 401. Don't add
  `fetch` calls that bypass it or stash tokens in `localStorage`.
- [ ] CORS stays locked to the single configured `ALLOW_ORIGIN` with
  credentials; never emit a wildcard origin.

## 6. Config changes

- [ ] A new setting is added in **all three**: `config.go` (with a safe default),
  `backend/.env.example`, and `config.md` (with description + possible values).
- [ ] Secrets default to empty/unset — never ship a real default secret.
