# SPRINT-18 — Stabilization and Production Blockers

**Goal:** Eliminate the 4 critical blockers (security, data consistency, static serving and session) that prevent delivering the binary to a real customer. This sprint introduces no new business features; it lays a solid foundation so the system does not crash or compromise information on first use.

**Start date:** (to be defined)

**References:** `project-docs/roadmap.md`, `project-docs/gaps.md`

**Incoming technical debt:**
- R-55: The binary does not serve the embedded frontend: blank page on `:8080`. -> **Closed in US-075**.
- R-57: `GET /setup` exposes `owner_email_app_password` to any authenticated user. -> **Closed in US-076**.
- R-49: Redirect to `/login` when any request answers 401. -> **Closed in US-077**.
- R-56: SQLite foreign keys are not enforced. -> **Closed in US-078**.
- R-35: 42 React lint problems (`set-state-in-effect`, `any`). -> **Closed in US-079**.
- Gaps (POS Prodcheck): Native ESC/POS support for thermal printers. -> **Closed in US-080**.
- E1/K1 (US-054 partial): Multi-register concurrency tests on LAN. -> **Closed in US-081**.

---

## Tasks

| US | Name | Status | Date |
|----|------|--------|------|
| US-075 | Binary serves the Next.js SPA correctly (R-55) | [x] Complete | |
| US-076 | Hide the email App password in GET (R-57) | [x] Complete (AC-7 manual browser check pending from the PO) | 2026-10-13 |
| US-077 | Forced redirect when the session expires (R-49) | [x] Complete (manual PO check pending) | 2026-10-20 |
| US-078 | Enable Foreign Keys in SQLite (R-56) | ⬜ Pending | |
| US-079 | Fix 42 Lint Errors (R-35) | ⬜ Pending | |
| US-080 | Native ESC/POS Printing 58/80mm | ⬜ Pending | |
| US-081 | Multi-Register LAN Concurrency and Stability | ⬜ Pending | |

**Total estimated effort: ~6 days**

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-075 | Unblocks testing the rest of the tasks from the real binary instead of the development server. |
| **2** | US-076 | It is the most critical security change and very quick (hide a field in a JSON). |
| **3** | US-077 | Improves everyone's testing experience, avoiding "broken screens" when the token expires. |
| **4** | US-078 | A database driver change that could uncover orphan data in fixtures; best left for the end, running all tests. |
| **5** | US-079 | Pays off the UI debt (avoiding flicker and re-renders) before touching heavy concurrency logic. |
| **6** | US-080 | The printing module is self-contained and of high commercial value; requires testing with hardware/emulators. |
| **7** | US-081 | Full LAN concurrency test; requires the whole system to be stable before simulating stress with 2+ devices. |

---

## US-075 — Binary serves the Next.js SPA correctly (R-55)

**As** the business owner,
**I want** to double-click the program and have the interface work instantly,
**so that** I can use the system without needing the Node console or an internet connection.

**Estimated effort:** 0.5 day. **Impact:** Critical (the software does not open).

**Technical context:**
- `web/static.go` mounts an `http.FileServer` with `embed.FS`.
- It already does a `TrimPrefix` of `/` so that `io/fs.ValidPath` does not fail (half of the fix).
- Missing: Next static exports without a trailing `.html` return 404 in `frontendFS.Open(cleanPath)`, the system falls to the fallback and serves `index.html`, breaking the routes Next exported as `<path>.html` (e.g. `/login.html`).
- `frontend/lib/api.ts:3` has the base URL `/api`, but it must be ensured it does not require strict development CORS, or it must be adjusted.
- `internal/middleware/middleware.go:96` blocks calls that do not come from `:3000`.

**Business rule:**
- If the URL is not `/api/...`, try serving `path`, then `path.html`, and finally fall back to `index.html`.
- The CORS middleware must allow the binary's origin (same domain/port) or be off for local requests where frontend/backend live on the same port.

**Scope — what is NOT included:**
- Modifying the login flow.

**Acceptance Criteria:**
- [x] Build the binary (`make build` or equivalent), run it and open `http://localhost:8080/login`.
- [x] The login screen loads without 404 errors for CSS or JS files.
- [x] Log in and see that the redirect to `/dashboard` works and calls the API.

**DoD:**
- Adjust `frontendFS.Open` in `web/static.go` trying `.html` if the base path fails.
- CORS middleware adjusted in `middleware.go` to work in prod (same host).
- Clean build verified.

---

## US-076 — Hide the email App password in GET (R-57)

**As** a cashier,
**I should not** be able to see the shop's email credentials,
**so that** the theft of the owner's Gmail app password is avoided.

**Estimated effort:** 0.5 day. **Impact:** High (secret leak).

**Technical context:**
- `internal/handler/setup.go` has `getSetup`, which answers the whole `repository.BusinessConfig`, including `OwnerEmailAppPassword`.
- This endpoint is protected only by `Auth` (cashiers have access).

**Business rule:**
- `GET /setup` must not include `owner_email_app_password`. You can serialize it empty, or create a response DTO that omits it.
- You can add a boolean `has_email_app_password: true` so the UI knows it is already configured.
- The update `PUT /setup` (which uses upsert) must not erase the existing password if the payload omits it, unless that is explicitly intended.

**Acceptance Criteria:**
- [x] A logged-in user with the `cashier` role inspects the network on `/setup` and does not see the `owner_email_app_password` field.
- [x] Updating the setup from the UI as admin does not destroy the previously saved password (the email configuration is kept).

**DoD:**
- Modify the response in `getSetup`.
- Adjust logic in `updateSetup` or `repo.Upsert` to ignore overwriting the password with a blank.
- 1 unit test that `GET /setup` does not leak the credential.

**Verified criteria (closed 2026-10-13):**

- AC-1: WHEN any authenticated user (admin or cashier) calls `GET /setup`, the system SHALL return a body without the key `owner_email_app_password` and with `has_email_app_password` (bool). Test: `TestSetup_US076_T1_CashierGetDoesNotLeakPassword` (`internal/handler/setup_test.go`).
- AC-2: WHILE a non-empty App Password is stored, `GET /setup` SHALL return `has_email_app_password: true`; otherwise `false`. Tests: T1 and `TestSetup_US076_T2_NoPasswordStoredReportsFalse`.
- AC-3: WHEN an admin calls `PUT /setup` omitting `owner_email_app_password` or sending `""`, the system SHALL keep the stored password. Test: `TestSetup_US076_T3_OmittedOrEmptyPasswordKeepsStored`.
- AC-4: WHEN an admin calls `PUT /setup` with a non-empty password, the system SHALL store it. Test: `TestSetup_US076_T4_NonEmptyPasswordReplacesStored`.
- AC-5: The `PUT /setup` response SHALL NOT contain `owner_email_app_password` and SHALL contain `has_email_app_password`. Test: `TestSetup_US076_T5_PutResponseDoesNotLeakPassword`.
- AC-6: WHEN the close summary, manual summary or `POST /settings/email/test` need credentials, the system SHALL read the real stored password. Test: `TestSetup_US076_T6_InternalConsumersStillSeeSecret`.
- AC-7: WHILE `has_email_app_password` is true and the admin has not typed a new password, `/settings` SHALL show the field as saved ("Guardada (escriba para cambiar)"), keep "Probar" enabled when `owner_email` is set, and send no password on save. Verified by code reading only (`frontend/app/settings/page.tsx:249,255,295,551,560`); there is no component-test setup. **Pending manual browser check by the PO:** as admin with a saved password, `/settings` shows the "saved" placeholder and "Probar" is enabled; saving with the field blank and reloading keeps it saved; typing a password and saving replaces it; as cashier, the Network tab of `GET /api/setup` shows no `owner_email_app_password`.
- AC-8: IF `GET /setup` fails to read the config, THEN the system SHALL answer 500 `Error al obtener configuración` (unchanged; no dedicated test).

QA: round 1 had one BLOCKER (spec delta not applied), resolved at close. `go test ./...` has one failure that also exists on clean `dev` (`TestDashboardToday_T20_Golden`, R-67). The smoke test was skipped because the binary hardcodes port 8080 (R-71). Lint: 0 new problems.

---

## US-077 — Forced redirect when the session expires (R-49)

**As** a user,
**I want** to be sent to the login if my session expired,
**so that** I do not face empty screens, cryptic errors or a blocked system.

**Estimated effort:** 0.5 day. **Impact:** Medium-High (UX and support).

**Technical context:**
- In `frontend/lib/api.ts`, if `res.ok` is false, it simply does `throw new Error(...)`.
- Components catch this and show a generic error or fail.

**Business rule:**
- If a fetch request returns `HTTP 401 Unauthorized`, clear the session (localStorage, cookies or the existing `logout` function).
- Redirect to `/login` on the client side (e.g. `window.location.href = '/login'`).

**Acceptance Criteria:**
- [x] Enter the POS with an active session.
- [x] Delete the JWT from localStorage (or wait 24h).
- [x] Click any view (Inventory, Dashboard, etc.).
- [x] Be redirected instantly to `/login`.
- [x] The app must not enter an infinite loop if the `/login` page makes a fetch that returns 401.

**DoD:**
- `frontend/lib/api.ts` adjusted.
- Verification that `window.location.pathname !== '/login'` before redirecting (optional but recommended).

**Verified criteria (closed 2026-10-20):**

- AC-1: WHEN any `api.*` request receives HTTP 401, the pathname is not `/login` or `/login/`, and the request path is not `/auth/login`, the system SHALL call `clearAuth()` (removes `pos_token`, `pos_user`, `pos_force_pw`) and navigate with `window.location.assign('/login')`. Tests: T-1, T-8 in `frontend/lib/api.test.ts`.
- AC-2: WHEN a 401 qualifies for redirect, the system SHALL clear and navigate first, then parse the body in a try/catch and reject with the body `error`, or `HTTP 401` if the body is not JSON (never a `SyntaxError`). Tests: T-2, T-3.
- AC-3: IF `/auth/login` answers 401 (wrong password), THEN the system SHALL NOT clear the session nor navigate, and SHALL reject with `credenciales inválidas`. Tests: T-4, T-5, T-8.
- AC-4: IF the pathname is `/login` or `/login/`, THEN a 401 SHALL NOT clear nor navigate (no loop). Tests: T-5, T-6, T-8.
- AC-5: WHEN the status is not 401 (200, 400, 403, 404, 500), behavior SHALL be unchanged and nothing is cleared or navigated. Test: T-7.
- AC-6: WHEN several requests receive 401 at once, the system SHALL navigate at most once per page load. Test: T-9.
- AC-7: WHILE `window` is undefined, a 401 SHALL NOT throw a `ReferenceError`; nothing is cleared or navigated. Test: T-10.
- The redirect decision is the exported pure function `shouldRedirectOnUnauthorized` (`frontend/lib/api.ts`).

Tests T-1..T-10 are all in `frontend/lib/api.test.ts` (63 vitest tests pass in total). The original manual checklist above (delete the token, click a view, land on `/login`) is **pending a manual check by the PO**: delete `pos_token` or set it to `x` (a missing token is already handled by `AppLayout`), click Inventory and expect `/login`; a wrong password on the login form must show `credenciales inválidas` without a page reload.

QA: round 1 OK, 0 BLOCKER, 3 MINOR accepted without items (`redirecting` flag stays set if `assign` throws, not reachable in browsers; a non-JSON 401 on `/auth/login` or `/login` becomes `HTTP 401`; T-10 also skips `clearAuth()` by design). `go test ./...` has one failure that also exists on clean `dev` (`TestDashboardToday_T20_Golden`, R-67). `pnpm lint`: 48 problems on both `dev` and the worktree, none in the touched files. Smoke test skipped (frontend-only change). Spec: no change (client-side UX, no business rule).

---

## US-078 — Enable Foreign Keys in SQLite (R-56)

**As** the system administrator,
**I want** the database not to let me delete tied entities (such as a product with sales),
**so that** my register-close reports do not break in the future.

**Estimated effort:** 0.5 day. **Impact:** High (data integrity).

**Technical context:**
- `internal/database/database.go` uses `?_foreign_keys=on`, which is `mattn/go-sqlite3` syntax but ignored by `modernc.org/sqlite`.
- The `modernc` syntax is `_pragma=foreign_keys(1)`.

**Business rule:**
- Enable `PRAGMA foreign_keys = ON` correctly.
- NOTE: This may break existing tests if they insert invoices using fake `register_id`s or ones not previously inserted. All tests must be run and the orphan fixtures fixed.

**Acceptance Criteria:**
- [ ] Run all backend tests (`go test ./...`) and have them pass with FKs enabled.
- [ ] If a `DELETE` of a `register` with sales is attempted through a direct SQL statement (or a handler), SQLite must raise an FK constraint error.

**DoD:**
- DSN in `database.go` adjusted.
- Test fixtures cleaned (no test inserts orphan data).
- Pipelines/build pass clean.

---

## US-079 — Fix 42 Lint Errors (R-35)

**As** a developer and end user,
**I want** the React code to respect the hooks rules (`set-state-in-effect`),
**so that** the interface does not do unnecessary re-renders that make the app feel "heavy" or flickery.

**Estimated effort:** 1 day. **Impact:** Medium (UX performance / code quality).

**Technical context:**
- The command `pnpm run lint` throws 42 errors, mainly in `AppLayout.tsx`, `useIsMobile.ts`, `ThemeProvider.tsx`.
- The common error is updating React state synchronously inside a `useEffect` (`react-hooks/set-state-in-effect`).

**Business rule:**
- Move the initial reads (such as the current theme or matchMedia) into the initial state of `useState` (or the lazy initializer `useState(() => ...)`), or use an asynchronous event handler so it does not collapse the React 18+ render.
- Remove unjustified `any` types (type as `unknown` or the proper interface).

**Acceptance Criteria:**
- [ ] Running `pnpm run lint` in `/frontend` must return zero errors and zero warnings.
- [ ] Navigate the app and verify there is no flicker when opening the side menu (AppLayout) or when switching from light to dark mode (ThemeProvider).

**DoD:**
- Modifications to the problematic hooks.
- Clean CI/Build (`pnpm lint` && `pnpm build`).

---

## US-080 — Native ESC/POS Printing 58/80mm

**As** a cashier or waiter,
**I want** the sale ticket or the order ticket to come out instantly from my thermal printer on checkout,
**so that** I do not have to click through the Windows print dialog or configure margins in the browser.

**Estimated effort:** 2 days. **Impact:** Very High (critical commercial feature).

**Technical context:**
- The browser print dialog (`window.print()`) is currently used.
- For ESC/POS, Go will have to communicate over the network (LAN to the printer's IP) or through a USB/Serial connection (using a thermal-command library such as `github.com/alexbrainman/printer` on Windows or generic raw printing).

**Business rule:**
- A backend endpoint must be added (`POST /api/printer/print-ticket/:sale_id`).
- The business configuration (Settings) must allow choosing the print method: `[Browser, ESC/POS Local, ESC/POS Network]`.
- If network, store IP and port. If local, store the name of the shared Windows printer.

**Acceptance Criteria:**
- [ ] In the POS configuration, the user can enable "Impresión Directa" (direct printing).
- [ ] On charging or reprinting a ticket, the backend dispatches the raw ESC/POS command.
- [ ] The printer outputs the paper immediately with no pop-ups on screen.
- [ ] The logo (if possible), the header, the items and the footer from the business configuration are printed correctly.

**DoD:**
- ESC/POS printing API (`internal/service/printer.go`).
- Frontend interface to configure the printer.
- Unit test simulating the thermal command generator.

---

## US-081 — Multi-Register LAN Concurrency and Stability

**As** the owner of the restaurant or shop,
**I want** several employees to be able to charge and use the system at the same time from different phones/tablets on the same WiFi network,
**so that** service is faster without data overwriting each other or accounts breaking.

**Estimated effort:** 1 day. **Impact:** High (scalability of real operation).

**Technical context:**
- US-054 enabled LAN access. SQLite is configured with `WAL` (Write-Ahead Logging) and a `5000ms` timeout, which supports concurrent read/write decently.
- It must be validated in a real environment that there are no locking failures (`database is locked`) under simultaneous load.

**Business rule:**
- If two registers deduct from the same product, the logic must handle transactions well (since the service does it in a Tx, this should be covered, but the frontend must be validated).
- Reinforce error handling: if the frontend receives `HTTP 503` or `locked`, it must retry automatically (retry policy in `api.ts`).

**Acceptance Criteria:**
- [ ] Load the POS on one PC and on 2 phones through the LAN IP.
- [ ] Simulate two charges from different tables/customers in the exact same second.
- [ ] Validate that no request dies with a white crash; if there is a collision, a *toast* warns that there was an error or the app retries invisibly.
- [ ] Review the register close to confirm that all amounts of both sessions match perfectly with no phantom records.

**DoD:**
- Add retry logic in `frontend/lib/api.ts` for requests that fail due to temporary locking.
- Console validations that no Go panics occur when handling concurrent transactions.

---

## Sprint success criterion
- The Windows `.exe` binary can be launched and used 100% on localhost:8080 without NodeJS.
- No cashier can steal the email password.
- Expired sessions redirect cleanly.
- The database is robust against accidental deletion of connected data.
- The React codebase is free of warnings and lint debt.
- The POS is able to spit out thermal tickets instantly without browser intervention.
- The system does not crash or corrupt when 3 employees are charging at the same time.

## Technical notes for the dev
- On US-076: Check whether the `BusinessConfig` struct is already used directly as request and response. It is advisable to separate them, or blank the field in the response and check in the PUT whether it comes empty to keep the previous value by doing a merge before the `Upsert`.
