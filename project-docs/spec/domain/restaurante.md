# Dominio — Restaurante (modo restaurante y mesas)

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** modo restaurante, mesas y estado de mesa
> **Actualizado:** 2026-09-30 · Verificado contra el código en esa fecha (US-055: `internal/repository/dining_table.go`, `internal/handler/tables.go`, `frontend/lib/nav.ts`; US-056: `internal/repository/order.go`, `internal/handler/order.go`, `internal/handler/handler.go:213-219`).

Módulo para cafeterías y restaurantes: mesas, pedidos y pantalla de barra (SPRINT-13). Hoy existen la gestión de mesas, el interruptor y los pedidos por mesa (abrir, ítems, anular, cancelar); cobro y barra llegan en US-057 y US-058 y agregarán sus reglas aquí.

---

## Modo restaurante

- `business_config.restaurant_mode` (0/1, default 0) decide si el negocio usa el módulo. Se activa en Ajustes con el toggle "Modo restaurante" (solo admin, vía `PUT /api/setup`).
- Apagado: no se muestra la sección de mesas en Ajustes ni "Mesas" (`/mesas`) y "Barra" (`/barra`) en el sidebar.
- Encendido: la sección de mesas aparece en Ajustes y el sidebar muestra "Mesas" y "Barra" tanto a `admin` como a `cashier`. Además el POS oculta los carritos guardados en `localStorage` (chip móvil, chip de escritorio y "Dejar en espera": `showParked = !restaurant_mode`, `frontend/app/pos/page.tsx:622,1030,1141,1155`); apagado, el POS no cambia.
- `business_config.lan_access_enabled` (0/1, default 0) se crea junto a esta migración pero pertenece a US-054 (acceso LAN); ver `servidor.md`.
- `PUT /api/setup` reemplaza la fila completa. El formulario de Ajustes debe reenviar siempre `restaurant_mode` y `lan_access_enabled` (leídos de `GET /setup`) para que guardar otro campo no los apague.

## Mesas

Tabla `dining_tables`: `name` único, `zone` (texto libre para agrupar), `seats` (sin uso en pantallas todavía), `sort_order`, `active`.

- Solo `admin` crea, edita o desactiva mesas (`POST/PUT/DELETE /api/tables`); cualquier usuario autenticado las lee (`GET /api/tables`, solo las activas, ordenadas por `sort_order` y luego `name`).
- Nombre vacío o solo espacios: 400.
- Nombre duplicado: 409 "Ya existe una mesa con ese nombre (puede estar desactivada)". La unicidad es exacta (distingue mayúsculas: "Mesa 1" y "mesa 1" coexisten) y **incluye mesas desactivadas**: no se puede recrear "Mesa 1" si existe una desactivada. Una mesa desactivada no aparece en la UI y hoy no se puede reactivar.
- Borrado = desactivación lógica (`active = 0`), nunca `DELETE`, porque los pedidos históricos la referencian.
  - Con un pedido `status='open'` en esa mesa: 409 "La mesa tiene un pedido abierto" y la mesa sigue activa.
  - Pedidos `closed` o `cancelled` no bloquean.
  - La comprobación y la desactivación son atómicas: un solo `UPDATE ... WHERE NOT EXISTS (pedido open)` (`internal/repository/dining_table.go:139-144`), de modo que un pedido abierto en paralelo no deja una mesa inactiva con pedido abierto (R-59 resuelto). `SQLITE_BUSY` → 409 "reintenta" (`internal/handler/order.go:14`, `tables.go`).
- Abrir un pedido exige mesa activa (ver Pedidos).

## Estado de una mesa (derivado, nunca almacenado)

`GET /api/tables` devuelve `state`, `open_order_id`, `open_total` y `opened_minutes`.

| Estado | Condición |
|---|---|
| `libre` | No hay pedido `open` con ese `table_id`. `open_order_id` null, `open_total` y `opened_minutes` en 0/null. |
| `ocupada` | Hay pedido `open` y al menos un ítem no anulado en `pending` o `preparing`. Un pedido abierto **sin ítems**, o con **todos sus ítems `void`**, también es `ocupada`. |
| `por_cobrar` | Hay pedido `open`, al menos un ítem no anulado, y todos los no anulados están en `ready` o `delivered`. |

- `open_total` = suma de `quantity * unit_price` de los ítems con `status != 'void'`. Ejemplo: café $4.500 x2 + torta $8.000 x1, ambos `ready` → `por_cobrar`, `$17.000`; con la torta anulada → `$9.000`.
- `opened_minutes` = minutos enteros desde `orders.created_at` (UTC contra UTC).
- Una lista sin mesas se serializa `[]`, no `null`.

## Pedidos

Un pedido es la cuenta abierta de una mesa. Todos los usuarios autenticados (`cashier` y `admin`) operan pedidos; sin token, 401. No escriben en `sales` (el cobro llega con US-057). Rutas en `internal/handler/handler.go:213-219`.

| Endpoint | Resultado |
|---|---|
| `POST /api/orders` `{table_id}` | 201 `{id,type,table_id,status,created_at}`. 400 si falta/≤0 `table_id` o `type` distinto de `table`; 404 mesa inexistente o desactivada; 409 si la mesa ya tiene pedido `open` o `SQLITE_BUSY`. |
| `GET /api/orders?status=open` | `[{id,table_id,table_name,total,items_pending,opened_minutes}]`, `[]` si no hay. `status` válido: `open|closed|cancelled`, otro → 400. `items_pending` = ítems no anulados en `pending` o `preparing`. |
| `GET /api/orders/{id}` | `{id,type,table_id,status,total,items[]}`; `items` incluye los `void` y nunca es `null`. 404 si no existe. |
| `POST /api/orders/{id}/items` `{product_id,quantity,note?}` | 201 con el ítem `{id,product_id,product_name,quantity,unit_price,subtotal,note,status}`. 404/400 producto inexistente o inactivo; 400 cantidad inválida; 409 pedido no `open`. |
| `PATCH /api/orders/{id}/items/{itemId}` `{quantity?,note?,status?}` | Actualiza solo lo enviado. 400 cantidad o `status` inválido; 404 ítem ajeno al pedido; 409 ítem `void` o pedido no `open`. |
| `DELETE /api/orders/{id}/items/{itemId}` | Anula (`status='void'`), 200 con el ítem. 409 si ya estaba `void` (sin segunda auditoría) o pedido no `open`. |
| `POST /api/orders/{id}/cancel` `{reason}` | 200; 400 motivo vacío o solo espacios; 404 no existe; 409 si no está `open`. |

Reglas:

- Ciclo: `open → cancelled` (`closed` llega con US-057). `type` es siempre `table` en v1.
- **Un solo pedido `open` por mesa**, garantizado por un `INSERT ... SELECT ... WHERE EXISTS (mesa activa) AND NOT EXISTS (pedido open)` en una sola sentencia (`internal/repository/order.go:131-136`), no por índice único (no hay migración 19). Quien pierde la carrera recibe 409 (`ErrTableHasOpenOrder`) o, si SQLite responde `SQLITE_BUSY` (código 5), `ErrBusy` → 409 "El sistema está ocupado, reintenta en un momento" (`internal/handler/order.go:14,22-23`), nunca 500.
- `unit_price` es un **snapshot** de `products.sale_price` al agregar el ítem; cambiar el precio después no altera pedidos existentes.
- `quantity` debe ser **entero > 0** (`0.5`, `2.0`, `"2"`, 0 y negativos → 400). Se guarda en la columna REAL como entero.
- `total` del pedido = Σ `quantity * unit_price` de los ítems con `status != 'void'`; subtotal de línea = `quantity * unit_price`.
- `PATCH` acepta `status` ∈ `pending|preparing|ready|delivered`, sin regla de orden entre transiciones. `void` u otro valor → 400: se anula solo por `DELETE`.
- Se puede editar `quantity` y `note` de un ítem `ready` o `delivered`; eso no cambia su `status`. Solo un ítem `void` es inmutable.
- `order_items.ready_at` se fija al pasar a `ready` desde otro estado; reenviar `ready` no lo sobrescribe; al salir de `ready` vuelve a `NULL` (`internal/repository/order.go:235-237`).
- Agregar, editar o anular ítems solo con el pedido `open`; si no, 409 sin cambios.
- Todo cambio de ítem (agregar, editar, anular) toca `orders.updated_at` en la misma transacción.
- **Anulación de ítem:** `audit_log` `action='order_item_void'`, `entity='order_item'`, `entity_id=<itemId>`, `user_id` de quien anula. `details` es JSON con exactamente las claves `order_id, product_id, product_name, quantity, unit_price, note, status, table_id, user_id`, donde `status` es el estado del ítem justo antes de anularlo. Sin `DELETE` físico.
- **Cancelación:** solo si `open`; `reason` obligatorio (tras `TrimSpace`). Pasa a `cancelled`, fija `closed_at` y `updated_at`, e inserta `audit_log` `action='order_cancel'`, `entity='order'`, `entity_id=<orderId>`, `details=<motivo>`, todo en una transacción. Los ítems se conservan. La mesa pasa a `libre` sin código adicional (estado derivado).
- Sin descuento de stock al agregar ítem (lo hará la venta, US-057). Sin actualización automática de `/mesas`: se refresca al entrar, al navegar y tras cada acción propia (roadmap R-63).

### Pantallas

- `/mesas`: grid de mesas activas con color por estado; ocupadas con total y minutos abiertos; tocar una libre abre pedido y navega a `/mesas/<id>`; tocar una ocupada o por cobrar navega al `open_order_id` sin crear otro; si el `POST` da 409 se refresca y se navega al pedido existente. Sin mesas, enlace a Ajustes → Mesas.
- `/mesas/[orderId]`: catálogo con buscador (tocar agrega el ítem al servidor de inmediato, sin "enviar a cocina"), líneas con cantidad, nota y estado (pendiente/preparando/listo/entregado), anulación con confirmación (línea tachada), cancelación con motivo. Un fallo al agregar muestra mensaje y no deja línea fantasma. En 320px catálogo y pedido se alternan por pestañas. No hay botones para avanzar el estado de un ítem (llega con la barra, US-058).
- Ruta dinámica exportada estáticamente con `generateStaticParams [{orderId:'_'}]`; ver `servidor.md`.

## Esquema (migración 18)

`dining_tables`, `orders` (`type` `table|takeaway`, `status` `open|closed|cancelled`) y `order_items` (`status` `pending|preparing|ready|delivered|void`) se crean de una vez; los endpoints de `orders`/`order_items` los introdujo US-056 (ver Pedidos). Las claves foráneas declaradas no se aplican en runtime (roadmap R-56), por lo que la integridad depende del borrado lógico y de las validaciones del servicio.
