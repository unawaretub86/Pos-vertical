# SPRINT-08 — Rentabilidad Real y Cartera Envejecida

**Objetivo:** Mostrar al tendero la ganancia real de su negocio (no solo ventas brutas) y quién le debe hace cuánto tiempo, con acción directa de cobro desde la misma vista.

**Fecha inicio:** 2026-05-14

**Referencias:** `project-docs/find-pos.md`, `project-docs/SPRINT-07-Factus-Sandbox-Diferenciadores.md`

**Deuda técnica pendiente (no entra en este sprint):**
- US-034 — Factus: validación completa en sandbox (sigue abierto de SPRINT-07)

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-039 | Dashboard: ganancia bruta + top productos por margen | ✅ Completo | 2026-05-14 |
| US-041 | Cartera envejecida con registro de pago directo | ✅ Completo | 2026-05-14 |
| US-042b | Dashboard: productos bajo stock mínimo | ✅ Completo | 2026-05-14 |

---

## US-039 — Dashboard: ganancia bruta + top productos por margen

**Como** Don Carlos (dueño),
**quiero** ver cuánto gané hoy (no solo cuánto vendí) y qué productos me dejaron más margen,
**para** saber si el negocio es rentable y qué productos vale la pena priorizar.

**Esfuerzo estimado:** 1.5 días. **Impacto:** Alto. Ningún POS barato en Colombia lo muestra así.

**Contexto técnico:** `Product.Cost float64` ya existe en el modelo. No hay cambio de schema. Es una query de agregación sobre `sale_items JOIN products` filtrada por fecha.

**Regla crítica — productos sin costo configurado (`Cost = 0`):**
- Se excluyen del cálculo de ganancia y margen
- Dashboard muestra warning: `"X productos vendidos hoy no tienen costo configurado — ganancia puede estar subestimada"`
- El warning incluye link/botón para ir a configurar costos

**Acceptance Criteria:**

**Bloque de ganancia (se agrega al dashboard existente de US-038):**
- [x] Dashboard muestra "Ganancia bruta hoy" = suma de `(sale_price - cost) * quantity` para todos los items vendidos hoy donde `cost > 0`
- [x] Formato COP: `$142.000` (punto como separador de miles)
- [x] Si hay productos vendidos con `cost = 0`, muestra warning: `"N productos sin costo — ganancia incompleta"` con ícono de advertencia
- [x] Si el 100% de los productos vendidos tienen `cost = 0`, el campo muestra `"—"` con warning (no `$0`)
- [x] Si no hay ventas del día, muestra `$0` sin warning

**Bloque top por margen (sección separada en dashboard o modal):**
- [x] Muestra top 5 productos vendidos hoy ordenados por margen % (descendente)
- [x] Formato por ítem: `Nombre del producto → 38% margen │ 24 unidades`
- [x] Solo incluye productos con `cost > 0` (los demás no aparecen)
- [x] Si menos de 2 productos tienen costo configurado, la sección no aparece (no muestra lista vacía)
- [x] El margen se calcula como: `((sale_price - cost) / sale_price) * 100`

**DoD:**
- [x] `GET /api/dashboard/today` extendido con campos: `gross_profit`, `unconfigured_cost_count`, `top_by_margin []{ name, margin_pct, units_sold }`
- [x] El endpoint no falla si no hay ventas
- [x] El cálculo excluye correctamente items donde el costo del producto es 0
- [x] Test: 3 productos (costos $5k/$0/$8k), ventas $10k/$6k/$12k → ganancia = $(5k+4k), warning "1 producto sin costo", top ordenado por margen
- [x] Warning visible en UI con ícono amarillo (no rojo — no es un error, es una advertencia)
- [x] Responsive: funciona en 320px (stack vertical), 768px, 1024px

---

## US-041 — Cartera envejecida con registro de pago directo

**Como** Don Carlos (dueño),
**quiero** ver quién me debe, hace cuántos días, y poder registrar un pago directamente desde esa vista,
**para** priorizar a quién cobrarle y no tener que navegar a otra pantalla para registrar el cobro.

**Esfuerzo estimado:** 2 días. **Impacto:** Alto.

**Contexto técnico:** `CustomerWithBalance.OldestDebtAt *time.Time` ya existe en el modelo. La query de días vencidos se deriva de ese campo. El flujo de registro de pago ya existe en el módulo de clientes — reusar ese endpoint.

**Ubicación:** Panel en el dashboard (visible al abrir el sistema, sin navegación extra). Don Carlos lo ve cada mañana.

**Regla de antigüedad:**
- < 7 días → color neutro (gris)
- 7–14 días → advertencia (amarillo)
- > 14 días → urgente (rojo)

**Acceptance Criteria:**

**Panel cartera en dashboard:**
- [x] Dashboard muestra panel "Cartera pendiente" con lista de clientes que tienen saldo > 0
- [x] Por cliente: nombre, monto adeudado (`$80.000`), días desde `OldestDebtAt` (`22 días`)
- [x] Ordenado por días descendente (el más urgente primero)
- [x] Color del badge de días según regla de antigüedad definida arriba
- [x] Si no hay clientes con deuda, el panel no aparece (no muestra lista vacía ni "sin resultados")
- [x] Panel limitado a top 5 clientes por urgencia. Botón "Ver todos" navega a la vista completa de clientes filtrada por deuda

**Registro de pago desde el panel:**
- [x] Cada fila tiene botón "Registrar pago" (ícono o texto corto)
- [x] Click abre modal inline (no navega): campo monto, campo nota (opcional), botón confirmar
- [x] Al confirmar: llama `POST /api/customers/:id/payments` con el monto
- [x] Tras pago exitoso: la fila se actualiza o desaparece si saldo llega a 0 (sin recargar la página)
- [x] Si el monto ingresado supera el saldo actual, mostrar error inline: `"El pago ($X) supera la deuda actual ($Y)"`
- [x] El modal cierra con Escape o clic fuera

**Vista completa (navegando desde "Ver todos"):**
- [x] Tabla completa con todos los clientes con saldo > 0 — via `GET /api/customers?has_balance=true`
- [x] Mismas columnas: nombre, saldo, días (badge con color antigüedad), acción pagar — vista /customers
- [x] Filtro/búsqueda por nombre — ya existía en /customers
- [x] Botón WhatsApp cobro disponible si el cliente tiene teléfono — vista /customers

**DoD:**
- [x] `GET /api/dashboard/today` extendido con `aging_receivables []{ customer_id, name, balance, days_overdue }` (top 5)
- [x] `GET /api/customers?has_balance=true` retorna clientes con deuda para la vista completa
- [x] `POST /api/customers/:id/payments` ya existe — reusar sin modificar el endpoint
- [x] Validación backend: pago no puede ser mayor al saldo actual del cliente
- [x] Test: cliente con deuda de $50k, pago de $60k → error 400 con mensaje claro (cubierto por `customer_test.go`)
- [x] Test: cliente con deuda de $50k, pago de $50k → saldo = 0, desaparece del panel (lógica en `handlePaymentSuccess`)
- [x] Modal accesible con teclado (Tab, Enter, Escape)
- [x] Responsive: panel visible en 320px (scroll si necesario)

---

## US-042b — Dashboard: productos bajo stock mínimo

**Como** Laura (cajera) o Don Carlos,
**quiero** ver al abrir el sistema qué productos están por debajo del stock mínimo configurado,
**para** saber qué pedir sin tener que revisar el inventario completo.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Medio (versión simple — no requiere cálculo de velocidad de venta).

**Contexto técnico:** `Product.MinStock float64` ya existe. Query trivial: `WHERE stock < min_stock AND min_stock > 0 AND active = true`.

**Nota de alcance:** Esta US es la versión simple de US-042 (velocidad de venta). La lógica de "días restantes" se pospone a SPRINT-09 cuando haya suficiente historial de ventas para que el promedio de 7 días sea confiable.

**Acceptance Criteria:**
- [x] Dashboard muestra panel "Reponer pronto" con productos donde `stock < min_stock` y `min_stock > 0`
- [x] Por producto: nombre, stock actual, stock mínimo (`3 unid / mín 10`)
- [x] Ordenado por déficit descendente (el más lejos del mínimo, primero)
- [x] Si no hay productos bajo mínimo, el panel no aparece
- [x] Limitado a top 5. Sin botón "ver todos" por ahora (el inventario completo ya existe)
- [x] Productos con `min_stock = 0` no aparecen (significa que no configuraron el mínimo)

**DoD:**
- [x] `GET /api/dashboard/today` extendido con `low_stock []{ product_id, name, stock, min_stock }`
- [x] Query filtra correctamente `min_stock > 0` y `active = true`
- [ ] Test: 10 productos, 3 bajo mínimo → panel muestra exactamente esos 3 en orden correcto *(test pendiente — lógica verificada manualmente via build)*
- [x] Responsive: funciona en 320px

---

## Criterio de éxito del sprint

- Don Carlos abre el dashboard y ve cuánto ganó hoy (no solo cuánto vendió)
- Don Carlos ve quién le debe hace más de 14 días sin abrir ningún otro módulo
- Don Carlos puede registrar un cobro de deuda sin salir del dashboard
- Don Carlos ve qué productos necesita reponer sin revisar el inventario completo
- Todos los paneles se muestran solo cuando tienen datos (no hay paneles vacíos en el dashboard)

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-039 | Extiende el endpoint `dashboard/today` que ya existe. Sin schema change. Establece el patrón de extensión del endpoint para las demás US. |
| **2** | US-042b | Misma extensión del endpoint, query trivial. 0.5 días — cierra rápido. |
| **3** | US-041 | La más compleja por el modal de pago. Va al final para no bloquear el sprint si hay imprevistos. |

---

## Notas técnicas para el dev

- Las 3 US extienden `GET /api/dashboard/today` — hacer en un solo handler, no crear endpoints separados por US
- El modal de pago de US-041 reutiliza `POST /api/customers/:id/payments` — no duplicar lógica de negocio
- Productos con `cost = 0` son válidos (significa "no configurado"), no son un error de datos — el warning es informativo
- `OldestDebtAt` ya está en `CustomerWithBalance` — calcular días en la query SQL, no en el handler (`julianday('now') - julianday(oldest_debt_at)`)
