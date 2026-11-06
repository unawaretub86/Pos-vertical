# Domain — Ticket printing (tirilla)

> **Type:** spec · **Current:** yes · **Rules over:** how the sale ticket (tirilla, printed receipt) reaches the printer, printer settings, ESC/POS content and encoding
> **Updated:** 2026-11-05 (US-080: direct ESC/POS printing) · Verified against the code (`internal/service/printer.go`, `printer_windows.go`, `printer_other.go`, `internal/handler/printer.go`, `internal/handler/setup.go`, `internal/database/migrate.go`, `frontend/lib/printFlow.ts`, `frontend/lib/printerSettings.ts`).

## Print methods

`business_config.print_method` picks how a sale ticket is printed (migration 20, `internal/database/migrate.go:577-585`):

| Method | Settings label | What happens |
|---|---|---|
| `browser` (default) | `Navegador` | The on-screen preview (`TicketPreview`) and `window.print()`, as before US-080. The server sends nothing to any printer. |
| `escpos_network` | `ESC/POS red` | The server opens a raw TCP connection to `printer_host:printer_port` (default port 9100) and writes ESC/POS bytes. Dial timeout 3 s, write timeout 10 s (`internal/service/printer.go:28-29`, `:355-370`). |
| `escpos_windows` | `ESC/POS Windows` | The server writes a `RAW` job to the Windows print queue named `printer_name` through `winspool.drv` (hand-written `syscall.NewLazyDLL`, no CGO, no new dependency; `internal/service/printer_windows.go:12-66`). Only on a Windows server (`printer_other.go` returns `ErrUnsupported`); on any other OS it answers 501. |

ESC/POS (the raw command set of thermal receipt printers) is also called **impresión directa** (direct printing) in the UI: no browser dialog, no margin setup.

Other settings, same table: `paper_width_mm` (`58` default, or `80`), `printer_host` (`''`), `printer_port` (`9100`), `printer_name` (`''`). Existing databases migrate to these defaults, so a shop keeps browser printing until an admin changes it.

Unsupported: USB direct (libusb), serial/COM, Bluetooth, macOS/Linux CUPS, printer discovery. A USB printer is used through its Windows queue.

## Who can do what

- Only an admin edits the printer settings: they travel in `PUT /setup`, which sits in the `RequireAdmin` group (non-admin gets 403 and nothing is stored).
- Any authenticated role (cashier, waiter, admin) prints a sale ticket (`internal/handler/handler.go:166`, authenticated group).
- Only an admin runs the test print (`internal/handler/handler.go:175-179`, `RequireAdmin` group).

## Settings validation (`PUT /setup`)

`normalizePrinterSettings` (`internal/handler/setup.go:117-146`, called at `:53`) runs before the save; on error it returns 400 with a Spanish message and stores nothing.

- An empty/absent `print_method` becomes `browser`; an absent/zero `paper_width_mm` becomes 58. **Consequence:** `Upsert` rewrites every column, so a `PUT /setup` that omits the printer fields resets the method to `browser`. The Settings form always sends all five fields (`buildPrinterPayload`, `frontend/lib/printerSettings.ts`).
- `print_method` must be `browser`, `escpos_network` or `escpos_windows`; `paper_width_mm` must be 58 or 80.
- `escpos_network`: `printer_host` not blank and `printer_port` in 1..65535.
- `escpos_windows`: `printer_name` not blank.
- The port is checked only for `escpos_network`. With another method an out-of-range port is accepted as sent, and a port of 0 is stored as 9100.

Settings shows the IP/port fields only for `escpos_network` and the printer-name field only for `escpos_windows`; hidden values are still sent back unchanged.

## Endpoints

### `POST /api/printer/print-ticket/{sale_id}` (any authenticated role)

Checks run in this order (`internal/handler/printer.go:40-66`; outcome mapping in `sendToPrinter`, `:28-38`):

| Case | Code |
|---|---|
| No or invalid token | 401 |
| `sale_id` not an integer, or `<= 0` | 400 `id inválido` (before any DB or printer access) |
| `print_method = browser` | 409 `La impresión directa no está activada en la configuración` (nothing written). This check comes before the sale lookup, so a missing sale in browser mode also answers 409. |
| No sale row with that id | 404 `venta no encontrada` |
| `escpos_windows` on a non-Windows server | 501 |
| Printer unreachable (connect timeout, refused, Windows queue not found) or write fails | 502 `No se pudo imprimir: <cause>` |
| Printed | 200 `{"printed": true}` |

The ticket data is the same as `GET /sales/{id}/ticket`: both call `buildTicket` (`internal/handler/sale.go:179`). Sales have no logical-delete column, so a voided sale is printable and carries the `*** ANULADO ***` banner.

A print failure never touches the sale: the sale was committed by `POST /sales` before printing starts.

### `POST /api/printer/test` (admin only)

Prints a short sample through the **saved** config: business name, `PRUEBA DE IMPRESION`, method, paper width, date/time, feed and cut. Same 409 / 501 / 502 / 200 semantics as above; non-admin gets 403. The Settings `Imprimir prueba` button is shown only for ESC/POS methods (hidden in `Navegador` mode, where the endpoint would answer 409; `frontend/app/settings/page.tsx:543`) and is disabled while the form has unsaved printer changes (`isTestPrintDisabled`). The contract's AC-1 listed the button unconditionally; hiding it was accepted at QA (Gemini MINOR).

## Serialization

`PrinterService` (`internal/service/printer.go:389-406`) holds one mutex per printer key (`host:port` for network, the queue name for Windows) for the whole job, so two tickets for the same printer are never interleaved on the wire; different printers print in parallel. The locks live in memory in the single `pos.exe` process.

## Frontend flow

- WHEN a sale is created from `/pos` (`frontend/app/pos/page.tsx:770`) or from a table checkout (`/mesas/{orderId}`, `OrderClient.tsx:141`) and the method is not `browser`, the frontend calls `POST /printer/print-ticket/{id}` once, right after it loads the ticket for the preview. No print dialog opens. Success: toast `Ticket enviado a la impresora`. Failure: error toast with the server message. In both cases the preview stays on screen.
- In ESC/POS modes the preview `Imprimir` button calls the same endpoint (a second copy); in `browser` mode it calls `window.print()` exactly as before. This button is the only reprint path today (no reprint from sales history, see roadmap).
- The method is read from the business-config context (`frontend/lib/business-config-context.tsx`), refreshed after Settings is saved in the same browser. Another device keeps the old method until it reloads (roadmap).
- If loading the ticket fails after the sale, there is no auto-print (the sale is still saved and a toast `Venta completada` shows).

## ESC/POS ticket content

`BuildTicket` (`internal/service/printer.go:239`) mirrors `TicketPreview`, in this order:

1. `ESC @` (`1B 40`) init, then `ESC t 19` (`1B 74 13`, codepage CP858).
2. `*** ANULADO ***` (bold, centered) when `status = 'voided'`.
3. Business name (bold, centered), address, `NIT: …`, header text, divider.
4. `Venta #id` with the date, `Método: …` with the time; for mixed payments one line per payment.
5. Credit sale (fiado): `*** CREDITO ***`, `Cliente: …`, `Tel: …`.
6. One block per item: the name, then `qty unit x price` (plus ` -$discount` when the line has one) with the subtotal right-aligned.
7. `Descuento -$X` and `IVA $X` only when > 0; `TOTAL` bold and double height.
8. Cash sales: `Recibido` and `Cambio` (bold). `Ref: …` when there is a reference. Footer text, divider, date and time.
9. Feed 3 lines (`1B 64 03`) and partial cut `GS V 1` (`1D 56 01`).

Layout: 32 columns at 58 mm and 48 columns at 80 mm (Font A). The left text is truncated so the right column never overflows; long centered text is word-wrapped. Money is `$1.234.567` (no decimals, dot thousands).

Not on the ESC/POS ticket: the logo, QR/CUFE of the electronic invoice, cash-drawer kick. The register-close ticket (`CloseTicketPreview`) still prints through the browser only.

## Encoding

CP858, fixed (no setting; `internal/service/printer.go:47-77`). `0x20-0x7E` map to themselves, `\n` is `0A`, `\r` is dropped, `\t` becomes a space. Spanish characters: `á A0`, `é 82`, `í A1`, `ó A2`, `ú A3`, `ü 81`, `ñ A4`, `Á B5`, `É 90`, `Í D6`, `Ó E0`, `Ú E9`, `Ü 9A`, `Ñ A5`, `¿ A8`, `¡ AD`, `€ D5`. Any other rune becomes `?` (`3F`); the encoder never panics.

## Browser path

Unchanged by US-080: the print CSS still forces `@page { size: 58mm auto }` (`frontend/app/globals.css`) and the preview is built at 58 mm; `paper_width_mm` only affects ESC/POS.

## Tests

- Generator and serialization: `TestPrinterBuildTicket_T1_Layout58mm` .. `_T6_DiscountAndTaxOnlyWhenPositive`, `TestPrinterService_T12_SamePrinterJobsAreSerialized`, `TestPrinterService_T12_DifferentPrintersRunConcurrently` (`internal/service/printer_test.go`).
- Endpoints: `TestPrintTicket_T7`..`_T10`, `TestPrintTest_T13_AdminTestPrint` (`internal/handler/printer_test.go`), `TestPrintTicket_T11_WindowsMethodOffWindowsReturns501` (`internal/handler/printer_other_test.go`, non-Windows only).
- Settings and migration: `TestSetup_US080_T14_PrinterSettingsPersist`, `TestSetup_US080_T15_PrinterSettingsValidation` (`internal/handler/setup_test.go`), `TestMigration20_T16_PrinterSettingsDefaults` (`internal/database/migrate_test.go`).
- Frontend: `frontend/lib/printFlow.test.ts`, `frontend/lib/printerSettings.test.ts`.
- Not automated: the winspool success path and real paper output (manual, `gaps.md`).
