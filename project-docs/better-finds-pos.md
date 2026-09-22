# POS Vertical — Investigación de Mercado y Features Diferenciadoras

> **Tipo:** research · **Vigente:** NO (pero es el más reciente) · **Fecha:** 2026-09-08
> Investigación de mercado, bloques A–K. **Sus afirmaciones de "esto no existe" se verifican con `grep` antes de usarse** — varias ya quedaron desactualizadas. Lo que viene está en [roadmap.md](roadmap.md), no aquí.


> Generado: 2026-09-08

## Estado general
Investigación de POS comerciales top (mundial + Colombia) para identificar features que ayuden a vender sobre competencia. 3 quick-wins de bajo esfuerzo, 1 apuesta grande fuera de alcance (fintech), 2 features complejas 100% ejecutables en solitario.

---

## A — Qué hacen los POS grandes (mundial) y qué aplica

| POS | Feature fuerte | Aplica a Pos-vertical? |
|---|---|---|
| **Square** | Free tier, setup en minutos, omnichannel | Onboarding ya existe — pulir a "5 min y listo" |
| **Toast** | UX guiada por rol, payroll integrado | Roles ya existen — falta wizard guía por tarea |
| **Clover** | Bring-your-own-processor, hardware flexible | No aplica (offline/local, no processor) |
| **Lightspeed** | Inventario profundo: variantes, proveedores, órdenes de compra | Gap real si el target son distribuidoras |
| **SumUp** | Simplicidad brutal | Ya es la ventaja actual — no diluir con features |

Patrón 2026: todos van hacia pagos integrados + inventario profundo + reportes accionables. **Ninguno hace fiado bien** — ahí seguimos ganando.

## B — Qué hacen los locales (Colombia) que este no

- **Alegra**: todo-en-uno (POS + facturación + contabilidad + nómina), acepta Wompi/MercadoPago/PayU/Nequi/QR bancario.
- **Siigo**: 30 años, marca de confianza, modular, fuerte en cumplimiento DIAN.
- **POS Colombia (genérico)**: recibos por WhatsApp, pagos Nequi/transferencia, térmica 58mm integrada.

**Gap que más duele:** todos los locales aceptan pagos digitales colombianos nativos (Nequi, Wompi, QR). Pos-vertical es 100% offline — bien para caja física, pero el tendero moderno ya cobra por Nequi todo el día.

## C — Quick wins priorizados (orden de venta, no de complejidad)

1. **Método de pago "Nequi/transferencia" en el ticket** — solo un campo, sin integración API real. Bajo esfuerzo, cierra brecha #1 vs todos los locales.
2. **Recordatorio de deuda por WhatsApp** — link `wa.me` con mensaje prellenado del saldo. Cero API, cero costo. Nadie lo hace bien hoy.
3. **Impresión térmica real ESC/POS** — ya en el mapa de gaps conocidos (`better-pos.md`). Bloqueante para reemplazar cuadernillo/otro POS en tienda física.
4. **Precio mayorista con regla clara** — el campo `wholesale_price` existe, falta lógica de cuándo aplicarlo (por cliente tipo, no manual).
5. **App móvil / vista web del dueño** — ver ventas del día desde el celular. Lo más pedido a los 30 días de uso.

---

## D — Apuesta grande: capital embebido (descartada por ahora)

Los 3 líderes mundiales (Square Loans, Toast Capital, Clover Capital) ya venden **capital de trabajo basado en data de ventas del POS**, con cobro automático como % de venta diaria. En Colombia el concepto está "en fase de consolidación" — nadie lo hace a nivel tienda de barrio todavía.

**Por qué se descarta por ahora:** requiere alianza con fintech regulada (SFC) para poder prestar plata — no es algo que se pueda construir y vender en solitario. Riesgo de "pegarse un tiro en el pie" si se intenta sin socio.

**Qué sí queda del concepto, ejecutable en solitario:**
- **Scoring interno del cliente fiado** — con el historial de abonos que ya se guarda, calcular automático: score de riesgo por cliente, límite de crédito sugerido, alerta temprana de mora. Ningún competidor local guarda ni analiza esto (lo tratan como campo contable, no como relación). Complejidad media, cero dependencia externa.

---

## E — Features complejas 100% ejecutables en solitario (sin socios, sin partners)

### E1 — Sync multi-caja LAN offline
Ferretería/distribuidora que crece pasa de 1 caja a varias. Ningún POS barato colombiano sincroniza múltiples cajas sin internet/nube (Alegra/Siigo necesitan su servidor cloud).

- Instancia "maestro" en LAN, otras cajas se conectan por WiFi local (mismo router de la tienda, sin internet real)
- Sync de inventario/ventas en tiempo real vía TCP/websocket local
- Parte dura: resolución de conflictos (dos cajas venden el último producto al mismo tiempo) — problema de ingeniería pura, no de terceros
- Venta: "2, 3, 5 cajas — mismo inventario, cero nube, cero mensualidad extra"

### E2 — Asistente IA local, 100% offline
"Primer POS colombiano con IA que corre sin internet" — literal, no marketing vacío. Consulta en lenguaje natural sobre la propia SQLite ("¿cuánto vendí ayer?", "¿quién me debe más de 50 mil?", "¿qué se me está acabando?").

**Chequeo de viabilidad de hardware (Colombia clase media-baja):**
- RAM típica: 4GB común, 8GB ya es "bueno". Muchos con disco mecánico.
- CPU sin GPU, a veces sin AVX2 → runtimes de IA optimizados no corren rápido o no corren.
- El binario ya usa RAM (Go + SQLite + servidor + Next.js embebido) — poco margen.

**Veredicto:** un LLM real (Qwen2.5-0.5B / Llama-3.2-1B cuantizados, ~400MB–1GB RAM) es viable solo en máquinas de 8GB+ con poco más abierto. En 4GB es apostar — riesgo de swap/colgada, y eso mata reputación peor que no tener la feature.

**Recomendación:** no apostar todo a un LLM.
1. **Motor de reglas/patrones** (regex + intención: "cuánto vendí", "quién me debe", "qué se acaba") — cero riesgo, corre instantáneo en cualquier PC, cubre ~80% de las preguntas reales de un tendero.
2. **Modelo chico opcional**, activado solo si detecta RAM disponible suficiente (≥6-8GB libres) al iniciar. Si no, se desactiva solo y usa el motor de reglas. El usuario nunca ve el modelo colgarse porque nunca se activa en máquina débil.

---

## Recomendación de orden

1. Quick wins de bloque C (bajo esfuerzo, cierran brecha inmediata vs locales).
2. Scoring interno del cliente fiado (profundiza el diferenciador ya existente, sin dependencias).
3. Elegir entre E1 (sync multi-caja) o E2 (asistente IA con motor de reglas + modelo opcional) según el target: negocios ya creciendo a multi-caja → E1 primero; gancho de marketing/venta inmediato → E2 primero (empezando por el motor de reglas, no el modelo pesado).
4. Capital embebido queda anotado como visión de largo plazo, condicionado a encontrar partner fintech regulado — no se persigue en solitario.

---

# Ronda 2 — Features estándar en otros POS que este no tiene (2026-09-08)

> Criterio: verificado por grep en `internal/` y `frontend/` que NO existe, y que NO está ya en `better-pos.md`, `find-pos.md`, `finds-betters.md` ni en SPRINT-07/08/09. Fuentes: Tiendatek, Treinta, Vendty, Siigo POS, Helisa POS, Scrampi (guías 2026 tienda de barrio).

## F — Operación diaria de caja (lo que un cajero espera de "cualquier POS")

| # | Feature | Qué hacen los demás | Estado aquí | Esfuerzo |
|---|---|---|---|---|
| F1 | **Pago mixto** | Una venta se paga $30K efectivo + $20K Nequi + resto fiado. Helisa, Siigo, Tiendatek lo tienen. Scrampi lo marca como requisito 2026 para tienda de barrio ("efectivo, Nequi y Daviplata, incluso combinados"). | `sales.payment_method` es una sola columna. Imposible hoy. | 2–3 días: tabla `sale_payments (sale_id, method, amount, reference)`, UI en modal de cobro, cierre de caja suma por método. |
| F2 | **Venta en espera / facturas en espera** | Cliente A deja el carrito para buscar plata, cliente B pasa. El cajero "parquea" la venta y la retoma. Estándar en Helisa/Vendty/Siigo. | No existe. Hoy el cajero pierde el carrito. | 1 día: guardar carrito en `localStorage` con nombre/hora, lista "En espera (2)" en el POS. Sin backend. |
| F3 | **Devolución parcial con reingreso a stock** | Devolver 1 de 3 unidades, reintegrar stock, generar salida de caja o saldo a favor. | Solo existe `void` total (`DELETE /sales/{id}/void`). | 2 días: `POST /sales/{id}/returns` con items, movimiento `cash_out` tipo `refund`, stock +N. Para FE ya existe nota crédito. |
| F4 | **Gastos del negocio** | "Pagué $80K de luz, $40K al domiciliario". Vendty y Tiendatek: gastos categorizados → utilidad neta real. | Solo `cash_out` genérico en caja, sin categoría ni reporte. | 1–2 días: `expenses (category, amount, note, register_id)`, categorías fijas (servicios, arriendo, nómina, proveedor, otro), dashboard "Ganancia neta = bruta − gastos". |
| F5 | **Arqueo con conteo de denominaciones** | Al cerrar caja: 3 × $50K, 12 × $20K, 8 × $10K… el sistema suma. Siigo lo documenta como flujo estándar. | Cierre pide un solo número "monto real". | 0.5 día: grid de denominaciones COP en modal de cierre, suma automática → `actual_close`. |
| F6 | **Cierre de turno imprimible (reporte X/Z)** | Tirilla al cerrar caja: ventas por método, #transacciones, entradas/salidas, diferencia, firmado por cajero. | El cierre existe, pero no hay ticket de cierre. | 1 día: reutilizar `TicketPreview` con plantilla de cierre. Cuando llegue ESC/POS sale gratis. |
| F7 | **PIN rápido por cajero / cambio de usuario sin logout** | Turnos: cajero se va, entra otro, 4 dígitos y listo. Estándar en POS retail. | Login completo con usuario+contraseña. | 1 día: campo `pin` en `users`, pantalla de bloqueo con teclado numérico, reutiliza JWT. |
| F8 | **Atajos de teclado en POS** | F1 buscar, F2 cantidad, F4 cliente, F9 cobrar, Esc cancelar. Cajeros de ferretería no sueltan el teclado. | Solo Enter en lector de código de barras. `finds-betters.md` §4 lo menciona como "modo vendedor rápido" pero no hay atajos definidos. | 1 día: `keydown` global en `/pos`, leyenda de atajos en pantalla. |

## G — Inventario que los competidores dan por sentado

| # | Feature | Qué hacen los demás | Estado aquí | Esfuerzo |
|---|---|---|---|---|
| G1 | **Proveedores + compras (órdenes de compra)** | Registrar factura de compra: proveedor, items, costo unitario → actualiza stock Y costo. Lightspeed/Siigo/Vendty. US-044 (sugerencia de pedido) no tiene a quién pedirle. | `stock_entries` tiene solo `quantity, note`. No hay tabla `suppliers` ni costo por entrada. | 3 días: `suppliers`, `purchases`, `purchase_items (cost)`. Actualiza `products.cost` con último costo o promedio ponderado. Prerequisito real de US-044. |
| G2 | **Kardex + ajustes de inventario + conteo físico** | Historial de movimientos por producto (entrada, venta, ajuste, devolución) y pantalla de "conteo" que compara esperado vs contado y genera ajuste. Scrampi lo llama requisito 2026. `finds-betters.md` §7 pide "diferencia inventario esperado vs real" pero no hay mecanismo. | `stock_entries` no distingue tipo. No hay ajuste negativo con motivo (merma, robo, vencido). | 2 días: columna `type` en `stock_entries` (`purchase, adjustment, return, loss`), vista kardex por producto, pantalla conteo por categoría. |
| G3 | **Fecha de vencimiento por producto / lote** | Tiendas venden alimentos, farmacia, cosméticos. Alerta "vence en 15 días" + reporte de vencidos. | No existe `expiry_date` en `products` ni lotes. | 1 día versión simple: `expiry_date` a nivel producto + alerta en dashboard. 3 días versión lotes (FIFO por entrada). |
| G4 | **Combos / kits** | "Paquete desayuno = pan + leche + huevos" con precio propio, descuenta stock de componentes. Vendty: "creación de promociones". | No existe. | 2 días: `product_components (parent_id, child_id, qty)`; al vender el combo, descuenta hijos. |
| G5 | **Impresión de etiquetas con código de barras** | Productos sin barcode (a granel, ferretería) → imprimir etiqueta Code128 con precio. Estándar en POS con inventario. | Existe `products.barcode` y lookup, pero nada genera códigos ni etiquetas. | 1 día: generar EAN-13 interno (prefijo 2xx = uso interno) + página imprimible de etiquetas (librería JS de barcode, offline). |
| G6 | **Historial de cambios de precio** | "¿Cuándo subí el aceite y a cuánto estaba?" Auditoría básica de precios. | `audit_log` ya registra `price_override` en venta, pero no cambios de precio en producto. | 0.5 día: insertar en `audit_log` al `PUT /products/{id}` cuando cambia `sale_price`/`cost`. Ver en detalle de producto. |
| G7 | **Foto del producto** | Grid visual del POS con imagen (maquillaje, ferretería). Todos los POS modernos. | No hay columna `image`. | 1 día: `products.image` (base64 pequeño o path en carpeta `data/images/`), grid del POS la muestra. Cuidado tamaño DB. |

## H — Específicas de Colombia que los locales sí tienen

| # | Feature | Qué hacen los demás | Estado aquí | Veredicto |
|---|---|---|---|---|
| H1 | **Recargas de celular y pago de servicios** | Tiendatek: "vende recargas y paga servicios, gana comisión extra". Es ingreso real de la tienda de barrio. | No existe. | **Requiere partner** (Movilred, Puntored, SuRed) con API y contrato. Igual que capital embebido: no ejecutable en solitario. Lo que SÍ se puede: registrar la recarga como venta tipo `service` con la comisión como ganancia, para que cuadre la caja. 0.5 día. |
| H2 | **Apartados / separados (layaway)** | Cliente abona $20K hoy, $30K la otra semana, se lleva el producto cuando complete. Muy común en ropa, electrodomésticos, ciclomontería. | No existe. El crédito actual es "se lleva y paga después"; apartado es el inverso. | 2 días: `layaways (customer_id, product_id, total, status)` + abonos reutilizando `customer_payments` con `layaway_id`. Stock queda reservado. Encaja natural con el diferenciador de fiado. |
| H3 | **Domicilios / pedidos** | Treinta: "recibos por WhatsApp, ideal para domicilios". Pedido pendiente → despachado → cobrado (contraentrega). | No existe estado intermedio de venta. | 2 días: `sales.status` gana `pending_delivery`, campo `delivery_address`, `delivery_fee`, lista "Pedidos del día". Ticket para el domiciliario. |
| H4 | **Impuesto a bolsas plásticas** | Obligatorio DIAN (INC bolsas). Los POS colombianos lo tienen como ítem automático. | No existe. | 0.5 día: producto tipo `service` "Bolsa plástica" con valor vigente configurable, botón rápido en POS. Importa para Documento Equivalente. |
| H5 | **Cotizaciones** | Ferretería/distribuidora: cliente pide cotización, vuelve en 3 días, se convierte en venta. Helisa/Siigo estándar. | No existe. | 2 días: `quotes` con items, PDF/ticket, botón "convertir a venta" que carga el carrito. Reutiliza casi todo el POS. |

## I — Control para el dueño

| # | Feature | Qué hacen los demás | Estado aquí | Esfuerzo |
|---|---|---|---|---|
| I1 | **Pantalla de auditoría** | "¿Quién anuló la venta #453? ¿Quién cambió el precio del arroz?" Detecta hurto interno — dolor #1 del dueño con empleados. | Tabla `audit_log` existe (migration6) pero solo se escribe en `price_override` y no hay UI ni endpoint. | 1 día: registrar también `void`, `discount`, `cash_out`, `product_price_change`; `GET /audit` + página filtrable por usuario/fecha. **Alto impacto, casi todo ya existe.** |
| I2 | **Resumen del día por WhatsApp al dueño** | Al cerrar caja, el dueño (que no está en la tienda) recibe: vendido, efectivo, fiado nuevo, diferencia de caja. Alternativa offline a "app móvil del dueño" (bloque C.5). | Existe `wa.me` para cobro a cliente (US-035). No para el dueño. | 0.5 día: botón "Enviar resumen" en cierre de caja → `wa.me/<tel_dueño>?text=…`. Cero API. Mata el 80% de la necesidad de app móvil. |
| I3 | **Comisiones por vendedor** | Ferretería/ciclomontería pagan % por venta. Reporte ventas por usuario ya sirve de base. | `sales.user_id` existe. No hay % ni reporte de comisión. | 1 día: `users.commission_pct`, reporte "comisiones del mes" por usuario. |
| I4 | **Permisos granulares por cajero** | "El cajero no puede anular ni dar descuento >10% sin PIN de admin". Toast/Vendty lo hacen por rol. | Solo `admin`/`cashier`. `allow_global_discount` es global, no por usuario. | 1–2 días: flags en `users` (`can_void`, `can_discount`, `max_discount_pct`) + "autorización de supervisor" (modal PIN admin) en POS. Combina con F7. |

## J — Priorización Ronda 2 (por venta al primer cliente, no por complejidad)

| Orden | Feature | Por qué primero |
|---|---|---|
| 1 | **F1 Pago mixto** | Único gap de esta ronda que un tendero nota en el primer día. "No puedo cobrar mitad Nequi mitad efectivo" = devolución del sistema. |
| 2 | **I1 Auditoría UI** | Casi gratis (tabla existe), y "saber qué hizo el empleado" vende el sistema al dueño. |
| 3 | **F2 Venta en espera** | 1 día, sin backend, dolor diario en hora pico. |
| 4 | **F4 Gastos** | Sin esto, la "ganancia bruta" de US-039 es mentira útil. Con gastos → ganancia neta = el número que el dueño quiere. |
| 5 | **G1 Proveedores + compras** | Desbloquea US-044 (sugerencia de pedido), actualiza costos reales, corrige margen. |
| 6 | **F5 + F6 Arqueo denominaciones + ticket de cierre** | 1.5 días juntos, profesionaliza el cierre. |
| 7 | **I2 Resumen WhatsApp al dueño** | 0.5 día, sustituye app móvil por ahora. |
| 8 | **H2 Apartados** | Extiende el diferenciador de fiado hacia el caso inverso. Nadie barato lo hace bien. |
| 9 | **G2 Kardex + ajustes** | Requisito 2026 según guías, pero el tendero lo pide al mes 2, no al día 1. |
| 10 | **F7 + I4 PIN + permisos** | Necesario en cuanto haya >1 empleado. |

**Descartado en solitario:** H1 recargas/servicios (requiere partner con contrato, igual que capital embebido). Se anota como visión con socio.

**No incluido por ya estar cubierto en docs previos:** ESC/POS, descuento por ítem, mayorista, WhatsApp cobro, cartera envejecida, margen, variantes, catálogo, multi-caja LAN, IA local, scoring fiado, backup USB/Drive, onboarding, app móvil.

## Fuentes Ronda 2
- https://scrampi.com/blog/sistema-pos-para-tiendas-de-barrio-colombia/
- https://scrampi.com/blog/sistema-pos-gratis-colombia/
- https://tiendatek.info/co
- https://treinta.co/blog/mejores-sistemas-pos-para-negocios-en-colombia-2025-guia-por-tipo-de-negocio
- https://www.vendty.com/sistema-pos
- https://helisa.com/productos/puntos-de-venta-pos/
- https://siigopyme.portaldeclientes.siigo.com/basedeconocimiento/arqueo-de-caja-cajero/
- https://www.siigo.com/sistema-pos/tiendas/

## G8 — Reportes por hora pico y por vendedor (adición 2026-09-08)

`internal/repository/report.go` agrupa ventas por timestamp individual, método de pago y producto. **No agrupa por hora del día ni por `user_id`.** Listas genéricas de "funciones de un POS" lo dan por básico. 0.5 día: dos queries `GROUP BY strftime('%H', created_at)` y `GROUP BY user_id`, dos tarjetas en `/reports`. Base para I3 (comisiones).

## K — Cómo darle al dueño visibilidad desde el celular sin nube

| Opción | Cómo | Alcance | Esfuerzo | Veredicto |
|---|---|---|---|---|
| **K1 — POS abierto en WiFi de la tienda** | `cmd/pos/main.go:41` escucha en `localhost:8080` → celular en misma WiFi no entra hoy. Setting "Permitir acceso desde celular" → bind `0.0.0.0:8080`, QR con IP LAN en settings. Frontend ya es mobile (SPRINT-05). | Solo dentro del local, en vivo | 0.5 día | ✅ Hacer. Advertir: solo con WiFi con clave. JWT ya protege login. |
| **K2 — Resumen por WhatsApp al cerrar caja** | = I2. Botón en cierre → `wa.me/<tel_dueño>?text=…` con vendido/efectivo/fiado/diferencia. Variante: "Enviar bajo stock" en dashboard. | Fuera del local, 1 vez al día, requiere clic del cajero | 0.5 día | ✅ Hacer. Cero API. |
| **K3 — Envío automático (WhatsApp API / correo SMTP)** | Meta Cloud API o intermediario (Treble, Kommo); o SMTP con credenciales Gmail. | Fuera del local, automático | 3+ días + costo recurrente | ❌ Ahora no. Requiere internet, cuenta business, config técnica. Contradice "offline sin mensualidad". Add-on pago v2 si un cliente lo pide. |
| **K4 — Acceso remoto en vivo (tunnel/nube)** | Cloudflare Tunnel / Tailscale / réplica cloud. | Desde cualquier lugar, en vivo | 5+ días | ❌ v2. Solo cuando cliente pregunte "¿y desde la casa?". |

**Recomendación:** K1 + K2 juntas = 1 día. Cubre "ver ventas en celular" (en tienda) y "recibir resumen" (fuera) sin romper posicionamiento offline.
