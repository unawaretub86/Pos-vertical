# Diseño — Módulo de Mesas, Pedidos y Pantalla de Barra (KDS)

> **Tipo:** diseño aprobado · **Vigente:** sí · **Fecha:** 2026-09-20
> Insumo de SPRINT-13. Manda sobre el diseño del módulo de restaurante.


> Fecha: 2026-09-20 · Estado: aprobado por Andrey · Origen: cliente potencial (cafetería)
> Implementación: SPRINT-13 (a generar desde este documento)

---

## 1. Contexto y problema

El POS actual modela **venta inmediata**: carrito → pago → ticket. No existe el concepto de pedido abierto ni de preparación.

El cliente potencial es una **cafetería**. Su operación real:

- El cliente pide en barra. A veces paga de una (para llevar), a veces se sienta y paga al irse.
- El cajero le pasa el pedido al barista **en post-its**.
- El barista prepara y no hay registro de qué está pendiente, qué lleva cuánto tiempo, ni qué se anuló.

El dolor central no es la mesa: es que **la información del pedido viaja en papel**. Por eso el corazón del módulo es la pantalla de barra (KDS), y las mesas son el soporte para la cuenta abierta.

### Lo que ya existe y se reusa

| Pieza | Ubicación | Uso en este módulo |
|-------|-----------|--------------------|
| Creación de venta (TX: venta + ítems + stock + movimiento de caja) | `internal/repository/sale.go` | El cobro del pedido la llama tal cual |
| Pagos mixtos | `sale_payments`, `frontend/lib/saleCalc.ts` | Cobro de mesa sin cambios |
| Caja / arqueo | `internal/repository/register.go` | Sin cambios: la venta nace al cobrar |
| Facturación electrónica | `internal/factelect/` | Sin cambios: opera sobre la venta ya creada |
| Auditoría | `audit_log` (migración 12) | Anulación de ítem y cancelación de pedido |
| Ticket térmico | `components/TicketPreview.tsx` | Sin cambios |

### Conflicto conocido: "carritos guardados"

`frontend/app/pos/page.tsx` implementa carritos aparcados en `localStorage` (`PARKED_CARTS_KEY`). Es una cuenta abierta casera, visible solo en ese navegador — el barista nunca la ve.

**Decisión:** con `restaurant_mode` activo, el botón de carritos guardados se oculta. Dos mecanismos de cuenta abierta en paralelo confunden al cajero y dividen la verdad entre `localStorage` y la base.

---

## 2. Alcance

### Entra en v1

- Mesas numeradas en lista/grid con estado (sin plano visual).
- Pedido por mesa: agregar ítems, nota libre por línea, anular línea, cancelar pedido.
- Pedido para llevar con número de turno diario.
- Cobro de pedido reusando el flujo de venta existente (efectivo, mixto, crédito).
- Pantalla de barra (KDS) con estados por ítem, minutos transcurridos y alerta sonora.
- Toggle `restaurant_mode`: el módulo no existe para clientes que no son restaurante.

### Fuera de v1 (explícito)

Propina · división de cuenta · transferir ítems entre mesas · plano visual arrastrable · modificadores con precio (leche de almendras +$2.000) · rol "mesero" · impresión de comanda en térmica.

Los modificadores con precio son el candidato número uno para v2: es lo que convierte esto en un producto vendible a cualquier café, no solo a este.

---

## 3. Modelo de datos — migración 15

La migración 14 ya está ocupada (correo del dueño, SPRINT-12). Este módulo es la **15**.

```sql
CREATE TABLE IF NOT EXISTS dining_tables (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	zone TEXT NOT NULL DEFAULT '',
	seats INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0,
	active INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL CHECK (type IN ('table','takeaway')),
	table_id INTEGER REFERENCES dining_tables(id),
	turn_number INTEGER,
	customer_name TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed','cancelled')),
	user_id INTEGER NOT NULL REFERENCES users(id),
	sale_id INTEGER REFERENCES sales(id),
	note TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	closed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_table ON orders(table_id);
CREATE INDEX IF NOT EXISTS idx_orders_updated ON orders(updated_at);

CREATE TABLE IF NOT EXISTS order_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	order_id INTEGER NOT NULL REFERENCES orders(id),
	product_id INTEGER NOT NULL REFERENCES products(id),
	quantity REAL NOT NULL,
	unit_price REAL NOT NULL,
	note TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending'
		CHECK (status IN ('pending','preparing','ready','delivered','void')),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	ready_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_status ON order_items(status);

ALTER TABLE business_config ADD COLUMN restaurant_mode INTEGER NOT NULL DEFAULT 0;
```

### Decisiones de diseño

**La mesa no guarda estado.** Mesa ocupada = existe un `orders` con `status='open'` y ese `table_id`. Un estado guardado se desincroniza con el primer crash o la primera condición de carrera; uno derivado no puede mentir.

**El estado vive en el ítem, no en el pedido.** El barista marca listo el capuchino sin tocar la torta que sigue en el horno. El estado del pedido se deriva de sus ítems. `orders.status` solo distingue abierto / cobrado / cancelado.

**`unit_price` es un snapshot.** Si el precio del producto cambia mientras la mesa está abierta, la cuenta respeta el precio del momento en que se pidió. Al cobrar, ese precio pasa directo a `sale_items`.

**El stock se descuenta al cobrar**, dentro de la transacción de venta que ya existe. Consecuencia aceptada: un pedido abierto no reserva stock. En una cafetería, donde los insumos no se descuentan por unidad vendida, es irrelevante; en un negocio con stock unitario escaso habría que revisarlo.

**`turn_number` se asigna dentro de la transacción de creación**: `SELECT COALESCE(MAX(turn_number),0)+1 FROM orders WHERE date(created_at)=date('now','localtime')`. Reinicia cada día. Fuera de transacción, dos pedidos simultáneos toman el mismo turno.

**`updated_at` se toca en todo cambio de pedido o de ítem.** Es lo que hace barato el polling del KDS.

---

## 4. API

```
GET    /api/tables                      lista + estado derivado + total del pedido abierto
POST   /api/tables                      admin
PUT    /api/tables/{id}                 admin
DELETE /api/tables/{id}                 admin · rechaza si hay pedido abierto

POST   /api/orders                      {type, table_id?, customer_name?}
GET    /api/orders?status=open
GET    /api/orders/{id}                 pedido + ítems + totales
POST   /api/orders/{id}/items           {product_id, quantity, note}
PATCH  /api/orders/{id}/items/{itemId}  quantity | note | status
DELETE /api/orders/{id}/items/{itemId}  → audit_log
POST   /api/orders/{id}/cancel          {reason} → audit_log
POST   /api/orders/{id}/checkout        mismo body que POST /api/sales

GET    /api/kds/orders                  ítems pending|preparing|ready + server_time
```

**El checkout no reimplementa nada de cobro.** Arma el payload de venta desde los ítems del pedido y llama el camino de `createSale` ya existente: pagos mixtos, crédito con límite, movimiento de caja, documento electrónico y ticket salen gratis y ya probados. Al terminar, el pedido queda `closed` con su `sale_id`. Todo en una transacción: si la venta falla, el pedido sigue abierto.

**El KDS filtra por estado de ítem, no de pedido.** Es el detalle que se escapa: el pedido para llevar se cobra *antes* de prepararse, así que su `orders.status` ya es `closed` mientras el café sigue pendiente. Filtrar por pedido abierto haría desaparecer de la barra todo lo prepagado.

Permisos: cajero y admin operan pedidos y KDS. La gestión de mesas (crear, editar, borrar) es admin. No se agrega rol nuevo.

---

## 5. Interfaz

### `/mesas`
Grid de tarjetas. Cada una: nombre, zona, estado por color (libre / ocupada con total y minutos / todo listo, por cobrar). Botón fijo **Para llevar** que abre pedido sin mesa. La administración de mesas vive en Ajustes, no aquí.

### `/mesas/[orderId]`
Catálogo a la izquierda (mismo buscador y grid de productos del POS), líneas del pedido a la derecha: cantidad, nota por línea, estado del ítem. Acciones: agregar, anular línea, cancelar pedido, **Cobrar**.

Los ítems se mandan a barra **al guardarse**. No hay botón "enviar a cocina": es un paso que la gente olvida y deja al barista esperando.

### `/barra` (KDS)
Tres columnas: Pendiente → Preparando → Listo. Tarjetas grandes y tipografía gruesa, legibles y tocables a un metro. Cada tarjeta muestra mesa o número de turno, los ítems con su nota destacada (la nota es la información que hoy va en el post-it), y los minutos transcurridos, que cambian de color al pasar el umbral. Un toque avanza el estado.

Sonido al entrar ítem nuevo: se detectan ids nuevos entre polls y se dispara un `<audio>` precargado. El navegador exige interacción previa del usuario, así que la pantalla arranca con un botón **Activar sonido**.

Refresco: polling cada 3 s contra `GET /api/kds/orders`. Sobrevive a que la tablet pierda WiFi y vuelva, sin reconexión ni estado en el servidor. SSE se puede montar después sin cambiar la API.

### Refactorización incluida
El panel de pago vive dentro de `frontend/app/pos/page.tsx` (1624 líneas), con todo el estado de pagos mixtos. Cobrar una mesa necesita exactamente ese panel. Sale a `components/SalePaymentPanel.tsx` y lo consumen POS y mesas. Sin esto, el cobro se duplica y las dos copias divergen en el primer bug.

Es la única refactorización del sprint.

### Navegación
"Mesas" y "Barra" aparecen en el sidebar solo con `restaurant_mode` activo.

---

## 6. Flujos

**Mesa:** abrir mesa → agregar ítems (aparecen en barra al instante) → barista marca preparando y luego listo → cobrar → venta + ticket → mesa libre.

**Para llevar prepago:** nuevo pedido takeaway → ítems → cobrar → venta cerrada y pedido `closed`, con los ítems todavía pendientes en barra bajo su número de turno → barista marca listo → se entrega.

**Caja:** sin cambios. La venta nace al cobrar, así que un pedido abierto no es dinero y no toca el arqueo.

**Cierre de caja con pedidos abiertos:** avisa ("hay 3 pedidos sin cobrar") pero **no bloquea**. Bloquear deja al cajero atrapado a medianoche por una mesa que se fue sin pagar.

**Cancelaciones:** anular un ítem ya preparado o cancelar un pedido con ítems queda en `audit_log` con usuario y motivo. Es el hueco clásico de fuga de caja en restaurantes.

---

## 7. Pruebas

Siguiendo el patrón existente (`internal/repository/*_test.go`, `internal/handler/*_test.go`):

- `turn_number` consecutivo, reinicio diario, y sin colisión entre dos pedidos creados a la vez
- checkout atómico: si la venta falla, el pedido no queda cerrado ni a medias
- checkout de pedido vacío → rechazado
- borrar mesa con pedido abierto → rechazado
- totales del pedido idénticos a los de la venta generada (reusando `lib/saleCalc`)
- estado del pedido derivado correctamente cuando hay ítems anulados

Frontend: pruebas de cálculo en `lib/` como ya se hace; el resto es verificación manual con el checklist de QA.

---

## 8. Riesgos

1. **El KDS necesita acceso LAN para correr en la tablet del barista.** Eso es US-054 (SPRINT-12), hoy pendiente. Sin ella, el KDS solo abre en la máquina del POS y el módulo pierde su razón de ser. Es dependencia dura: va antes o dentro de este sprint.
2. **Sin modificadores con precio**, un café grande y uno pequeño cuestan igual salvo que se creen como productos separados. Hay que decírselo al cliente en la demo, no dejar que lo descubra.
3. El polling de 3 s con 20 pedidos abiertos es trivial para SQLite local. Sin riesgo de carga.
4. Retirar los carritos guardados en modo restaurante cambia un flujo que el cajero ya conoce, si el cliente venía usando el POS. Para un cliente nuevo, no aplica.

---

## 9. Orden de implementación sugerido

1. Migración 15 + toggle `restaurant_mode` (base para todo, cero riesgo sobre lo existente)
2. CRUD de mesas + pantalla de administración
3. Pedidos: crear, agregar ítems, nota, anular, cancelar (backend + `/mesas`)
4. Extracción de `SalePaymentPanel` + checkout de pedido
5. KDS: endpoint, pantalla, polling, tiempos y sonido
6. Para llevar con turno
7. US-054 (acceso LAN) si no se hizo antes — sin esto no hay demo real
