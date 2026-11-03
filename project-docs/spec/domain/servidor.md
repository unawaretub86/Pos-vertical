# Domain — Server and LAN access

> **Type:** spec · **Current:** yes · **Rules over:** server bind, access from other devices, serving the embedded frontend
> **Updated:** 2026-10-01 (US-056: `/mesas/*` route, CORS `PATCH`) · Verified against the code (US-054: `cmd/pos/main.go`, `web/static.go`, `internal/netutil/lan.go`, `internal/handler/setup.go`).

## LAN access

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

The UI uses relative `/api` unless `NEXT_PUBLIC_API_URL` is defined (dev on :3000). Dev CORS allows `GET, POST, PUT, PATCH, DELETE, OPTIONS` (`internal/middleware/middleware.go:97`; `PATCH` was added with US-056) and `api.patch` exists in `frontend/lib/api.ts`.
