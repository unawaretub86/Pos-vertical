# SPRINT-18 — Stabilization and Production Blockers

**Goal:** Eliminate the 4 critical blockers (security, data consistency, static serving and session) that prevent delivering the binary to a real customer. This sprint introduces no new business features; it lays a solid foundation so the system does not crash or compromise information on first use.

**Start date:** (to be defined)

**References:** `project-docs/roadmap.md`, `project-docs/gaps.md`

**Incoming technical debt:**
- R-55: The binary does not serve the embedded frontend: blank page on `:8080`. -> **Closed in US-075**.
- R-57: `GET /setup` exposes `owner_email_app_password` to any authenticated user. -> **Closed in US-076**.
- R-49: Redirect to `/login` when any request answers 401. -> **Closed in US-077**.
- R-56: SQLite foreign keys are not enforced. -> **Closed in US-078**.
- R-35: 48 React lint problems (36 errors, 12 warnings; `set-state-in-effect`, `any`, `static-components` and others). -> **Closed in US-079**.
- Gaps (POS Prodcheck): Native ESC/POS support for thermal printers. -> **Closed in US-080**.
- E1/K1 (US-054 partial): Multi-register concurrency tests on LAN. -> **Closed in US-081**.

---

## Tasks

| US | Name | Status | Date |
|----|------|--------|------|
| US-075 | Binary serves the Next.js SPA correctly (R-55) | [x] Complete | |
| US-076 | Hide the email App password in GET (R-57) | [x] Complete (AC-7 manual browser check pending from the PO) | 2026-10-13 |
| US-077 | Forced redirect when the session expires (R-49) | [x] Complete (manual PO check pending) | 2026-10-20 |
| US-078 | Enable Foreign Keys in SQLite (R-56) | [x] Complete in code (PO validation pending) | 2026-10-13 |
| US-079 | Fix 48 Lint Problems (R-35) | [x] Complete in code (manual PO checks M-1..M-3 pending) | 2026-10-28 |
| US-080 | Native ESC/POS Printing 58/80mm | [x] Complete in code (manual PO hardware checks P-1..P-4 pending; logo and reprint deferred) | 2026-11-05 |
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

## US-079 — Fix 48 Lint Problems (R-35)

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
- [x] Running `pnpm run lint` in `/frontend` must return zero errors and zero warnings. (QA: 48 -> 0.)
- [ ] Navigate the app and verify there is no flicker when opening the side menu (AppLayout) or when switching from light to dark mode (ThemeProvider). **Pending manual PO check M-1** (`gaps.md`).

**DoD:**
- [x] Modifications to the problematic hooks.
- [x] Clean CI/Build (`pnpm lint` && `pnpm build`); CI now runs `pnpm lint` (`.github/workflows/test.yml:35`).

**Verified criteria (closed 2026-10-28):**

The measured baseline was 48 problems (36 errors, 12 warnings) on `dev`, not 42, and they were mostly in page files rather than in `AppLayout`/`useIsMobile`/`ThemeProvider` (4 of 48).

- AC-1: WHEN `pnpm lint` runs in `frontend/`, the system SHALL exit 0 with 0 errors and 0 warnings. QA run: exit 0, 0 problems.
- AC-2: WHEN `pnpm build` runs, the system SHALL succeed. QA run: succeeded, all routes prerendered.
- AC-3: WHEN `pnpm test` runs, the system SHALL pass all tests. QA run: 7 files, 71 tests passed (baseline 63 plus 8 new).
- AC-4/AC-9/AC-14: no `eslint-disable`, `@ts-ignore`, `@ts-expect-error` or explicit `any` added; `eslint.config.mjs` untouched; unused bindings removed, not renamed with `_`. Checked by QA with grep on the diff.
- AC-5: WHEN `localStorage['pos_theme']` is `'light'`, the system SHALL use light; for any other value (absent, `'dark'`, unknown, empty) it SHALL use dark. This is the only intentional behavior change (before, any stored string was cast to `Theme`). Tests: `frontend/lib/theme.test.ts` (`readTheme`).
- AC-6/AC-6b: `useIsMobile` and `AppLayout` share `breakpointQuery(breakpoint)`; `breakpointQuery(768)` = `(max-width: 767px)`; the server snapshot is `false` and `matchMedia` is not touched during render. Tests: `frontend/lib/breakpoint.test.ts`.
- AC-7, AC-8, AC-13, AC-15: the drawer closes on pathname change, `SortIcon` moved to module level, data-load effects keep their dependency arrays (barra polling restarts only when `soundEnabled` changes; users `load` deps `[toast]` with a stable `toast`) and so the same requests. Verified by QA code reading only; behavior checks are M-1, M-2, M-3 below.
- AC-10: the diff touches only `frontend/**`, `.github/workflows/test.yml` and docs; no backend, migration, `package.json` or lockfile.
- AC-11: the 4 `<img>` sites (`CatalogClient`, `settings`, `setup`, `Sidebar`) use `next/image` with `unoptimized` and explicit width/height, same `src`, `alt` and rendered size.
- AC-12: the `frontend` job runs `pnpm lint` (`.github/workflows/test.yml:35`).

New tests: `frontend/lib/theme.test.ts`, `frontend/lib/breakpoint.test.ts`.

**Pending, PO decision Q-3:** manual checks M-1 (theme and mobile first paint, drawer close on navigation, resize), M-2 (sort icons on `/customers` and `/portfolio`) and M-3 (smoke of `/pos`, `/inventory`, `/users`, `/barra`, `/reports`, `/purchases`, `/suppliers`, `/audit`: same requests as `dev`) are NOT done; no browser was available to QA. Exact steps in `gaps.md`.

QA: round 1 OK, 0 BLOCKER, 3 MINOR (latent drawer state keyed by pathname; `data-theme` effect re-runs after hydration, same net behavior as `dev`; a throwing `localStorage` now throws in render, out of scope -> R-75). `go test ./...` has one failure that also exists on clean `dev` (`TestDashboardToday_T20_Golden`, R-67). Gemini diff review: clean. Spec: no change (code-quality refactor, no business rule; `git diff dev...HEAD` touches nothing under `project-docs/spec/`).

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
- [x] In the POS configuration, the user can enable "Impresión Directa" (direct printing). (A 3-way method selector; `Navegador` = off.)
- [x] On charging a ticket the backend dispatches the raw ESC/POS command; the preview `Imprimir` button sends a second copy. Reprint from sales history is deferred (no such flow exists) -> roadmap R-76.
- [ ] The printer outputs the paper immediately with no pop-ups on screen. No dialog opens in ESC/POS mode (tests); real paper output is **pending manual PO check** P-1..P-4 (`gaps.md`).
- [x] The header, the items and the footer from the business configuration are printed (bytes verified by tests and by QA smoke on a TCP listener). The logo was skipped by PO decision -> roadmap R-77.

**DoD:**
- [x] ESC/POS printing API (`internal/service/printer.go`).
- [x] Frontend interface to configure the printer (`frontend/app/settings/page.tsx`).
- [x] Unit test simulating the thermal command generator (`internal/service/printer_test.go`).

**Verified criteria (closed 2026-11-05):**

Discrepancies resolved at GATE 1: no reprint flow exists (D-1, deferred); route is chi `{sale_id}` (D-2); no third-party printing library and no CGO: hand-written `winspool.drv` via `syscall.NewLazyDLL` (D-3); paper width is a new setting, ESC/POS only (D-4); there is no separate order ticket, table checkout prints the same sale ticket (D-5); the toggle is a 3-way method selector (D-6); logo skipped (D-7). Spec: `spec/domain/impresion.md`.

- AC-1: WHEN an admin opens Settings, the system SHALL show the method selector (`Navegador` default, `ESC/POS red`, `ESC/POS Windows`), paper width 58/80 mm, method-specific fields (IP + port 9100, or Windows printer name) and, for ESC/POS methods only, `Imprimir prueba` (hidden in `Navegador` mode, `frontend/app/settings/page.tsx:543`; accepted deviation from the contract text, Gemini MINOR). Tests: `frontend/lib/printerSettings.test.ts` (`buildPrinterPayload`, `isTestPrintDisabled`); layout by code reading.
- AC-2: WHEN an admin saves Settings, the system SHALL persist `print_method`, `paper_width_mm`, `printer_host`, `printer_port`, `printer_name`; existing databases migrate to `browser`, 58, 9100, `''`, `''`. Tests: `TestSetup_US080_T14_PrinterSettingsPersist` (`internal/handler/setup_test.go`), `TestMigration20_T16_PrinterSettingsDefaults` (`internal/database/migrate_test.go`), `printerSettings.test.ts`.
- AC-3: IF a non-admin calls `PUT /setup`, THEN 403; IF the method or width is invalid, or network has no host or a port outside 1..65535, or Windows has no name, THEN 400 and nothing stored (port checked only for the network method). Test: `TestSetup_US080_T15_PrinterSettingsValidation`.
- AC-4: WHEN any authenticated user calls `POST /api/printer/print-ticket/{sale_id}` in an ESC/POS method, the system SHALL send the ticket bytes and return 200 `{"printed": true}`. Test: `TestPrintTicket_T7_NetworkPrintsSale` (`internal/handler/printer_test.go`).
- AC-5: WHILE `print_method = browser`, the endpoint SHALL return 409 and write nothing. Test: `TestPrintTicket_T8_BrowserModeReturns409`.
- AC-6: IF `sale_id` is not an integer or `<= 0`, THEN 400; no row, 404; no token, 401; a voided sale prints with `*** ANULADO ***`. Test: `TestPrintTicket_T9_IdValidationAndVoided`. (In browser mode the 409 is returned before the sale lookup, QA MINOR.)
- AC-7: IF the printer is unreachable or the write fails, THEN 502 with the cause, and the sale stays committed. Test: `TestPrintTicket_T10_UnreachablePrinterReturns502` (network only; the Windows failure path is not executed in CI).
- AC-8: IF the server is not Windows and the method is `escpos_windows`, THEN 501 (print-ticket and test). Test: `TestPrintTicket_T11_WindowsMethodOffWindowsReturns501` (`internal/handler/printer_other_test.go`).
- AC-9: WHILE a job is in progress for a printer key, a second job for the same key SHALL wait; different keys run in parallel. Tests: `TestPrinterService_T12_SamePrinterJobsAreSerialized`, `TestPrinterService_T12_DifferentPrintersRunConcurrently` (`internal/service/printer_test.go`).
- AC-10: WHEN an admin calls `POST /api/printer/test`, the system SHALL print `PRUEBA DE IMPRESION` through the saved config (200); 409 browser, 502 printer failure, 501 Windows off Windows, 403 non-admin. Tests: `TestPrintTest_T13_AdminTestPrint`, `TestPrintTicket_T11_...`.
- AC-11, AC-12, AC-13: WHEN a sale is created from `/pos` or table checkout in an ESC/POS method, the system SHALL call the print endpoint once, open no print dialog, keep the preview and show a success or error toast; in `browser` mode the flow is unchanged; in ESC/POS modes the `Imprimir` button calls the endpoint instead of `window.print()`. Tests: `frontend/lib/printFlow.test.ts` (`shouldAutoPrint`, `handleTicketPrint`).
- AC-14, AC-16, AC-17: the generator SHALL emit init + CP858 codepage, header, items, `Descuento`/`IVA` only when > 0, bold double-height `TOTAL`, `Recibido`/`Cambio`, footer, feed 3 and partial cut, wrapped to 32 (58 mm) / 48 (80 mm) columns, money as `$1.234.567`. Tests: `TestPrinterBuildTicket_T1_Layout58mm`, `_T2_Layout80mm`, `_T3_LongNameTruncated`, `_T6_DiscountAndTaxOnlyWhenPositive`.
- AC-15: text SHALL be CP858 with the fixed Spanish table and `?` for any other rune, never panicking. Test: `TestPrinterBuildTicket_T4_CP858Encoding`.
- AC-18: a voided sale SHALL start with `*** ANULADO ***`; a credit sale SHALL print `*** CREDITO ***` and the customer name/phone. Test: `TestPrinterBuildTicket_T5_VoidedAndCredit`.

QA: round 1 OK, 0 BLOCKER, 5 MINOR (stale `print_method` on other LAN sessions until reload -> R-80; `ctx` ignored by `WindowsTransport.Send` and winspool only cross-compiled -> R-81; 409 before 404 in browser mode -> R-82; spec delta, applied at close; `Imprimir prueba` hidden in browser mode while contract AC-1 listed it unconditionally, found by the Gemini diff review, accepted and recorded in `spec/domain/impresion.md`). `go build ./...`, `GOOS=windows go build ./...`, `go test ./...`, `pnpm lint` (0 problems), `pnpm test` (10 files, 90 tests) and `pnpm build` passed. Smoke on port 18082 with a TCP listener: test print bytes start `1B 40 1B 74 13` and end `1B 64 03 1D 56 01`; a real sale ticket was not smoked (covered by T-7..T-9). Out of scope, sent to roadmap: reprint from history (R-76), logo (R-77), cash drawer (R-78), register-close ticket over ESC/POS (R-79).

**Pending, PO (contract Q-9):** real hardware checks P-1..P-4 in `gaps.md`. Q-10 (keep the 2-day estimate) is still open in the contract; no re-baseline recorded.

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
