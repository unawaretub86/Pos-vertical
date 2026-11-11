# Domain — Server and LAN access

> **Type:** spec · **Current:** yes · **Rules over:** server bind, access from other devices, serving the embedded frontend
> **Updated:** 2026-11-11 (US-089: "Start with Windows", LAN restart wording) · 2026-11-08 (US-091: route certification, F5 tests, `copy-frontend.go` keeps `.gitkeep`, CI export step) · 2026-11-07 (US-081: concurrent devices, `_txlock=immediate`, busy writes → 503, client retry) · 2026-11-06 (US-086: dynamic pages read their segment from the browser path) · 2026-10-01 (US-056: `/mesas/*` route, CORS `PATCH`) · Verified against the code (US-054: `cmd/pos/main.go`, `web/static.go`, `internal/netutil/lan.go`, `internal/handler/setup.go`).

## LAN access

- The port is `8080` unless `--port N` or the environment variable `POS_PORT` says otherwise (flag first; anything outside 1-65535 is ignored). The bind address, the URL opened in the browser and the `url` of `GET /api/setup/lan-info` (taken from the request `Host`) all follow it (`cmd/pos/main.go`, `resolvePort`; `internal/handler/setup.go`).
- `business_config.lan_access_enabled = 0` (or unreadable config): listens on `127.0.0.1:8080`. `= 1`: `0.0.0.0:8080`. It is read at startup; changing it takes effect when `pos.exe` is restarted (or after the next Windows logon when it starts by itself, § Start with Windows).
- The browser always opens at `http://localhost:8080`.
- `GET /api/setup/lan-info` (admin only) returns `{ip, url}`; `ip` is the first private IPv4 (10/8, 172.16/12, 192.168/16); never loopback nor 169.254.x. With no IP: both empty and Settings shows help.
- With the toggle on, Settings warns about changing the initial `admin/admin` password.

## Start with Windows (US-089)

After a power cut the POS comes back by itself when the shop's Windows user logs in.

- **Install / uninstall** are flags of `pos.exe`, handled before the database is opened, the server started or the browser opened (`cmd/pos/main.go:42-48`, `cmd/pos/autostart.go`). `pos.exe --install-autostart` registers the per-user scheduled task `POS-Vertical` (replacing one of that name, so running it twice leaves one) and prints `Inicio automático instalado`; `--uninstall-autostart` deletes it and prints `Inicio automático quitado`, or `El inicio automático no estaba instalado` (still exit 0) when there was none. A `schtasks.exe` failure or a non-Windows OS exits 1 with a Spanish message that includes the `schtasks` text; the temporary XML file is always removed. If both flags are given, install wins (QA MINOR, roadmap R-171).
- **Mechanism:** Go writes the task XML (UTF-16 LE with BOM, as `schtasks /XML` expects) and calls the built-in `schtasks.exe /Create /TN POS-Vertical /XML <file> /F` (`autostart.go:154-184`); no third-party service manager, no PowerShell. The real Task Scheduler behavior was not run by QA: PO manual check `gaps.md` US-089 M-1.
- **Task definition** (`buildTaskXMLFor`, `autostart.go:87-128`): `LogonTrigger` of the installing user with no delay; runs the absolute path of the running `pos.exe`; **`WorkingDirectory` = the folder of `pos.exe`**; `ExecutionTimeLimit` `PT0S` (the Task Scheduler default of 72 h would kill the POS); `MultipleInstancesPolicy` `IgnoreNew`; starts and keeps running on battery; restarts on failure 3 times, 1 minute apart; `LeastPrivilege` (the POS itself is not elevated). Paths with spaces or `&` are XML-escaped.
- **Why the working directory matters:** the database path is the relative string `data.db` (`cmd/pos/main.go:56` resolves it against the process working directory). A task with no "Start in" would run from `C:\Windows\System32` and create an empty `data.db` there: the onboarding wizard would reappear and the *fiado* ledger would look lost. The process logs the absolute path it opened (`database: <path>`, `main.go:57`).
- **"Under 30 s after a reboot"** is measured from the user's logon: a logon task cannot run before someone signs in, so the shop PC must sign in to Windows by itself (`netplwiz`, documented in `INSTALL.md`). The app answers `GET /api/setup/status` in well under a second; the rest is Windows. The backup goroutine does not delay the listener (`backup.md`).
- **Console window:** `pos.exe` is a console-subsystem executable (no `-H windowsgui`, `Makefile:19`, `:25`), so a black window shows at every logon and **closing it stops the POS**. `INSTALL.md` says to minimize it, not close it. The browser also opens at logon (`cmd/pos/main.go`). A hidden build with a log file and a way to quit and restart is roadmap R-169.
- **One instance:** a second `pos.exe` fails at `ListenAndServe` and exits (`log.Fatalf`); the task's `IgnoreNew` covers the scheduler side.
- **Resetting:** with autostart the POS is normally running, so `reset-data` finds the port busy and refuses; close `pos.exe` first (`reset.md`).
- Tests: `TestAutostart_T15_BuildTaskXML`, `TestAutostart_T16_InstallAndUninstallCalls`, `TestAutostart_T17_Failures`, `TestResolvedDBPath_T18_IsAbsolute` (`cmd/pos/autostart_test.go`, fake `schtasks` runner).

## Default password (server-enforced)

- `POST /api/auth/login` with the password `admin` returns `force_password_change: true` and a token carrying the claim `fpc` (`internal/handler/auth.go:52`, `internal/middleware/middleware.go:34`).
- While the claim is set, `Auth` answers `403 {"error":"debe cambiar la contraseña antes de continuar"}` on every protected route except `GET /auth/me` and `PUT /users/me/password` (`allowedWhilePasswordForced`). Calling the API directly with `curl` cannot bypass the change.
- `PUT /users/me/password` returns `{message, token}`; the token is clean (no `fpc`) and the frontend stores it (`frontend/app/change-password/page.tsx`). The old token stays blocked. The new password needs 6+ characters, so `admin` cannot be reused.
- Tests: `TestForcePassword_T1`..`_T4` (`internal/handler/auth_force_password_test.go`).

## Embedded frontend

Resolution order: exact file -> `<path>.html` (no redirect) -> `customers/_.html`, `catalogo/_.html` or `mesas/_.html` for `/customers/*`, `/catalogo/*` and `/mesas/*` (`dynamicRoutes`, `web/static.go:19`; a bare `/mesas` still serves `mesas.html`) -> 404 if it starts with `/_next/` or has an extension -> `index.html`.

### Route certification (US-091)

- The export has 21 pages (`page.tsx` under `frontend/app`): `/`, `/audit`, `/barra`, `/catalogo/[token]`, `/change-password`, `/customers`, `/customers/[id]`, `/inventory`, `/login`, `/mesas`, `/mesas/[orderId]`, `/onboarding`, `/portfolio`, `/pos`, `/purchases`, `/register`, `/reports`, `/settings`, `/setup`, `/suppliers`, `/users`. A GET for each, with a concrete segment for the dynamic ones (`/customers/42`, `/mesas/7`, `/catalogo/abc123`), with or without a trailing `/` and with a query string, answers 200 `text/html` with that route's own exported file. A reload (F5) therefore never gets a 404.
- An unknown path (`/nope`) and a bare `/catalogo` get `index.html` (the home page) with 200, not a 404. A missing `/_next/...` asset or any path with an extension answers 404.
- Four public pages redirect on load by design, so F5 on them does not always stay: `/login` → `/` when logged in, → `/onboarding` when setup is incomplete (`frontend/app/login/page.tsx:24-30`); `/change-password` → `/` without the force-change flag (`frontend/app/change-password/page.tsx:21-23`); `/onboarding` → `/login` once setup is complete (`frontend/app/onboarding/page.tsx:27-30`); `/setup` → `/login` when logged out (`frontend/app/setup/page.tsx:42-44`).
- `dynamicRoutes` (`web/static.go:19`) is a hand-kept list. `TestFileServer_T18_EveryAppRouteResolves` (`web/static_routes_test.go`) walks `frontend/app` and fails, naming the route, if a dynamic prefix is missing from it or its page does not bake `'_'`. It models each dynamic route as `<prefix>/_.html`, so a nested dynamic route (`app/p/[q]/edit`) would need the test extended (roadmap R-135).
- `TestFileServer_T19_RealExportEveryRoute` and `TestCopyFrontend_T20_KeepsGitkeep` (`web/static_export_test.go`, build tag `export`) run against the real embedded export. Locally: `cd frontend && pnpm run build && cd .. && go run scripts/copy-frontend.go && go test -tags export ./web/`. CI runs them in the `frontend` job right after `pnpm run build`, followed by `git diff --exit-code -- web/out/.gitkeep` (`.github/workflows/test.yml:34-43`).
- `scripts/copy-frontend.go` replaces `web/out` with `frontend/out` and then recreates the tracked, empty `web/out/.gitkeep` (`scripts/copy-frontend.go:29-35`), so the `go:embed` directory exists in a fresh clone and the build never shows the file as deleted.
- F5 in a real browser with an admin session and data is a manual check (`gaps.md`, US-091 M-1).

### Dynamic pages (US-086)

- The export (`output: "export"`, `frontend/next.config.ts:4`) pre-renders each dynamic route only as `_` (`generateStaticParams`, `frontend/app/customers/[id]/page.tsx:3-5`, `frontend/app/mesas/[orderId]/page.tsx:3-5`, `frontend/app/catalogo/[token]/page.tsx:3-5`). The value baked into that page is `_`, so in the binary the route param seen by the page is never the real id.
- `/customers/<id>`, `/mesas/<orderId>` and `/catalogo/<token>` therefore take their segment from `window.location.pathname` through `resolveRouteParam(param, pathname, prefix)` (`frontend/lib/routeParam.ts:17-36`). The route param wins only when it is a real value, i.e. a non-empty string other than `_`. Otherwise the helper returns the URI-decoded first segment after `/<prefix>/`, ignoring a trailing `/`. It returns `null` when the segment is missing, empty, `_` or badly percent-encoded (`%E0`), and never throws.
- The path is read with `useSyncExternalStore` and a `null` server snapshot, so the first client render matches the exported HTML (no hydration error). Until the path is known the page shows its loading state.
- A `null` segment sends no API request. `/customers/<x>` shows `Cliente no encontrado`, `/mesas/<x>` shows the empty order, and `/catalogo/<x>` shows `Catálogo no encontrado o no disponible.`. Customer ids must also be positive integers (`parseCustomerId`, `frontend/lib/routeParam.ts:39-43`), so `/customers/abc` sends no request either. Order ids are not validated: `/mesas/abc` sends `GET /api/orders/abc` → 400 and shows the empty order (R-97).
- Navigating into these pages from inside the app is a full page load in the binary, not a soft navigation (R-98).
- Under `pnpm dev` the three pages answer 500 (`missing param ... in "generateStaticParams()"`, required by `output: "export"`). This was unchanged by US-086 (R-95).
- Tests: `frontend/lib/routeParam.test.ts` T-1..T-8. The Go serving side is unchanged and covered by `TestFileServer_T9_DynamicRoutes` and `TestFileServer_T17_MesasDynamicRoute` (`web/static_test.go`).

The UI uses relative `/api` unless `NEXT_PUBLIC_API_URL` is defined (dev on :3000). CORS only echoes the dev-server origins `http://localhost:3000` and `http://127.0.0.1:3000` (`devOrigins`, `internal/middleware/middleware.go`); any other `Origin` gets no `Access-Control-Allow-Origin`, and the production UI never needs it because it is same-origin. Allowed methods are `GET, POST, PUT, PATCH, DELETE, OPTIONS` (`PATCH` was added with US-056) and `api.patch` exists in `frontend/lib/api.ts`.

## Login attempt limit (R-62)

- `POST /api/auth/login` counts failed logins per client IP (`RemoteAddr`; `X-Forwarded-For` is ignored). The 10th failure inside 10 minutes locks that IP for 5 minutes: `429` with `Retry-After` (seconds), even for the right password (`internal/middleware/ratelimit.go`, `internal/handler/auth.go`).
- A successful login resets the counter. The limit is per IP, not per username, so nobody can lock the admin out by guessing their name. State is in memory; restarting `pos.exe` clears it.
- Tests: `TestLoginLimiter_T1`..`_T6`, `TestCORS_T7`..`_T9` (`internal/middleware/ratelimit_test.go`), `TestLoginRateLimit_T1`..`_T3` (`internal/handler/login_ratelimit_test.go`).

## Session expiry and the live POS cart (R-73)

- The session token lasts 24 hours. When a request returns `401`, the frontend clears the session and goes to `/login` (US-077).
- The open POS cart (lines, customer, global discount) is saved in `localStorage` under `pos.activeCart` on every change and removed when the cart is empty (`frontend/lib/activeCart.ts`, `frontend/app/pos/page.tsx`). After logging in again, `/pos` restores it once the products load, refreshes each line's product data (a product no longer returned is flagged inactive) and shows "Recuperamos la venta que tenías abierta".
- A saved cart older than 12 hours is discarded, because its prices and stock are probably stale. Parked carts (`pos.parkedCarts`) are unchanged.
- The cart belongs to the browser, not to the user: whoever logs in next on the same machine sees it. That is intended for a shared till.
- Tests: `frontend/lib/activeCart.test.ts`.

## Concurrent devices on the LAN (US-081)

Several devices (the PC plus phones or tablets over LAN access) may charge at the same time. They all share the **single open register** of the shop (`RegisterService.Open` refuses a second one, `internal/service/register.go:46-50`). Several independent registers are roadmap `R-18`.

**Database.** WAL, `busy_timeout(5000)`, foreign keys, and `_txlock=immediate` on every pooled connection (`internal/database/database.go:20-23`). Every `BeginTx` is therefore `BEGIN IMMEDIATE`, which takes the write lock at `BEGIN`. A second writer waits up to 5 s for it, instead of failing on a read-to-write upgrade. Before US-081 that upgrade made about half to three quarters of simultaneous credit (*fiado*) sales fail with `database is locked` (measured at the US-081 contract: 5–9 of 20 and 16–26 of 50 succeeded). Test: `TestTxLock_T1_BeginTxTakesWriteLockAtBegin` (`internal/database/txlock_test.go:25`).

**Busy writes → 503.**
- If a write still cannot get the lock (`SQLITE_BUSY`, primary code 5, including 517 `BUSY_SNAPSHOT`, mapped by `mapBusy`, `internal/repository/order.go:52-59`), the server answers **503** `{"error":"El sistema está ocupado, reintenta en un momento"}` (`msgBusy`, `internal/handler/order.go:17`). This covers `POST /sales` and `DELETE /sales/{id}/void` (`internal/handler/sale.go:42-45`, `103-106`), `POST /orders/{id}/checkout` (`internal/handler/order.go:233-236`), every orders endpoint through `respondOrderError` (`order.go:25-26`), and `DELETE /tables/{id}` (`internal/handler/tables.go:91-92`). The driver text never reaches the user.
- **A 503 means nothing was committed.**
  - Inside `saleRepo.Create` / `Void` and the order transactions, `ErrBusy` is produced only before or at `COMMIT` (`internal/repository/sale.go:41-173`, `278-349`). The re-read after the commit (`r.GetByID`) is not mapped, so it never yields 503.
  - A failed statement before `COMMIT` is discarded by the deferred `tx.Rollback()`.
  - A failed `COMMIT` is rolled back by the driver itself: `modernc.org/sqlite` runs `rollback` when `commit` fails while the connection is still inside the transaction (`modernc.org/sqlite@v1.46.1/tx.go:41-50`). The deferred `tx.Rollback()` cannot do it, because `database/sql` marks the transaction done before it calls the driver's `Commit`.
  - `POST /orders` and `DELETE /tables/{id}` map `ErrBusy` only on single autocommit statements (`internal/repository/order.go:120-139`, `internal/repository/dining_table.go:139-146`), and a failed single statement writes nothing.
  - The handlers check `ErrBusy` before any side effect. The e-invoice goroutine runs only on success.
  - `POST /sales/{id}/void-with-credit-note` is deliberately **not** mapped to 503, because by then the credit note has already been issued.
- Other errors keep their previous status (sales: 400 with the message).

**Client retry** (`frontend/lib/api.ts:29-70`).
- On **503**, `request` repeats the same request (method, path, headers, body) up to 2 more times, so 3 attempts in total, waiting 500 ms and then 1000 ms. This applies to any method, because a 503 did not commit. Nothing is shown while retrying.
- After the third 503 it throws the server's message, or `El sistema está ocupado, reintenta en un momento` if the body is not JSON. The caller's existing toast shows it once, and the POS cart is kept.
- Every other status (2xx, 400, 401, 403, 404, 409, 500, 502…) is never retried, and the 401 redirect is unchanged.
- A network error (`fetch` rejects) is **never retried**, for any method, because a sale may have committed before the connection dropped. `networkErrorMessage(method, path)` (`api.ts:41-45`) picks the message:
  - `POST /sales` and `POST /orders/{id}/checkout` (path without `?query`): `No se pudo conectar con el servidor. Revise la red y el historial de ventas antes de cobrar de nuevo.`
  - Anything else: `No se pudo conectar con el servidor. Revise la red e intente de nuevo.`
- Worst case before the error toast: about 16.5 s (3 attempts × up to 5 s of server wait, plus 1.5 s of backoff). The Cobrar button stays disabled meanwhile.

**Guarantees under concurrency.**
- Simultaneous sales on the same product commit whole or not at all.
- Stock never goes negative: the guarded `UPDATE … WHERE stock >= ?` (`internal/repository/sale.go`, `Create`) commits exactly as many sales as there is stock for, and the rest get 400 `stock insuficiente…`.
- Each committed sale has exactly one `sale` register movement, and the register's `cash_sales` and per-method totals equal the sum of the committed payments.
- Two simultaneous voids of the same sale void it once. Two simultaneous checkouts of the same order create one sale.
- No idempotency key exists yet (roadmap `R-120`).

Tests: `TestConcurrency_T2_TwentyCashSales`, `_T3_FiftyCreditSales`, `_T4_MixedPaymentsThirtySales`, `_T5_StockContention`, `_T6_BusyWriteAnswers503`, `_T7_OrderItemBusyAnswers503`, `_T8_DoubleVoidAndDoubleCheckout`, `_T9_DeleteTableBusyAnswers503` (`internal/handler/concurrency_test.go`, file-backed DB, since `:memory:` gives each pooled connection its own DB); F-1..F-5 in `frontend/lib/api.test.ts` (`describe('api 503 retry and network errors (US-081)')`). Real-device checks M-1..M-3 are pending (`gaps.md`).
