# Domain — Server and LAN access

> **Type:** spec · **Current:** yes · **Rules over:** server bind, access from other devices, serving the embedded frontend
> **Updated:** 2026-10-01 (US-056: `/mesas/*` route, CORS `PATCH`) · Verified against the code (US-054: `cmd/pos/main.go`, `web/static.go`, `internal/netutil/lan.go`, `internal/handler/setup.go`).

## LAN access

- The port is `8080` unless `--port N` or the environment variable `POS_PORT` says otherwise (flag first; anything outside 1-65535 is ignored). The bind address, the URL opened in the browser and the `url` of `GET /api/setup/lan-info` (taken from the request `Host`) all follow it (`cmd/pos/main.go`, `resolvePort`; `internal/handler/setup.go`).
- `business_config.lan_access_enabled = 0` (or unreadable config): listens on `127.0.0.1:8080`. `= 1`: `0.0.0.0:8080`. It is read at startup; changing it takes effect when `pos.exe` is restarted.
- The browser always opens at `http://localhost:8080`.
- `GET /api/setup/lan-info` (admin only) returns `{ip, url}`; `ip` is the first private IPv4 (10/8, 172.16/12, 192.168/16); never loopback nor 169.254.x. With no IP: both empty and Settings shows help.
- With the toggle on, Settings warns about changing the initial `admin/admin` password.

## Default password (server-enforced)

- `POST /api/auth/login` with the password `admin` returns `force_password_change: true` and a token carrying the claim `fpc` (`internal/handler/auth.go:52`, `internal/middleware/middleware.go:34`).
- While the claim is set, `Auth` answers `403 {"error":"debe cambiar la contraseña antes de continuar"}` on every protected route except `GET /auth/me` and `PUT /users/me/password` (`allowedWhilePasswordForced`). Calling the API directly with `curl` cannot bypass the change.
- `PUT /users/me/password` returns `{message, token}`; the token is clean (no `fpc`) and the frontend stores it (`frontend/app/change-password/page.tsx`). The old token stays blocked. The new password needs 6+ characters, so `admin` cannot be reused.
- Tests: `TestForcePassword_T1`..`_T4` (`internal/handler/auth_force_password_test.go`).

## Embedded frontend

Resolution order: exact file -> `<path>.html` (no redirect) -> `customers/_.html`, `catalogo/_.html` or `mesas/_.html` for `/customers/*`, `/catalogo/*` and `/mesas/*` (`dynamicRoutes`, `web/static.go:19`; a bare `/mesas` still serves `mesas.html`) -> 404 if it starts with `/_next/` or has an extension -> `index.html`.

The UI uses relative `/api` unless `NEXT_PUBLIC_API_URL` is defined (dev on :3000). CORS only echoes the dev-server origins `http://localhost:3000` and `http://127.0.0.1:3000` (`devOrigins`, `internal/middleware/middleware.go`); any other `Origin` gets no `Access-Control-Allow-Origin`, and the production UI never needs it because it is same-origin. Allowed methods are `GET, POST, PUT, PATCH, DELETE, OPTIONS` (`PATCH` was added with US-056) and `api.patch` exists in `frontend/lib/api.ts`.

## Login attempt limit (R-62)

- `POST /api/auth/login` counts failed logins per client IP (`RemoteAddr`; `X-Forwarded-For` is ignored). The 10th failure inside 10 minutes locks that IP for 5 minutes: `429` with `Retry-After` (seconds), even for the right password (`internal/middleware/ratelimit.go`, `internal/handler/auth.go`).
- A successful login resets the counter. The limit is per IP, not per username, so nobody can lock the admin out by guessing their name. State is in memory; restarting `pos.exe` clears it.
- Tests: `TestLoginLimiter_T1`..`_T6`, `TestCORS_T7`..`_T9` (`internal/middleware/ratelimit_test.go`), `TestLoginRateLimit_T1`..`_T3` (`internal/handler/login_ratelimit_test.go`).
