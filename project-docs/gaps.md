> **Type:** current · **Rules over:** what is missing for production · **Updated:** 2026-11-06

This document lists **verified** limitations and debt of the current product; what will be built in the future lives in [roadmap.md](roadmap.md), which is a different thing.

## Open debt (not roadmap, it is what was left half-done)

| US / Sprint | What | Since |
|---|---|---|
| US-034 | Factus: full sandbox validation | SPRINT-07 — requires external access |
| SPRINT-05 | Mobile Responsive, marked pending although the style base is already in use | SPRINT-05 |
| US-042c, US-043, US-045 | Sales speed, customer history in the POS, WhatsApp catalog | SPRINT-09 |
| US-053 | Register-close summary by email — there is uncommitted work in `internal/mailer/` | SPRINT-12 |
| US-054 | POS access over local WiFi | SPRINT-12, reassigned to SPRINT-13 |
| US-055 to US-058 | Full restaurant module | SPRINT-13 |
| US-059 to US-062 | Local assistant | SPRINT-14 |

## Pending PO validations (Andrey, manual)

Code and tests are done for these; only a human check on a real browser or device is missing. Parked by the PO on 2026-10-20 until he has time. Remove a row when the check is done.

| US | What to check | Where it is described |
|---|---|---|
| US-083 M-1 | AC-9, the part of T-M1 that QA did not exercise. Create a customer with no phone and give them debt (one fiado sale of $45.000). On that customer's detail, `Cobrar` is shown but disabled, at reduced opacity, with the tooltip `Agrega teléfono al cliente`. On the Customers list, that row has no WhatsApp icon, only `Abonar`. In the compiled binary the customer detail may not load at all (`Cliente no encontrado`, R-92); if so, record it under R-92 and do this check once R-92 is fixed. | `SPRINT-19-Fiado-Cobranza.md` (US-083) |
| US-082 M-1 | AC-14, close modal and ticket. Open a register with $100.000, make a cash sale of $30.000, register a cash *abono* of $20.000 from a customer's detail, then `Cerrar Caja`: the modal shows `Abonos en efectivo` $20.000 and `Total esperado` $150.000; after closing, the printed close ticket shows the same line. Repeat with a register that has no *abonos*: the line is still there with `$0`. | `SPRINT-19-Fiado-Cobranza.md` (US-082) |
| US-082 M-2 | AC-17. Log in as a cashier with no open register, open a customer with debt, `Registrar abono`, enter an amount and save: the modal stays open and shows `Debe haber una caja abierta para registrar un abono`. As admin with no open register, save a cash *abono*: the toast `Este abono no quedará en ningún cierre de caja` appears after `Abono registrado exitosamente`. | same |
| US-082 M-3 | AC-18. At 320 px (DevTools device mode) the *abono* modal with the `Efectivo` / `Transferencia` / `Tarjeta` selector fits without horizontal scroll, in dark and light themes (colors follow the theme, nothing hard-coded). | same |
| US-080 P-1 | Network ESC/POS printer. In Settings pick `ESC/POS red`, enter the printer IP and port 9100, save, press `Imprimir prueba`: paper comes out at once with `PRUEBA DE IMPRESION`, no browser dialog, and is cut. Unplug the printer and press it again: a red toast `No se pudo imprimir: ...` within about 3 s. | `SPRINT-18-Estabilizacion-Produccion.md` (US-080), `spec/domain/impresion.md` |
| US-080 P-2 | Windows queue printer (USB) via winspool, never executed by any test. On the Windows PC running `pos.exe`, install the printer driver, pick `ESC/POS Windows`, type the exact queue name from Windows "Printers & scanners", save, press `Imprimir prueba`: paper comes out and is cut. Type a wrong name: 502 toast. If the spooler hangs, note it (the job has no timeout, R-81). | same |
| US-080 P-3 | Paper width. With a 58 mm printer and `58 mm`, then an 80 mm printer and `80 mm`: charge one cash sale with 2 items, a long product name and an accented one (e.g. "Café", "Piña", "Jamón"). Lines do not wrap unexpectedly, the subtotal column is flush right, `TOTAL` is bold and tall, accents print correctly (not `?` or garbage; if garbled, the printer may not support CP858). | same |
| US-080 P-4 | Flow after a sale. In `/pos` and in a table checkout (`/mesas`) with an ESC/POS method: the ticket prints once by itself, the preview stays on screen with toast `Ticket enviado a la impresora`, and `Imprimir` prints a second copy. Switch back to `Navegador`: the browser dialog opens as before. Charge a credit sale (fiado): `*** CREDITO ***` and the customer print. | same |
| US-079 M-1 | Theme, layout and drawer. Set `localStorage.pos_theme = 'light'` and reload: first paint is dark and light applies after mount, same as on `dev`, with no second dark frame. Toggle the theme, reload: it persists. Set `pos_theme = 'garbage'`: the app uses dark. At 800 px wide resize to 700 px: mobile layout appears and returns when widened. In mobile layout open the drawer and click a menu link: the drawer closes. | `SPRINT-18-Estabilizacion-Produccion.md` (US-079) |
| US-079 M-2 | Sort icons. On `/customers` and `/portfolio` click each sortable header: same ordering and same icon/direction as on `dev`. | `SPRINT-18-Estabilizacion-Produccion.md` (US-079) |
| US-079 M-3 | Smoke of `/pos`, `/inventory`, `/users`, `/barra`, `/reports`, `/purchases`, `/suppliers`, `/audit`: each loads its data on first render and the Network tab shows the same requests (URLs and count) as on `dev`. `/barra` polls every 3 s and the interval restarts only when the sound toggle changes. | `SPRINT-18-Estabilizacion-Produccion.md` (US-079) |
| US-077 | In DevTools set `pos_token` to `x`, click any view: lands on `/login`. Wrong password on `/login`: shows "credenciales inválidas" without reloading. | `SPRINT-18-Estabilizacion-Produccion.md` (US-077), contract §4 |
| US-076 | AC-7: as a cashier, inspect the network on `/setup` in a browser and confirm `owner_email_app_password` is absent; as admin, save Settings and confirm the stored email password is kept. | `SPRINT-18-Estabilizacion-Produccion.md` (US-076) |
| US-078 | Confirm 3 product choices made by the Tech Lead: customers deleted logically (`active = 0`), delete answers 409 when the balance is above 0, credit sales (`fiado`) to inactive customers are rejected. Open the app on an existing database and check that migration 19 runs. | `roadmap.md` R-56 and decision log 2026-10-13 |
| US-062 | M-1..M-9 on a real phone: visual, theme, 320 px, keyboard. | `workflow.md` SPRINT-14 row |
| US-054 | Open `/login` from the compiled binary and from a phone or tablet over WiFi (AC-16). | `roadmap.md` R-55 |
| US-074 | Responsive 320 px on a phone or emulator. | `workflow.md` SPRINT-17 row |
| US-034 | Factus full sandbox validation (needs external access). | Open debt table above |

## Documents with outdated information

| Gap | Evidence | Status |
|---|---|---|
| `archive/2026-05-plan-mvp.md` reports a `cash_in` bug in the register close | `archive/2026-05-plan-mvp.md:1134` vs `internal/service/register.go:113` | **Already fixed**, the doc is what is outdated |

## To verify

Gaps that are suspected but not confirmed against the code go here. Each one is confirmed with `file:line` before moving up to the main table.
