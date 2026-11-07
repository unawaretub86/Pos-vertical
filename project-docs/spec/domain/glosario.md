# Glossary

> **Type:** spec · **Current:** yes · **Rules over:** domain vocabulary
> **Updated:** 2026-11-07 (US-084: Saldo a favor) · 2026-11-05 (US-080: Tirilla, ESC/POS, Impresión directa)

Colombian shop terms and their equivalent in the code. It exists so that a US and a table speak the same language, and for the local assistant's phrase bank. The Spanish term is the canonical name; docs in English use it with this gloss.

| Shopkeeper term (Spanish) | What it means | Where it lives in the code |
|---|---|---|
| **Fiar / fiado** | Selling on credit, the customer pays later | `sale_payments.method = 'credit'` |
| **Cartera** | The total owed to the business (receivables) | Sum of balances with `balance > 0` |
| **Abono** | Partial payment against the debt, not against an invoice | `customer_payments` |
| **Saldo a favor** | Credit in the customer's favor: a balance below zero (overpaid). Not part of the cartera | Balance `< 0`; UI `Saldo a favor: $X` |
| **Cuadrar / cuadre** | Counting the cash at close and comparing it to what is expected | `registers.expected_close` vs `actual_close` |
| **Arqueo** | Same as the cuadre, more formal term | Register close |
| **Base** | The cash the shift starts with (opening float) | `registers.opening_amount` |
| **Faltante / sobrante** | Negative / positive difference of the cuadre (shortage / surplus) | `registers.difference` |
| **Quincena** | Payday, the 15th and 30th. **It is the fiado collection cycle.** | Not modeled yet — roadmap `R-03` |
| **Tirilla** | The printed receipt | `TicketPreview`, `CloseTicketPreview`; ESC/POS output in `internal/service/printer.go`. See [impresion.md](impresion.md) |
| **ESC/POS** | The raw command set of thermal receipt printers (Epson standard) | `internal/service/printer.go` (`BuildTicket`) |
| **Impresión directa** | Direct printing: the server sends ESC/POS bytes to the printer, with no browser print dialog | `business_config.print_method` = `escpos_network` / `escpos_windows`. See [impresion.md](impresion.md) |
| **Nequi / Daviplata** | Colombian digital wallets, widely used | Today they fall under `method = 'transfer'` |
| **Mayorista** | Special price by volume or customer type (wholesale) | `products.wholesale_price` |
| **Lo que más deja** | The highest-margin product, distinct from the best seller | See [rentabilidad.md](rentabilidad.md) |
| **Plata** | Money. Used far more than "dinero" in real speech | — |
| **Se está acabando** | Product below minimum stock | `stock < min_stock` |

## Note for end-user text

The product speaks to the shopkeeper the way their supplier does, not the way a bank does. UI copy (in Spanish): "plata" before "dinero", "cuánto te deben" before "cartera vigente". Formal vocabulary is reserved for fiscal documents, where it is expected. (This note is about the Spanish UI strings; all docs remain in English.)
