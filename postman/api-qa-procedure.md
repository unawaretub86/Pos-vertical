# Procedimiento QA de API — POS Vertical

Complementa `qa-checklist.md` (pruebas manuales de UI). Esto es para probar el
backend (`internal/handler`) directo, sin pasar por el frontend — util para
aislar si un bug es de API o de UI.

## 0. Bugs confirmados corriendo la colección (2026-07-29)

Corrida completa contra backend real (`go build ./cmd/pos`, DB SQLite limpia).
329 asserts, 325 pasan. Los 4 que fallan son 2 bugs reales de servidor,
no problemas de la colección:

1. **`PUT /users/me/password` siempre rechaza la clave correcta.**
   `internal/repository/user.go:39-48` (`UserRepository.GetByID`) no incluye
   `password_hash` en el SELECT. `ChangePassword` (`internal/service/user.go:127`)
   compara con `bcrypt.CompareHashAndPassword` contra un hash vacío → siempre
   falla con "contraseña actual incorrecta", sin importar la clave real.
   **Ningún usuario puede cambiar su propia contraseña.** Fix: agregar
   `password_hash` al SELECT de `GetByID`.

2. **`DELETE /sales/:id/void` revienta si la venta tiene caja abierta asociada.**
   `internal/repository/sale.go:281` inserta un movimiento de caja con
   `type = 'withdrawal'` al anular una venta, pero el CHECK constraint de
   `register_movements` (`internal/database/migrate.go:245`) solo permite
   `'sale', 'cash_in', 'cash_out', 'opening'`. Toda anulación con caja abierta
   falla con `constraint failed` (400), la venta NO queda anulada, y el stock
   tampoco se revierte. Fix: usar `'cash_out'` (o agregar `'withdrawal'` al
   CHECK constraint vía migración).

Ambos son bloqueantes para producción — el segundo en particular rompe el
caso de uso más común (anular una venta durante el turno con caja abierta).

## 1. Importar en Postman

1. Postman → **Import** → arrastrar los dos archivos:
   - `postman/POS-Vertical.postman_collection.json`
   - `postman/POS-Vertical.postman_environment.json`
2. Arriba a la derecha, seleccionar el environment **"POS Vertical - Local"**.
3. Confirmar que `baseUrl` = `http://localhost:8080` (puerto real de `cmd/pos/main.go`).

## 2. Levantar el backend

```
go run ./cmd/pos
```

o usar `pos.exe` ya compilado. Confirmar que `GET /health` responde `{"data":{"status":"ok"}}` antes de seguir.

## 3. Orden de ejecución recomendado

La colección está numerada en carpetas (`01` a `13`) en el orden en que hay que
correrla — varios requests dependen de datos creados por el anterior (guardan
IDs automáticamente en variables de entorno vía **Tests scripts**).

| Orden | Carpeta | Qué hace | Variables que setea |
|---|---|---|---|
| 1 | Health | Smoke test | — |
| 2 | Auth | Login guarda `authToken` (usado por Bearer en todo lo demás) | `authToken` |
| 3 | Setup | Config del negocio | — |
| 4 | Categories | Crea categoría | `categoryId` |
| 5 | Products | Crea producto, entrada de stock | `productId`, `barcode` |
| 6 | Registers | Abre caja, movimientos, cierre | `registerId` |
| 7 | Sales | Venta efectivo y fiada | `saleId` |
| 8 | Customers | Cliente, pagos, saldo | `customerId` |
| 9 | Dashboard | Lee agregados (no depende de nada nuevo) | — |
| 10 | Reports | Reportes por rango de fechas | — |
| 11 | Fiscal | Config DIAN/Factus | — |
| 12 | Users | Crea usuario, cambia clave | `userId` |
| 13 | Casos negativos | Sin token / token inválido / rol incorrecto | — |

Correr con **Runner** (colección completa) o carpeta por carpeta a mano.
Con Runner: seleccionar la colección → Run → dejar orden tal cual (no alfabetizar).

## 3.1 Asserts automatizados (Tests scripts)

Cada uno de los 67 requests trae asserts (`pm.test`) en su pestaña **Tests**,
además de 3 checks globales que corren en todos (status < 500, Content-Type
JSON, tiempo de respuesta < 2s). Qué valida cada uno según el tipo de endpoint:

- **Status code exacto** (200/201/4xx/409 según corresponda) — no solo "no truena".
- **Forma de la respuesta**: campos esperados presentes, arrays vs objetos,
  tipos (`number`, `string`, `boolean`).
- **Valores de negocio calculados por el servidor**: totales de venta
  (`subtotal`/`total`), stock después de una entrada, orden de `low_stock` por
  déficit, orden de `aging_receivables` por días de mora, `gross_margin =
  total_revenue - total_cost`.
- **Casos negativos explícitos** (requests nuevos, uno por caso):
  `PUT /setup` sin name, `POST /products` con wholesale < cost,
  `PUT /categories/:id` inexistente, `POST /registers/.../movements` con
  `type` inválido, `POST /sales` fiado sin `customer_id`, doble `void`, doble
  `close`, `void-with-credit-note` sin `motive`, `from` inválido en reportes,
  y los 3 de la carpeta `13 - Casos negativos` (sin token / token inválido /
  rol incorrecto).

Correr con **Collection Runner** y revisar la pestaña de resultados: verde =
passed, rojo = failed con el assert que rompió. No hace falta leer cada
response body a mano — el runner ya dice cuál campo no coincidió.

### Correr headless con Newman (opcional, para CI o antes de cada entrega)

```
npm install -g newman
newman run postman/POS-Vertical.postman_collection.json \
  -e postman/POS-Vertical.postman_environment.json \
  --reporters cli
```

Newman corre todos los asserts y devuelve exit code != 0 si algo falla —
sirve para meterlo en un script de "antes de entregar al cliente".

## 4. Qué NO automatiza esta colección (probar a mano)

- `POST /registers/:id/close` y `DELETE /categories/:id`/`DELETE /products/:id`
  quedan al final de sus carpetas a propósito — correrlos borra/cierra estado
  que otros requests todavía necesitan. No correr "toda la colección de un
  tirón" en un ambiente que ya tiene datos reales.
- `PUT /users/me/password` cambia la clave de la cuenta que estás usando
  (`{{username}}`/`{{password}}`) — después de correrlo, actualizar la variable
  `password` o el próximo login falla.
- Casos que requieren estado externo (Factus/DIAN real, lector de código de
  barras físico, apagón de PC a mitad de venta) — no son de API, quedan en
  `qa-checklist.md`.

## 5. Casos ya identificados como críticos (de `qa-checklist.md`)

Priorizar estos con la colección antes de aprobar una entrega:

- **Venta fiada sin `customer_id`** → debe bloquear (`07 - Sales` → request
  "fiado - requiere customer_id", correr con el campo vacío/omitido).
- **Doble apertura de caja** → `POST /registers` dos veces seguidas sin cerrar
  → segunda debe dar `409`.
- **Anular venta con factura DIAN aceptada** → `DELETE /sales/:id/void` debe
  dar `409` con `requires_credit_note: true` en vez de anular directo.
- **Rol `cashier` contra endpoints admin** → carpeta `13 - Casos negativos`,
  request "POST /users como cashier" → debe dar `403`, no `200`.
- **Pago de cliente mayor al saldo** → `POST /customers/:id/payments` con
  `amount` > deuda actual → definir y verificar comportamiento esperado.
- **Producto sin costo (`cost: 0`)** → no debe romper `GET /dashboard/today`,
  solo debe excluirse de `gross_profit` y sumar a `unconfigured_cost_count`.

## 6. Reportar bugs encontrados

Mismo formato que `qa-checklist.md`: anotar debajo de la sección
correspondiente, con request usado, body enviado, respuesta recibida y
respuesta esperada.
