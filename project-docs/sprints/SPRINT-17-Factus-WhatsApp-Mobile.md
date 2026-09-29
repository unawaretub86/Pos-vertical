# SPRINT-17 — Dependencias Críticas: Factus, WhatsApp y Mobile

**Objetivo:** Retomar y cerrar las deudas técnicas más importantes vinculadas a diferenciadores y dependencias externas (Factus, comunicaciones y acceso móvil), garantizando que el POS sea verdaderamente usable en el celular y cumpla las promesas de facturación y notificaciones.

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/gaps.md`, `project-docs/sprints/SPRINT-05-Mobile-Responsive.md`, `project-docs/sprints/SPRINT-07-Factus-Sandbox-Diferenciadores.md`, `project-docs/sprints/SPRINT-09-Historial-Pedidos-Catalogo.md`

**Deuda técnica entrante:**
- SPRINT-05 (Mobile Responsive): Se integran aquí las correcciones finales en un solo bloque (US-074).
- US-034 (Factus sandbox): Pendiente de validación externa real.
- US-045 (Catálogo por WhatsApp): Pendiente desde SPRINT-09.
- US-053 (Cierre por correo): `mailer.go` fue creado pero no integrado al cierre de caja (SPRINT-12).

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-034 | Factus: validación completa en sandbox | [ ] Pendiente | — |
| US-045 | Catálogo compartible por WhatsApp | [ ] Pendiente | — |
| US-053 | Resumen de cierre de caja por correo | [ ] Pendiente | — |
| US-074 | Consolidación Mobile Responsive (320px) | [ ] Pendiente | — |

**Esfuerzo total estimado: ~5.5 días**

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-053 | Muy bajo esfuerzo. El código en `mailer.go` ya existe, solo falta integrarlo en `register.go` y probar. Da un quick win de comunicación externa. |
| **2** | US-034 | Desbloquea la facturación electrónica DIAN (la dependencia externa de mayor riesgo). Requiere pruebas puras de API y poller. |
| **3** | US-045 | Diferenciador de alto impacto (WhatsApp). Implementación fullstack estática que agrega valor directo a Don Carlos. |
| **4** | US-074 | Correcciones de UI que se benefician de probar el resto del sistema. Cierra la promesa de "Demo-Ready en Celular". |

---

## US-053 — Resumen de cierre de caja por correo

**Como** Don Carlos (dueño),
**quiero** recibir un email automático al final del día con el cuadre de la caja,
**para** no tener que estar físicamente en la tienda durante el cierre ni revisarlo manualmente.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Medio. Aporta confianza y control a distancia.

**Estado verificado:**
- `internal/mailer/mailer.go` tiene `BuildRegisterSummaryEmail` y `Send` implementados.
- `internal/service/register.go:Close` no llama a `mailer.Send`.
- No hay interfaz UI para configurar el correo del dueño (requiere SMTPConfig: Email y AppPassword).

**Regla de negocio:**
- Si en `business_config` (o tabla nueva de config de correo) están las credenciales SMTP configuradas, al hacer `Close` de caja se despacha el correo de manera asíncrona.
- El error de envío no debe bloquear el cierre de la caja en la BD.

**DoD:**
- [ ] Configuración SMTP guardada en DB (`business_config` o nueva `email_settings`).
- [ ] UI simple en Ajustes para ingresar Gmail y App Password.
- [ ] `internal/service/register.go` llama a `mailer.Send` asíncronamente (goroutine) al hacer `Close`.
- [ ] Endpoint de test `/api/settings/email/test` para verificar credenciales.
- [ ] Pruebas unitarias/integración de que el cierre de caja no falla si el correo falla.

---

## US-034 — Factus: validación completa en sandbox

**Como** desarrollador,
**quiero** probar el flujo completo contra el sandbox de Factus,
**para** confirmar que el payload es exacto y la facturación electrónica funciona antes de ir a producción.

**Esfuerzo estimado:** 1 día. **Impacto:** Crítico. Facturación electrónica es mandatoria en muchos casos (Distribuidoras).

**Estado verificado:**
- `internal/factelect/client.go` y `poller.go` existen.
- Estructura y middleware están listos.
- Falta la validación empírica contra la API `developers.factus.com.co`.

**Acceptance Criteria & Preguntas a resolver:**
- [ ] Confirmar si `payment_form` es string o int en Factus API y corregir struct si es necesario.
- [ ] Confirmar URL endpoint (`/api/v1/bills` vs `/v1/bills`).
- [ ] Confirmar manejo de `is_excluded` y `tribute_id` para impuestos cero.

**DoD:**
- [ ] Se puede crear un Documento Soporte/Factura en sandbox con `ambiente: "test"` y config real.
- [ ] Response de Factus retorna `status: "pending"` o `"accepted"` (no error 4xx).
- [ ] El poller en background detecta y actualiza el doc en BD a `"accepted"` cuando DIAN valida.
- [ ] Todos los campos de payload de `client.go` están verificados contra la doc oficial y ajustados.

---

## US-045 — Catálogo compartible por WhatsApp

**Como** Don Carlos (dueño),
**quiero** generar un link con mis productos y precios para enviárselo a clientes por WhatsApp,
**para** que puedan ver mi inventario disponible y me hagan pedidos directamente.

**Esfuerzo estimado:** 2.5 días. **Impacto:** Alto. Diferenciador comercial.

**Estado verificado:**
- Completamente pendiente. Sin ruta `/public/catalog`, sin columna de token en DB.

**Scope — qué NO entra:**
- Fotos de productos (demasiado peso en local).
- Carrito de compras/pedidos dentro de la página. Es un catálogo estático.

**Acceptance Criteria:**
- [ ] Toggle en Settings: "Catálogo público" (default apagado).
- [ ] Al activarlo se genera link único `.../catalogo/[token]`.
- [ ] Botón "Compartir por WhatsApp" abre `wa.me` con mensaje invitando a ver el catálogo.
- [ ] Vista `/catalogo/[token]` pública y responsive que lista productos activos con precio > 0.
- [ ] Precio mayorista oculto.

**DoD:**
- [ ] Nueva migración agregando `catalog_token` (TEXT) y `catalog_enabled` (BOOLEAN) a `settings` o `business_config`.
- [ ] Endpoint `GET /public/catalog/:token` registrado **fuera** del JWT middleware.
- [ ] Renderizado en UI Next.js desde exportación estática que consume la API.
- [ ] Pruebas confirmando que sin token o con toggle apagado retorna 404.

---

## US-074 — Consolidación Mobile Responsive (320px)

**Como** cajero o dueño,
**quiero** que todas las pantallas críticas sean utilizables en un celular normal,
**para** hacer ventas en el local o revisar reportes sin necesitar un PC ni tablet.

**Esfuerzo estimado:** 1.5 días. **Impacto:** Alto. Necesario para demostraciones y clientes pequeños.

**Estado verificado:**
- `useIsMobile.ts` existe y se usa en varios componentes, pero hay recortes de texto y layouts rotos pendientes del SPRINT-05.
- La tabla de movimientos y el POS tienen scroll horizontal forzado o botones inaccesibles.

**Acceptance Criteria:**
- [ ] Dashboard: Las tarjetas de "Transacciones" y "Ventas hoy" no cortan el texto, se apilan o reducen fuente.
- [ ] Navegación: Sidebar funciona correctamente con el botón hamburger en todas las rutas sin parpadeos.
- [ ] POS: Buscador ocupa 100% del ancho, botón de "Cobrar" visible sin hacer scroll excesivo.
- [ ] Modal de cobro/Ticket: Funciona sin que el teclado del celular oculte los botones de acción.
- [ ] Clientes/Inventario: Tablas sin overflow X incontrolable (esconder columnas secundarias en `< 768px`).

**DoD:**
- [ ] No se requieren librerías extra (usar media queries o `useIsMobile` actual).
- [ ] Verificación manual obligatoria en emulador 320px o celular real.
- [ ] CI limpio (`npm run lint` o `build`).

---

## Criterio de éxito del sprint
- El dueño puede encender su catálogo y pasarlo por WhatsApp.
- Al cerrar la caja, llega un correo estructurado.
- El POS se ve profesional y sin recortes en un iPhone o Android barato.
- La facturación electrónica con Factus dejó de ser un riesgo hipotético y pasó al sandbox real.
