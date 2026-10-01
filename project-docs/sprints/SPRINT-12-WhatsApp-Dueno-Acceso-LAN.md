# SPRINT-12 — Owner Summary and POS Access from the Phone

**Goal:** Give Don Carlos (owner, not always at the shop) visibility into the business from his phone without depending on the cloud or a monthly fee: a register-close summary delivered to the owner automatically, and the ability to open the POS from his phone while he is at the shop, connected to the same WiFi.

**Start date:** (to be defined)

**References:** `project-docs/better-finds-pos.md` (block K — K1, K2/I2), `project-docs/SPRINT-10-Cobro-Mixto-Auditoria-Gastos.md` (roadmap), `project-docs/SPRINT-11-QA-Proveedores-Cierre-Caja.md` (roadmap)

**Incoming technical debt:**
- US-034 — Factus sandbox validation (open since SPRINT-07, requires external access to Factus) — **still open**, non-blocking, no action in this sprint.
- SPRINT-05 — Mobile Responsive (marked ⬜ Pending in `BACKLOG-Development-Workflow.md`, now `workflow.md`) — **still open**, unrelated to this sprint (although this sprint does require 320px responsive in what it touches).
- US-042c, US-043, US-044, US-045 (SPRINT-09, ⬜ Pending) — **still open**. US-044 (supplier order suggestion) has not been blocked since SPRINT-11 closed (US-052 provided suppliers), but its implementation itself does not enter here.

**Size note:** this sprint is 2 USs of ~0.5 day each (the source doc itself documents them as a "1-day combo" in block K). It is intentionally short — both previous sprints (10 and 11) already fixed this specific combination in their Roadmap; it is not inflated with SPRINT-13 features (H2 layaway, G2 kardex) so as not to break that decision already made twice.

---

## Tasks

| US | Name | Status | Date |
|----|------|--------|------|
| US-053 | Register-close summary to the owner (I2) | [ ] Pending | — |
| US-054 | POS access from the phone over local WiFi (K1) | [ ] Pending | — |

---

## Recommended implementation order

| Order | US | Reason |
|-------|----|--------|
| **1** | US-053 | Purely additive: a new column and a button. Does not touch how the server starts — zero risk of leaving the POS unreachable. |
| **2** | US-054 | Touches the server bind (`cmd/pos/main.go`) — the only change in this sprint with real blast radius (if something goes wrong, nobody gets into the POS). Done afterwards, with US-053 already tested and migration 14 already in place. |

---

## US-053 — Register-close summary by automatic email to the owner (I2, pivot)

**As** Don Carlos (owner, not always present at the shop),
**I want** to receive by email the register-close summary (sold per method, cash in, cash out, expected vs. actual) automatically as soon as Laura closes the register, without anyone having to click anything,
**so that** I know how the day went without depending on the cashier remembering to send it.

**Estimated effort:** 0.5 day. **Impact:** High — covers the need for an "owner mobile app" without workflow friction.

**Pivot (2026-09-12):** the original version of this US used a "Send by WhatsApp" button (`wa.me`, manual cashier click). In manual testing, Andrey found that the browser silently blocked the WhatsApp popup (no console error) and, above all, that an extra manual step in the close flow was not the desired UX — he explicitly asked for something automatic and integrated. Alternatives for WhatsApp automation were evaluated (Meta's Business Cloud API: requires business verification and charges per conversation; unofficial libraries like `whatsapp-web.js`: violate WhatsApp's ToS, ban risk) and discarded. The decision was to switch channel to **email via Gmail SMTP**, sent automatically from the backend on register close — no click, no cost (within Gmail's free limit), no new libraries (uses Go stdlib `net/smtp`). Note: this does require internet at close time for the email to go out (unlike click-to-WhatsApp, which did not require it until the owner opened the message) — if there is no internet, sending fails silently and is only logged on the server; the register close itself is never blocked or retried.

**Technical context (implemented):**
- **Migration 14** (shared with US-054 — see US-054 for the second block):
  ```sql
  ALTER TABLE business_config ADD COLUMN owner_email TEXT;
  ALTER TABLE business_config ADD COLUMN owner_email_app_password TEXT;
  ```
- `internal/repository/setup.go` — `BusinessConfig` gains `OwnerEmail` and `OwnerEmailAppPassword`; `Get()`/`Upsert()` include both columns. No new endpoint — `PUT /setup` already accepts the full body.
- `internal/mailer/mailer.go` (new package) — pure, tested `BuildRegisterSummaryEmail(data) (subject, body string)` (Go, not frontend), and `Send(cfg SMTPConfig, subject, body string) error` which sends through `smtp.gmail.com:587` (STARTTLS via `net/smtp.SendMail`, no external dependency). The same Gmail address acts as sender and recipient — a single pair of credentials to configure.
- `internal/handler/register.go` (`closeRegister`) — after closing the register successfully, it fires `go h.sendRegisterSummaryEmail(summary, cashier)` in a goroutine (fire-and-forget): if `owner_email`/`owner_email_app_password` are not configured, or sending fails (no internet, invalid credentials), it is logged with `log.Printf` and does not affect the HTTP response — the close was already persisted before attempting the send.
- `frontend/app/settings/page.tsx` — replaces the phone field with two `ConfigRow`s under Administration: "Correo del dueño" (`type="email"`) and "Contraseña de aplicación" (`type="password"`, with a note that it is the Gmail app password, not the account's normal password). (Quoted labels are UI strings, Spanish.)
- `frontend/components/CloseTicketPreview.tsx` and `frontend/app/register/page.tsx` — the WhatsApp button and the `ownerPhone` plumbing are reverted; the close ticket goes back to having only the "Imprimir" button (the email was already sent on its own, no cashier action is needed).
- `frontend/lib/registerSummaryMessage.ts` (created in the WhatsApp attempt) is removed — the logic for building the message now lives in Go (`internal/mailer`), not in the frontend, because sending is no longer triggered by the browser.

**Business rule (same as before, now in the email body; the email text itself is Spanish product content):**
```
body = "Resumen de cierre — <business.name>\n"
     + "Cajero: <cashier> · <closed_at formatted>\n\n"
     + for each line in by_payment_method: "<label(method)>: <formatCOP(total)>\n"
     + "Entradas: <formatCOP(cash_in)>\n"
     + "Salidas: <formatCOP(cash_out)>\n"
     + "Esperado: <formatCOP(expected)>\n"
     + "Real: <formatCOP(actual_close)>\n"
     + if difference < 0 → "Faltante: <formatCOP(abs(difference))>"
       else              → "Diferencia: <formatCOP(difference)>"
```

**Scope — what is NOT included:**
- Multiple recipients (several owners/partners) — a single `owner_email`, extendable if requested.
- Retries or a send queue on failure (no internet, expired credentials) — it is logged server-side and that is it; there is no "emails pending resend" UI.
- Sending an attached PDF/image of the ticket — the email is plain text, same as the WhatsApp message was.

**Acceptance Criteria:**

**Configuration:**
- [x] "Correo del dueño" and "Contraseña de aplicación" fields in Settings → Administration, saved with `PUT /setup` like the rest of the form.
- [x] Empty by default — does not break already-configured businesses (new columns with `NULL`/`''`).

**Register close:**
- [x] When closing the register with `owner_email` and `owner_email_app_password` configured, the backend sends the summary email automatically — no click or action from the cashier.
- [x] Without `owner_email` configured: the close works exactly the same (nothing is attempted, no visible error).
- [x] If sending fails (no internet, invalid credentials): the register close completes anyway, the error is only left in the server log.
- [x] Close with a negative difference (shortage) → email body says "Faltante: $X", not "Diferencia: -$X".

**DoD:**
- [x] Migration 14 (shared with US-054): columns `owner_email` and `owner_email_app_password` in `business_config` — see exact SQL above.
- [x] `GET /setup` and `PUT /setup` (already existing) include both fields in the JSON, with no change of route or auth group.
- [x] Test (Go, `internal/mailer`): `BuildRegisterSummaryEmail` with a sample close — sales $60,000 (cash $30,000 + transfer $20,000 + credit $10,000), cash_in $0, cash_out $5,000, expected $25,000, actual_close $24,500 → body contains "Faltante: $500" and the 3 method lines with their exact amounts.
- [x] Test (Go, `internal/mailer`): same case but `actual_close` = `expected` exactly (difference 0) → body says "Diferencia: $0", not "Faltante: $0".
- [x] Clean build (`go build ./...`, `npm run build`).
- [ ] Manual test (not automatable without a real Gmail account — pending for Andrey): configure a real email + app password, close the register, confirm the email arrives with the correct amounts.

---

## US-054 — POS access from the phone over local WiFi (K1)

**As** Don Carlos (owner, present at the shop),
**I want** to open the POS from my phone's browser while connected to the shop's WiFi, without installing anything,
**so that** I can see the day's sales live without standing in front of the register computer.

**Estimated effort:** 0.5 day. **Impact:** Medium-high — covers "see sales on phone" live, complements US-053 (which covers the summary outside the shop).

**Verified state:**
- `cmd/pos/main.go:41` has a hardcoded `addr := "localhost:8080"` — a phone on the same WiFi as the register cannot get in today (`localhost` only accepts connections from the same machine).
- The frontend has been responsive/mobile-first since SPRINT-05 (although that US is still marked ⬜ Pending in the general backlog, the mobile style base was already used and validated in SPRINT-11 for the new components).
- Login uses JWT (`internal/middleware`, confirmed by use of `appmw.GetClaims` in multiple handlers) — the app already demands authentication even if the port is exposed on the LAN.
- No endpoint exists that reports the machine's local IP (`grep -rn "InterfaceAddrs\|network-info" internal/` finds nothing).

**Technical context:**
- **Migration 14** (same migration as US-053 — a single `migration14` file with both `ALTER TABLE`s):
  ```sql
  ALTER TABLE business_config ADD COLUMN lan_access_enabled INTEGER NOT NULL DEFAULT 0;
  ```
  Default `0` = current behavior unchanged for every existing database.
- `internal/repository/setup.go` — `BusinessConfig` gains `LanAccessEnabled bool \`json:"lan_access_enabled"\``, included in `Get()`/`Upsert()` like `AllowGlobalDiscount` (same boolean pattern already used in that same table).
- `cmd/pos/main.go` — extract a pure function `listenAddr(lanEnabled bool) string` (`"0.0.0.0:8080"` if `true`, `"localhost:8080"` if `false`), called after `database.New(...)` and before `http.ListenAndServe`, reading the flag through `h.setup` (or the repo directly) right after the `Handler` is created. **The change only applies on restarting the binary** — Go already has the socket open at the original bind; there is no way to re-bind without restarting the process.
- New endpoint `GET /api/network-info` (admin group, same pattern as `/api/audit`) → `{lan_ip: string, port: string, enabled: bool}`. `lan_ip` is obtained with the standard Go trick: open a `net.Dial("udp", "8.8.8.8:80")` (sends no real traffic, it just makes the OS resolve which interface it would use) and read `LocalAddr()`; if it fails (no network at all), return `lan_ip: ""` and the frontend shows "No se detectó red local" instead of breaking.
- `frontend/app/settings/page.tsx` — new `ConfigRow` "Acceso desde celular (WiFi local)": `lan_access_enabled` toggle (saved with the same `PUT /setup` of the existing form); below it, when enabled, `GET /api/network-info` and show `http://<lan_ip>:8080` in large monospace font + a "Copiar" button; fixed warning text: "Solo funciona dentro de la WiFi de este local. Asegúrate que tenga contraseña. Debes reiniciar la aplicación para que el cambio tome efecto." (UI strings, Spanish.)

**Business rule:** N/A (configuration flag + network detection, no arithmetic).

**Scope — what is NOT included:**
- QR code — the source doc suggests it, but adding a new library just for this is not justified compared to showing the IP in large, copyable text; if a real customer asks for it, it is a small, isolated change in the same component.
- Remote access outside the LAN (tunnel/cloud) — that is K4 in the source doc, explicit `v2`, requires 5+ days and contradicts the offline positioning.
- Hot-reload of the bind without restarting — a real limitation of `http.ListenAndServe`, documented in the UI instead of being solved with extra complexity (e.g. an internal reverse proxy).

**Acceptance Criteria:**

**Configuration:**
- [ ] "Acceso desde celular" toggle in Settings, off by default — existing databases do not change behavior without explicit action from the owner.
- [ ] With the toggle on, the local IP + port and the restart notice are shown.
- [ ] If no local network is detected (`lan_ip` empty), "No se detectó red local" is shown instead of an empty IP or a raw error.
- [ ] Responsive: section usable at 320px.

**Real access:**
- [ ] With the toggle on and the application restarted, a phone connected to the same WiFi opens `http://<lan_ip>:8080` and sees the login screen.
- [ ] With the toggle off (default), the server keeps listening only on `localhost:8080` — behavior identical to the current one.
- [ ] Login from the phone works the same as from the register computer (same JWT, same roles).

**DoD:**
- [ ] Migration 14 (shared with US-053): column `lan_access_enabled` — see exact SQL above.
- [ ] `GET /api/network-info` (admin group) → `{lan_ip, port, enabled}`.
- [ ] `GET /setup` and `PUT /setup` include `lan_access_enabled`.
- [ ] Test (Go, `cmd/pos` or extracted package): `listenAddr(true) == "0.0.0.0:8080"`, `listenAddr(false) == "localhost:8080"`.
- [ ] Test (Go, `internal/handler`): `GET /api/network-info` without an admin session → 403; with an admin session → 200 with `port: "8080"`.
- [ ] Manual test (not automatable without two devices on a network): turn on the toggle, restart `pos.exe`, confirm access from a real phone on the same WiFi — **pending for Andrey**, same criterion as the GitHub Actions checks of US-050 that do not run in this environment.
- [ ] Clean build (`go build ./...`, `npm run build`).

---

## Sprint success criterion

- Laura closes the register, Don Carlos receives (without being at the shop) the exact summary of how the day went — sold per method, cash in, cash out, whether money was left over or missing.
- Don Carlos, standing in his shop with his phone, turns on LAN access once, restarts the app, and from then on enters the POS from his phone on the same WiFi to see sales live without touching the register computer.

---

## Technical notes for the dev

- **US-053 and US-054 share migration 14** — a single `migration14` file/function with both `ALTER TABLE`s, same criterion SPRINT-10 applied to US-046/US-049.
- The WhatsApp button of US-053 (now removed) and the `whatsappUrl` of `frontend/app/customers/page.tsx` should share the `57` prefix and the digit-cleaning logic (`replace(/\D/g, '')`) — it is worth extracting a common `lib/whatsapp.ts` helper instead of duplicating it a third time if a fourth use case appears.
- `listenAddr()` in US-054 must remain a pure exported function (not inline in `main()`) specifically so it can be tested without starting a real server — `main.go` has no tests today because all its content is orchestration; this is the first piece of logic there worth isolating.
- Warn Andrey in the demo: the US-054 toggle requires a manual restart of the `.exe` — it is not a bug, it is a limitation of how `net/http` binds the socket. If in the future the toggle should apply live, the real solution is a second conditional listener or an internal proxy — not worth it for this sprint.

---

## Roadmap

What remains to be built lives in [roadmap.md](../roadmap.md), which is the single source. This document does not reserve future sprint numbers.

> There used to be a roadmap table of its own here. It was consolidated on 2026-09-21: the per-sprint copies diverged from each other and ended up claiming the same sprint number for different features.
