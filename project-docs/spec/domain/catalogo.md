# Domain — Product catalog (products, categories, catalog import)

> **Type:** spec · **Current:** yes · **Rules over:** what a product is, barcode and category uniqueness, how stock is traced, how a catalog is imported from CSV
> **Updated:** 2026-11-13 (US-102: whole-peso prices and cost on the manual form and API) · 2026-11-10 (US-093: pointer to the go-live reset) · 2026-11-09 (US-097: singular wording for count 1 in the catalog import UI; US-094: catalog import from CSV) · Verified against the code on that date (`internal/model/model.go:71-87`, `internal/database/migrate.go:170-192`, `:302-303`, `internal/service/product.go`, `internal/service/product_import.go`, `internal/repository/product_import.go`, `internal/handler/product_import.go`, `frontend/lib/productImport.ts`, `frontend/components/ProductImportModal.tsx`).

The catalog is master data: the US-093 reset keeps products, categories and the stock loaded by an import, and rebuilds `products.stock` from `stock_entries` (rule B, [reset.md](reset.md)). Stock that no entry explains (manual product form, R-153) is lost unless re-entered; the reset dry-run lists it.

---

## Product

| Field | Rule |
|---|---|
| `name` | Required, trimmed. **Not unique** for the manual form (two active `Leche` may coexist); the import refuses duplicates (below). |
| `code` (*código interno*) | Free text, not unique. |
| `barcode` | Optional. Empty is stored as NULL, so many products may have none. Unique among non-NULL values **including deleted products** (`idx_products_barcode` has no `active` filter, `migrate.go:303`), while `GET /products/barcode/{barcode}` finds only active ones. A deleted product's barcode therefore cannot be reused (roadmap R-151). |
| `category_id` | Optional (NULL = no category); must exist (foreign key). |
| `type` | `product` or `service`. |
| `unit` | `unit`, `kg`, `lb` or `liter` (CHECK + service). |
| `sale_price`, `cost`, `wholesale_price` | Whole pesos (`REAL` columns that only hold integers, [dinero.md](dinero.md)). The manual form and the API reject a fraction with 400 `el precio de venta / el costo / el precio mayorista debe ser un número entero de pesos` (US-102). |
| `stock`, `min_stock` | `REAL`; `stock < min_stock` is *se está acabando* (running low, `glosario.md`). |
| `tax_rate` | IVA (Colombian VAT) as a fraction: 0, 0.05, 0.19. The form shows a percentage. |
| `active` | Logical delete: `DELETE /products/{id}` sets `active = 0`. |

Known gap: the manual create/edit form validates name, `sale_price ≥ 0`, type, unit and (since US-102) whole-peso prices and cost, but still accepts a negative cost, stock and minimum stock (roadmap R-150, now only that part). The import is stricter.

## Category

Names are `UNIQUE` with binary collation: `Bebidas` and `bebidas`, or `Lácteos` and `Lacteos`, are different categories (roadmap R-152). The import matches categories ignoring case and accents, so it never adds another variant.

## Stock traceability

| Who changes stock | `stock_entries` row | Audit row |
|---|---|---|
| Purchase (`Compra #N`) | yes | no |
| Manual stock entry (`Entrada de inventario`) | yes | no |
| Catalog import | yes, `Importación de catálogo`, one per product with stock > 0 | one `product_import` per import |
| Sale / void | no (stock guarded in the sale, restored on void, `caja.md`) | `void` on void |
| Manual product create or edit | **no** (roadmap R-153) | no |

The go-live reset (`reset.md`) sets every `product`-type product (inactive included) to `SUM(stock_entries)`, which is why untraced stock is reported before it is lost.

---

## Catalog import

`POST /api/products/import`, admin only (inside the `RequireAdmin` products group; cashier → 403, no token → 401). It loads a product list from a spreadsheet or from another POS program's export. Decided at GATE 1 (PO, 2026-11-09): a CSV with automatic column recognition plus manual mapping; no `.xlsx`, no SQL or SQLite dump; no update of existing products (roadmap R-154).

### Request

JSON `{"file_name", "content_base64", "mapping": object|null, "dry_run": bool}`. The file goes base64 in JSON because the frontend `api` client only sends JSON.

- Request body over 3 MiB, or decoded file over 2 MiB → 413 `el archivo supera 2 MB`.
- Malformed JSON or base64, or an empty file → 400 `archivo inválido`.

### Reading the file

- **Encoding:** a UTF-8 BOM is stripped; valid UTF-8 is read as UTF-8, anything else as Windows-1252 (Spanish Excel's "CSV (delimitado por comas)"); reported in `encoding`. A UTF-16 BOM → 400 `formato no compatible: guarda el archivo como CSV UTF-8 o CSV (delimitado por comas)`. An `.xlsx` (`PK\x03\x04`) → 400 `es un archivo de Excel (.xlsx): ábrelo en Excel y guárdalo como CSV`.
- **Delimiter:** `;`, `,` or tab, whichever appears most often outside quotes in the first non-blank line (tie order `;`, `,`, tab). `;` is Excel's default in Colombia. RFC 4180 quoting, lazy quotes (`Pizza 12"` is kept), ragged rows allowed.
- **Rows:** the first non-blank record is the header. All-blank records are skipped. A row's number is the physical line where it starts, so it matches the Excel row. No data rows → 400 `el archivo no tiene filas de productos`; more than 5,000 → 400 `el archivo supera 5.000 productos; divídelo en partes`. A short row gets blank cells; a non-blank cell beyond the header width is the row error `la fila tiene más columnas que el encabezado`.

### Column mapping

Headers are normalized (trim, lowercase, accents and `ñ` removed, `. _ - / % ( ) :` to spaces, spaces collapsed) and matched against synonyms. The leftmost match wins; other headers are returned in `ignored_headers`.

| Field | Label | Synonyms |
|---|---|---|
| `name` * | nombre | nombre, producto, descripcion, nombre del producto, articulo, name, product |
| `sale_price` * | precio de venta | precio de venta, precio venta, precio, pvp, valor, valor venta, price, sale price |
| `barcode` | código de barras | codigo de barras, codigo barras, cod barras, barcode, ean, ean13, upc |
| `category` | categoría | categoria, linea, grupo, departamento, familia, category |
| `unit` | unidad | unidad, unidad de medida, um, medida, unit |
| `cost` | costo | costo, precio de compra, precio compra, costo unitario, cost |
| `stock` | stock | stock, existencia, existencias, cantidad, inventario, stock actual, qty |
| `min_stock` | stock mínimo | stock minimo, minimo, existencia minima, min stock |
| `tax_rate` | IVA | iva, impuesto, tax, porcentaje iva |
| `code` | código interno | codigo, codigo interno, referencia, ref, sku, code |

An explicit `mapping` `{field: header}` replaces the automatic one (exact header text after trimming). Errors, all 400: unknown field → `campo desconocido: <key>`; header not in the file → `la columna «<h>» no existe en el archivo`; one header for two fields → `la columna «<h>» está asignada a dos campos`; `name` or `sale_price` unmapped → `faltan columnas obligatorias: nombre, precio de venta` (only the missing ones).

### Row rules

Each error is `{row, field, column, message}`, where `column` is the file's header text.

- **Name:** trimmed, inner spaces collapsed; blank → `el nombre es obligatorio`; over 200 characters → `el nombre supera 200 caracteres`.
- **Money (`sale_price`, `cost`), whole pesos:** spaces, `$` and `COP` are removed, and a leading `-` is kept apart as the sign. Then: digits only; or thousands grouped by `.` or `,` (`1.234.567`, `1,234,567`); or a last `.`/`,` followed by 1–2 digits is a decimal separator that must be zero (`12.500,00` → 12500, `2500.00` → 2500). Precedence: invalid text (`-abc` → `valor inválido: escribe solo números, por ejemplo 2.500`) and cents (`2.500,50` → `el valor debe ser en pesos enteros, sin centavos`) win over the sign; a parsed number with a `-` (also `-0`) → `el valor no puede ser negativo`. Above 100,000,000 → `el valor supera $100.000.000`. The sale price is required (blank → `el precio de venta es obligatorio`) and at least $1 (0 → `el precio de venta debe ser mayor que 0`). A blank cost is 0.
- **Quantities (`stock`, `min_stock`):** `,` is the decimal separator, `.` grouped in threes is thousands (`1.500` → 1500), any other single `.` is decimal (`3.5`). Blank → 0. Invalid → `cantidad inválida: escribe solo números, por ejemplo 24 o 2,5`; negative → `la cantidad no puede ser negativa`; over 1,000,000 → `la cantidad supera 1.000.000`; more than 3 decimals → `máximo 3 decimales`; not whole for `unit` → `un producto por unidad necesita una cantidad entera`.
- **Unit:** blank, unit, unidad, und, un, u, unidades → `unit`; kg, kilo(s), kilogramo(s) → `kg`; lb, libra(s) → `lb`; liter, litro(s), lt, l → `liter`; anything else → `unidad inválida: usa unidad, kg, lb o litro`.
- **IVA:** `0`, `5`, `19` with an optional `%`, blank = 0; else `IVA inválido: usa 0, 5 o 19`.
- **Barcode:** trimmed, Excel's `="…"` unwrapped, blank → none. Scientific notation (`7,70123E+12`) → `Excel convirtió el código de barras a notación científica: formatea la columna como Texto y vuelve a escribirlo`; over 64 characters → `el código de barras supera 64 caracteres`. Leading zeros already dropped by Excel cannot be detected.
- **Duplicate barcode:** repeated in the file → `código de barras repetido en la fila <n>`; already in `products` → `el código de barras ya existe en el producto «<name>»`, plus ` (eliminado)` for a deleted product. Existing products are never changed.
- **Duplicate name** (normalized as the headers): repeated in the file → `producto repetido en la fila <n>`; equal to an active product → `ya existe un producto con este nombre`. A deleted product's name does not block. When a row matches both an existing product and an earlier row, only the "already exists" message is shown.
- **Category:** trimmed; blank → none. Matched to existing categories ignoring case and accents (lowest `id` wins when several match). Unmatched names become new categories, one per normalized name, spelled as in their first row, listed in `new_categories` in order of first appearance.
- **Code:** trimmed; over 64 characters → `el código interno supera 64 caracteres`.

### Dry run and commit (all-or-nothing)

- **Dry run** (`dry_run: true`) → 200 with `{file_name, encoding, delimiter, headers, mapping, ignored_headers, total_rows, valid_rows, error_count, errors, new_categories, valid, preview}`, and nothing is written. `errors` is sorted by row, then by the field order of the mapping table, and capped at 200 (`error_count` is the full count). `preview` holds the first 5 valid rows.
- **Commit** (`dry_run: false`) re-validates against the database as it is at that moment. Any error → 422 `el archivo tiene <n> errores; no se importó nada` (`el archivo tiene 1 error; no se importó nada` when exactly one row has an error), and nothing is written. A second commit of the same file is therefore rejected (duplicate barcodes or names).
- **When the file is valid**, one transaction writes:
  - the new categories;
  - every row as an active `product` (`wholesale_price` NULL), in file order, so ids are consecutive;
  - one `stock_entries` row `Importación de catálogo` per product with stock > 0;
  - one `audit_log` row: action `product_import`, entity `product`, `entity_id` = first product id, details `{file_name, rows, products_created, categories_created, stock_entries_created, first_product_id, last_product_id}`.

  The answer is 201 `{products_created, categories_created, stock_entries_created, first_product_id, last_product_id}`.
- **If any write fails**, everything rolls back:
  - `SQLITE_BUSY` → 503 `El sistema está ocupado, reintenta en un momento`;
  - another user created the same barcode during the commit → 409 `otro usuario creó un producto con el código de barras <barcode> mientras importabas; vuelve a revisar el archivo`;
  - another user created the same category during the commit → 409 `otro usuario creó la categoría «<name>» mientras importabas; vuelve a revisar el archivo`;
  - anything else → 500 `error importando productos`.

### UI

- Inventario → Productos shows `Importar productos` next to `Nuevo producto` only when `canImportProducts(getUser())` is true (admin). A cashier who types `/inventory` sees no button and no modal.
- The modal `Importar productos` shows the help text `Sube un archivo CSV. Desde Excel: Archivo › Guardar como › CSV UTF-8.`, a `Descargar plantilla` link and a file picker (`.csv`, `.txt`).
- The file is read as bytes (so Windows-1252 survives), base64-encoded and dry-run with no mapping. The modal shows one select per field (required ones marked `*`), each with the file headers plus `(no importar)`, preset to the returned mapping. `Revisar de nuevo` dry-runs again with the chosen mapping.
- When the backend recognized no required column it answers 400 with no report. The modal then reads the header row itself (`readHeaderLine`, `frontend/lib/productImport.ts`: same UTF-8 / Windows-1252 and delimiter rules as the backend) and offers an empty manual mapping. Known divergence: a BOM followed by a non-UTF-8 body keeps the BOM in the first header (roadmap R-156).
- Errors show as a `Fila | Columna | Error` table with `y <k> errores más` (`y 1 error más` when one error is hidden), and the commit button stays disabled. A valid file shows `<n> productos listos para importar` (`1 producto listo para importar`), `Categorías nuevas: …` and `Importar <n> productos` (`Importar 1 producto`). Until a valid file with at least one row is picked, the commit button reads plain `Importar` and is disabled (`Importando...` while committing). A success shows the toast `Se importaron <n> productos` (`Se importó 1 producto`) and reloads products and categories; any error stays in the modal.
- `plantilla-productos.csv` is built in the browser with a UTF-8 BOM, `;` and CRLF. Header: `Nombre;Código de barras;Categoría;Unidad;Costo;Precio de venta;Stock;Stock mínimo;IVA;Código interno`. Example rows: `Arroz Diana 500 g` and `Queso campesino`. Every header auto-maps.
- In the Auditoría page, a `product_import` row reads `Importación de productos` with `<n> productos importados de «<file>»` (`1 producto importado de «<file>»` for one), and the action filter offers it.

Tests: `internal/service/product_import_test.go` (`TestProductImport_T3`, `_T4`, `_T6`..`_T9`, `_T17`..`_T19`), `internal/handler/product_import_test.go` (`TestProductImport_T1`, `_T2`, `_T5`, `_T10`..`_T16`, `_T20`..`_T22`), `frontend/lib/productImport.test.ts` (F-1..F-6). Manual browser checks: `gaps.md` US-094 M-1.
