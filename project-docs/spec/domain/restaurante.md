# Dominio — Restaurante (modo restaurante y mesas)

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** modo restaurante, mesas y estado de mesa
> **Actualizado:** 2026-09-29 · Verificado contra el código en esa fecha (US-055: `internal/repository/dining_table.go`, `internal/handler/tables.go`, `frontend/lib/nav.ts`).

Módulo para cafeterías y restaurantes: mesas, pedidos y pantalla de barra (SPRINT-13). Hoy solo existe la gestión de mesas y el interruptor; pedidos, cobro y barra llegan en US-056, US-057 y US-058 y agregarán sus reglas aquí.

---

## Modo restaurante

- `business_config.restaurant_mode` (0/1, default 0) decide si el negocio usa el módulo. Se activa en Ajustes con el toggle "Modo restaurante" (solo admin, vía `PUT /api/setup`).
- Apagado: no se muestra la sección de mesas en Ajustes ni "Mesas" (`/mesas`) y "Barra" (`/barra`) en el sidebar.
- Encendido: la sección de mesas aparece en Ajustes y el sidebar muestra "Mesas" y "Barra" tanto a `admin` como a `cashier`.
- `business_config.lan_access_enabled` (0/1, default 0) se crea junto a esta migración pero pertenece a US-054 (acceso LAN); su regla de negocio se documenta al cerrar esa US.
- `PUT /api/setup` reemplaza la fila completa. El formulario de Ajustes debe reenviar siempre `restaurant_mode` y `lan_access_enabled` (leídos de `GET /setup`) para que guardar otro campo no los apague.

## Mesas

Tabla `dining_tables`: `name` único, `zone` (texto libre para agrupar), `seats` (sin uso en pantallas todavía), `sort_order`, `active`.

- Solo `admin` crea, edita o desactiva mesas (`POST/PUT/DELETE /api/tables`); cualquier usuario autenticado las lee (`GET /api/tables`, solo las activas, ordenadas por `sort_order` y luego `name`).
- Nombre vacío o solo espacios: 400.
- Nombre duplicado: 409 "Ya existe una mesa con ese nombre (puede estar desactivada)". La unicidad es exacta (distingue mayúsculas: "Mesa 1" y "mesa 1" coexisten) y **incluye mesas desactivadas**: no se puede recrear "Mesa 1" si existe una desactivada. Una mesa desactivada no aparece en la UI y hoy no se puede reactivar.
- Borrado = desactivación lógica (`active = 0`), nunca `DELETE`, porque los pedidos históricos la referencian.
  - Con un pedido `status='open'` en esa mesa: 409 "La mesa tiene un pedido abierto" y la mesa sigue activa.
  - Pedidos `closed` o `cancelled` no bloquean.

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

## Esquema (migración 18)

`dining_tables`, `orders` (`type` `table|takeaway`, `status` `open|closed|cancelled`) y `order_items` (`status` `pending|preparing|ready|delivered|void`) se crean de una vez; `orders` y `order_items` no tienen aún endpoints (US-056 en adelante). Las claves foráneas declaradas no se aplican en runtime (roadmap R-56), por lo que la integridad depende del borrado lógico y de las validaciones del servicio.
