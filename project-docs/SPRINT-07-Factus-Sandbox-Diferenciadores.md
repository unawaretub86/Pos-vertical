# SPRINT-07 — Factus Sandbox Validation + Diferenciadores de Producto

**Objetivo:** Dos tracks paralelos: (1) dejar Factus listo para usarse con un cliente real pasando por sandbox, (2) implementar los diferenciadores de alto impacto que ningún competidor barato ofrece.

**Fecha inicio:** 2026-05-13

**Referencias:** `project-docs/better-pos.md`, `project-docs/finds-betters.md`, `project-docs/SPRINT-06-Facturacion-Electronica.md`

---

## Tareas

| US | Nombre | Track | Estado | Fecha |
|----|--------|-------|--------|-------|
| US-033 | Factus: numbering_range_id y municipality_id configurables | Factus | ✅ Completo | 2026-05-13 |
| US-034 | Factus: validación completa en sandbox | Factus | ⬜ Pendiente | — |
| US-035 | WhatsApp: botón de cobro con link directo | Diferenciadores | ⬜ Pendiente | — |
| US-036 | Clientes: notas y tope de crédito | Diferenciadores | ⬜ Pendiente | — |
| US-037 | POS: precio mayorista activable en carrito | Diferenciadores | ⬜ Pendiente | — |
| US-038 | Dashboard: vista "cuaderno digital" | Diferenciadores | ⬜ Pendiente | — |

---

## TRACK A — Factus: Fix + Sandbox Validation

### US-033 — Factus: numbering_range_id y municipality_id configurables

**Como** administrador del negocio,
**quiero** configurar el rango de numeración DIAN y el municipio desde el panel fiscal,
**para** que las facturas electrónicas no sean rechazadas por datos hardcodeados incorrectos.

**Problema actual:** `internal/factelect/client.go:205` tiene `"numbering_range_id": 0` hardcoded. Factus rechaza `0`. `municipality_id: 149` (Bogotá) está hardcoded en línea 197.

**Acceptance Criteria:**
- [x] `FiscalConfig` model tiene campos `NumberingRangeID int` y `MunicipalityID int`
- [x] Migration SQL agrega ambas columnas (`municipality_id` con default `149`)
- [x] `GET /api/fiscal-config` retorna ambos campos
- [x] `PUT /api/fiscal-config` acepta y persiste ambos campos
- [x] `billRequestToFactus` usa `cfg.NumberingRangeID` y `cfg.MunicipalityID` (sin hardcoding)
- [x] UI muestra campo "ID Rango de Numeración" con tooltip: "Encuéntralo en tu cuenta Factus → Numeración"
- [x] UI muestra campo "Municipio DIAN (ID)" con default 149
- [x] Campo `NumberingRangeID` es requerido cuando facturación está habilitada

**DoD:**
- [x] Migration v9 aplicada sin errores
- [x] Unit test: `billRequestToFactus` con `NumberingRangeID=5` → payload tiene `"numbering_range_id": 5`
- [x] No existe ningún `0` hardcoded como `numbering_range_id` en el paquete `factelect`
- [x] No existe `149` hardcoded como `municipality_id` en el paquete `factelect`

---

### US-034 — Factus: validación completa en sandbox

**Como** desarrollador,
**quiero** probar el flujo completo contra el sandbox de Factus,
**para** confirmar que los nombres de campos son correctos antes de instalarlo en un cliente real.

**Preguntas a resolver antes de codificar:**
- 🔴 ¿`payment_form: "1"` es string o int? (`client.go:209`)
- 🔴 ¿El endpoint correcto es `/api/v1/bills` o `/v1/bills`? Verificar en `developers.factus.com.co`
- 🟡 ¿`is_excluded: true` cuando `tax_rate == 0` es correcto, o solo debe ir `tribute_id: 22`?

**Acceptance Criteria:**
- [ ] Se puede crear un DEE en sandbox con `ambiente: "test"` y config real
- [ ] Response de Factus retorna `status: "pending"` o `"accepted"` (no error 4xx)
- [ ] Si Factus responde 422, el error body completo se loguea (no solo el status code)
- [ ] El poller detecta y actualiza el doc a `"accepted"` cuando DIAN valida
- [ ] Todos los campos del payload verificados contra la documentación oficial

**DoD:**
- [ ] Checklist de campos verificados (agregar como comentario en `client.go`): `numbering_range_id`, `tribute_id`, `unit_measure_id`, `identification_document_id`, `payment_method_code`, `payment_form`
- [ ] Al menos 1 DEE exitoso en sandbox con un producto real
- [ ] Error logging en `client.go:post()` incluye el body completo del error (no solo `resp.StatusCode`)
- [ ] Credenciales sandbox configuradas via env vars, no hardcodeadas

---

## TRACK B — Diferenciadores de producto

### US-035 — WhatsApp: botón de cobro con link directo

**Como** Don Carlos (dueño),
**quiero** enviar un mensaje de cobro por WhatsApp con un clic desde la vista del cliente,
**para** recordarle su deuda sin escribir el mensaje manualmente.

**Esfuerzo estimado:** 1 día. **Impacto:** Alto. Ningún POS barato en Colombia lo tiene.

**Acceptance Criteria:**
- [ ] En vista de cliente con saldo > 0, aparece botón "Cobrar por WhatsApp"
- [ ] Clic abre `wa.me/57[numero]?text=...` con mensaje prellenado: nombre del cliente, monto exacto, nombre del negocio
- [ ] Si el cliente no tiene teléfono registrado, botón deshabilitado con tooltip "Agrega teléfono al cliente"
- [ ] Funciona sin internet (abre WhatsApp instalado en el PC)
- [ ] Mensaje en español: `"Hola [Nombre], le recordamos que tiene un saldo pendiente de $[monto] en [Negocio]. Gracias."`
- [ ] Si el cliente no tiene deuda (saldo = 0), el botón no aparece

**DoD:**
- [ ] Implementado con `wa.me` link — NO usa API de WhatsApp Business
- [ ] URL correctamente encoded (`encodeURIComponent`)
- [ ] `window.open(url, '_blank')` con el link correcto
- [ ] Test manual: link genera el mensaje correcto con valores reales de la BD
- [ ] No se muestra en la lista de clientes sin deuda

---

### US-036 — Clientes: notas y tope de crédito

**Como** Don Carlos,
**quiero** agregar notas y un límite de crédito a cada cliente,
**para** recordar acuerdos como "no fiar más de $150K" o "paga los viernes".

**Esfuerzo estimado:** 2 días. **Impacto:** Alto.

**Acceptance Criteria:**
- [ ] Formulario de cliente tiene campo "Notas" (textarea, max 500 chars) y "Tope de crédito" (número COP, opcional)
- [ ] Notas visibles en el perfil del cliente
- [ ] Al seleccionar un cliente en el carrito POS, si tiene nota, se muestra como alerta inline
- [ ] Si la venta superaría el tope de crédito, mostrar warning rojo: "Este cliente tiene un tope de $X. Su deuda actual es $Y." — la venta NO se bloquea, solo avisa
- [ ] Si no hay tope configurado, no aparece ningún warning de límite

**DoD:**
- [ ] Migration: columnas `notes TEXT` y `credit_limit INTEGER NULL` en tabla `customers`
- [ ] `GET /api/customers/:id` retorna `notes` y `credit_limit`
- [ ] `PUT /api/customers/:id` acepta y persiste ambos campos
- [ ] Warning de tope calculado en frontend: `deuda_actual + monto_venta > credit_limit`
- [ ] Test: cliente con tope $100.000 y deuda $80.000 → venta de $30.000 muestra warning

---

### US-037 — POS: precio mayorista activable en carrito

**Como** Laura (cajera),
**quiero** poder seleccionar el precio mayorista de un producto con un clic al agregar al carrito,
**para** no hacer descuentos manuales a clientes que siempre compran al por mayor.

**Esfuerzo estimado:** 3 días. **Impacto:** Alto.

**Nota:** `wholesale_price` ya existe en BD. Falta el campo en UI de productos y la lógica de selección en el carrito.

**Acceptance Criteria:**
- [ ] Al agregar producto al carrito, si tiene `wholesale_price` configurado, aparece dropdown: "Menudeo: $X" / "Mayoreo: $Y"
- [ ] Si no tiene precio mayorista, no aparece dropdown — solo precio normal
- [ ] El precio seleccionado se refleja en el total del carrito inmediatamente
- [ ] El ticket/factura muestra el precio aplicado
- [ ] En la pantalla de producto (admin), campo "Precio mayorista" visible y editable

**DoD:**
- [ ] No se puede guardar `wholesale_price` menor al costo del producto (validación backend)
- [ ] Si `wholesale_price` es NULL o 0, la UI no muestra el dropdown
- [ ] Test: producto con precio $10.000 / mayoreo $8.000 → carrito muestra el correcto según selección
- [ ] La venta guardada registra el precio real cobrado (no el precio base del producto)

---

### US-038 — Dashboard: vista "cuaderno digital"

**Como** Don Carlos,
**quiero** ver al abrir el sistema un resumen diario que entienda de un vistazo,
**para** saber cómo va el negocio sin revisar reportes complicados.

**Esfuerzo estimado:** 2 días. **Impacto:** Medio-Alto.

**Acceptance Criteria:**
- [ ] Dashboard home muestra: Vendido hoy, Fiado nuevo hoy, Cobrado de fiado hoy, Efectivo en caja estimado, Producto más vendido del día
- [ ] Números en pesos colombianos: formato `$340.000` (punto como separador de miles)
- [ ] Si no hay ventas del día, muestra `$0` (no error ni loading infinito)
- [ ] Datos se actualizan al recargar (no requiere tiempo real / websockets)
- [ ] No hay gráficas en la vista principal — solo números y texto

**DoD:**
- [ ] `GET /api/dashboard/today` retorna todos los valores en un solo endpoint
- [ ] "Cobrado de fiado hoy" = solo pagos de deuda registrados en la fecha actual
- [ ] "Efectivo en caja" = ventas en efectivo del día − retiros del día
- [ ] Test: 3 ventas efectivo + 1 fiado + 1 cobro de deuda → todos los valores correctos
- [ ] Endpoint no falla si no hay sesión de caja abierta

---

## Criterio de éxito del sprint

- Factus emite al menos 1 DEE exitoso en sandbox
- Botón WhatsApp funciona con un cliente real de prueba
- Clientes tienen notas y topes configurables
- Precio mayorista seleccionable en el carrito
- Dashboard muestra los 5 números del día sin errores

---

## Orden de implementación recomendado

| Semana | Items | Razón |
|--------|-------|-------|
| **1** | US-033 + US-035 | Fix Factus config (desbloquea sandbox). WhatsApp es 1 día, impacto inmediato. |
| **2** | US-034 + US-036 | Validar Factus con config real. Notas/tope = alto impacto para Don Carlos. |
| **3** | US-037 + US-038 | Completa diferenciadores core del sprint. |
