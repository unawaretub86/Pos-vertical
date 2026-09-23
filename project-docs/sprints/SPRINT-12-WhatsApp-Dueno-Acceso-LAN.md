# SPRINT-12 — Resumen por WhatsApp al Dueño y Acceso al POS desde el Celular

**Objetivo:** Darle a Don Carlos (dueño, no siempre en la tienda) visibilidad del negocio desde el celular sin depender de nube ni mensualidad: un resumen de cierre de caja por WhatsApp con un clic, y la posibilidad de abrir el POS desde su celular mientras está en la tienda, conectado al mismo WiFi.

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/better-finds-pos.md` (bloque K — K1, K2/I2), `project-docs/SPRINT-10-Cobro-Mixto-Auditoria-Gastos.md` (roadmap), `project-docs/SPRINT-11-QA-Proveedores-Cierre-Caja.md` (roadmap)

**Deuda técnica entrante:**
- US-034 — Factus sandbox validation (abierta desde SPRINT-07, requiere acceso externo a Factus) — **sigue abierta**, no bloqueante, sin acción en este sprint.
- SPRINT-05 — Mobile Responsive (marcado ⬜ Pendiente en `BACKLOG-Development-Workflow.md`) — **sigue abierta**, no relacionada con este sprint (aunque este sprint sí exige responsive 320px en lo que toca).
- US-042c, US-043, US-044, US-045 (SPRINT-09, ⬜ Pendiente) — **sigue abierta**. US-044 (sugerencia de pedido) ya no está bloqueada desde el cierre de SPRINT-11 (US-052 le dio proveedores), pero su implementación en sí no entra aquí.

**Nota de tamaño:** este sprint son 2 US de ~0.5 día cada una (el propio doc fuente las documenta como "Combo de 1 día" en el bloque K). Es intencionalmente corto — ambos sprints anteriores (10 y 11) ya fijaron esta combinación específica en su Roadmap; no se infla con features de SPRINT-13 (H2 apartados, G2 kardex) para no romper esa decisión ya tomada dos veces.

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-053 | Resumen de cierre de caja por WhatsApp al dueño (I2) | [ ] Pendiente | — |
| US-054 | Acceso al POS desde el celular por WiFi local (K1) | [ ] Pendiente | — |

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-053 | Aditivo puro: una columna nueva y un botón. No toca cómo arranca el servidor — cero riesgo de dejar el POS inalcanzable. |
| **2** | US-054 | Toca el bind del servidor (`cmd/pos/main.go`) — el único cambio de este sprint con radio de impacto real (si algo sale mal, nadie entra al POS). Se hace después, con US-053 ya probada y la migración 14 ya en pie. |

---

## US-053 — Resumen de cierre de caja por correo automático al dueño (I2, pivote)

**Como** Don Carlos (dueño, no siempre presente en la tienda),
**quiero** recibir por correo el resumen del cierre de caja (vendido por método, entradas, salidas, esperado vs. real) automáticamente apenas Laura cierra caja, sin que nadie tenga que hacer clic en nada,
**para** saber cómo fue el día sin depender de que el cajero recuerde enviarlo.

**Esfuerzo estimado:** 0.5 día. **Impacto:** Alto — cubre la necesidad de "app móvil del dueño" sin fricción de flujo.

**Pivote (2026-09-12):** la versión original de esta US usaba un botón "Enviar por WhatsApp" (`wa.me`, clic manual del cajero). En pruebas manuales, Andrey encontró que el navegador bloqueaba el popup de WhatsApp silenciosamente (sin error de consola) y, sobre todo, que un paso manual extra en el flujo de cierre no era el UX deseado — pidió explícitamente algo automático e integrado. Se evaluaron alternativas de automatización de WhatsApp (Business Cloud API de Meta: requiere verificación de negocio y cobra por conversación; librerías no oficiales tipo `whatsapp-web.js`: violan ToS de WhatsApp, riesgo de baneo) y se descartaron. Se decidió cambiar de canal a **correo vía Gmail SMTP**, enviado automáticamente desde el backend al cerrar caja — sin clic, sin costo (dentro del límite gratuito de Gmail), sin librerías nuevas (usa `net/smtp` del stdlib de Go). Nota: esto sí requiere internet en el momento del cierre para que el correo salga (a diferencia del click-to-WhatsApp, que no lo requería hasta que el dueño abriera el mensaje) — si no hay internet, el envío falla silenciosamente y solo se registra en el log del servidor; el cierre de caja en sí nunca se bloquea ni se reintenta.

**Contexto técnico (implementado):**
- **Migración 14** (compartida con US-054 — ver US-054 para el segundo bloque):
  ```sql
  ALTER TABLE business_config ADD COLUMN owner_email TEXT;
  ALTER TABLE business_config ADD COLUMN owner_email_app_password TEXT;
  ```
- `internal/repository/setup.go` — `BusinessConfig` gana `OwnerEmail` y `OwnerEmailAppPassword`; `Get()`/`Upsert()` incluyen ambas columnas. Sin endpoint nuevo — `PUT /setup` ya acepta el body completo.
- `internal/mailer/mailer.go` (nuevo paquete) — `BuildRegisterSummaryEmail(data) (subject, body string)` pura y testeada (Go, no frontend), y `Send(cfg SMTPConfig, subject, body string) error` que envía por `smtp.gmail.com:587` (STARTTLS vía `net/smtp.SendMail`, sin dependencia externa). El mismo correo Gmail actúa como remitente y destinatario — un solo par de credenciales a configurar.
- `internal/handler/register.go` (`closeRegister`) — tras cerrar la caja con éxito, dispara `go h.sendRegisterSummaryEmail(summary, cashier)` en goroutine (fire-and-forget): si `owner_email`/`owner_email_app_password` no están configurados, o el envío falla (sin internet, credenciales inválidas), se loguea con `log.Printf` y no afecta la respuesta HTTP — el cierre ya quedó persistido antes de intentar el envío.
- `frontend/app/settings/page.tsx` — reemplaza el campo de teléfono por dos `ConfigRow` en Administración: "Correo del dueño" (`type="email"`) y "Contraseña de aplicación" (`type="password"`, con nota de que es la contraseña de aplicación de Gmail, no la contraseña normal de la cuenta).
- `frontend/components/CloseTicketPreview.tsx` y `frontend/app/register/page.tsx` — se revierte el botón de WhatsApp y el plumbing de `ownerPhone`; el ticket de cierre vuelve a tener solo el botón "Imprimir" (el correo ya se envió solo, no hace falta acción del cajero).
- `frontend/lib/registerSummaryMessage.ts` (creado en el intento con WhatsApp) se elimina — la lógica de armar el mensaje vive ahora en Go (`internal/mailer`), no en el frontend, porque el envío ya no lo dispara el navegador.

**Regla de negocio (igual que antes, ahora en el cuerpo del correo):**
```
cuerpo = "Resumen de cierre — <business.name>\n"
       + "Cajero: <cashier> · <closed_at formateado>\n\n"
       + para cada línea en by_payment_method: "<label(method)>: <formatCOP(total)>\n"
       + "Entradas: <formatCOP(cash_in)>\n"
       + "Salidas: <formatCOP(cash_out)>\n"
       + "Esperado: <formatCOP(expected)>\n"
       + "Real: <formatCOP(actual_close)>\n"
       + si difference < 0 → "Faltante: <formatCOP(abs(difference))>"
         si no             → "Diferencia: <formatCOP(difference)>"
```

**Scope — qué NO entra:**
- Múltiples destinatarios (varios dueños/socios) — un solo `owner_email`, ampliable si se pide.
- Reintentos o cola de envío si falla (sin internet, credenciales vencidas) — se loguea server-side y ya; no hay UI de "correos pendientes de reenviar".
- Envío de PDF/imagen adjunta del ticket — el correo es texto plano, igual que era el mensaje de WhatsApp.

**Acceptance Criteria:**

**Configuración:**
- [x] Campos "Correo del dueño" y "Contraseña de aplicación" en Configuración → Administración, se guardan con `PUT /setup` igual que el resto del formulario.
- [x] Vacíos por defecto — no rompe negocios ya configurados (columnas nuevas con `NULL`/`''`).

**Cierre de caja:**
- [x] Al cerrar caja con `owner_email` y `owner_email_app_password` configurados, el backend envía automáticamente el correo con el resumen — sin clic ni acción del cajero.
- [x] Sin `owner_email` configurado: el cierre funciona exactamente igual (no se intenta enviar nada, no hay error visible).
- [x] Si el envío falla (sin internet, credenciales inválidas): el cierre de caja se completa igual, el error solo queda en el log del servidor.
- [x] Cierre con diferencia negativa (faltante) → cuerpo del correo dice "Faltante: $X", no "Diferencia: -$X".

**DoD:**
- [x] Migración 14 (compartida con US-054): columnas `owner_email` y `owner_email_app_password` en `business_config` — ver SQL exacto arriba.
- [x] `GET /setup` y `PUT /setup` (ya existentes) incluyen ambos campos en el JSON, sin cambio de ruta ni de grupo de auth.
- [x] Test (Go, `internal/mailer`): `BuildRegisterSummaryEmail` con un cierre de ejemplo — ventas $60.000 (cash $30.000 + transfer $20.000 + credit $10.000), cash_in $0, cash_out $5.000, expected $25.000, actual_close $24.500 → cuerpo contiene "Faltante: $500" y las 3 líneas de método con sus montos exactos.
- [x] Test (Go, `internal/mailer`): mismo caso pero `actual_close` = `expected` exacto (diferencia 0) → cuerpo dice "Diferencia: $0", no "Faltante: $0".
- [x] Build limpio (`go build ./...`, `npm run build`).
- [ ] Test manual (no automatizable sin cuenta Gmail real — pendiente para Andrey): configurar correo + contraseña de aplicación real, cerrar caja, confirmar que el correo llega con los montos correctos.

---

## US-054 — Acceso al POS desde el celular por WiFi local (K1)

**Como** Don Carlos (dueño, presente en la tienda),
**quiero** abrir el POS desde el navegador de mi celular mientras estoy conectado al WiFi del local, sin instalar nada,
**para** ver las ventas del día en vivo sin pararme frente al computador de la caja.

**Esfuerzo estimado:** 0.5 día. **Impacto:** Medio-alto — cubre "ver ventas en celular" en vivo, complementa a US-053 (que cubre el resumen fuera del local).

**Estado verificado:**
- `cmd/pos/main.go:41` tiene `addr := "localhost:8080"` hardcodeado — un celular en la misma WiFi que la caja no puede entrar hoy (`localhost` solo acepta conexiones desde la misma máquina).
- El frontend ya es responsive/mobile-first desde SPRINT-05 (aunque esa US sigue marcada ⬜ Pendiente en el backlog general, la base de estilos mobile ya se usó y se validó en SPRINT-11 para los componentes nuevos).
- El login usa JWT (`internal/middleware`, confirmado por uso en `appmw.GetClaims` en múltiples handlers) — la app ya exige autenticación aunque el puerto quede expuesto en la LAN.
- No existe ningún endpoint que reporte la IP local del equipo (`grep -rn "InterfaceAddrs\|network-info" internal/` no encuentra nada).

**Contexto técnico:**
- **Migración 14** (misma migración que US-053 — un solo archivo `migration14` con ambos `ALTER TABLE`):
  ```sql
  ALTER TABLE business_config ADD COLUMN lan_access_enabled INTEGER NOT NULL DEFAULT 0;
  ```
  Default `0` = comportamiento actual sin cambios para toda base de datos existente.
- `internal/repository/setup.go` — `BusinessConfig` gana `LanAccessEnabled bool \`json:"lan_access_enabled"\``, incluido en `Get()`/`Upsert()` igual que `AllowGlobalDiscount` (mismo patrón booleano ya usado en esa misma tabla).
- `cmd/pos/main.go` — extraer una función pura `listenAddr(lanEnabled bool) string` (`"0.0.0.0:8080"` si `true`, `"localhost:8080"` si `false`), llamada después de `database.New(...)` y antes de `http.ListenAndServe`, leyendo el flag con `h.setup` (o el repo directo) recién creado el `Handler`. **El cambio solo aplica al reiniciar el binario** — Go ya tiene el socket abierto en el bind original; no hay forma de re-bindear sin reiniciar el proceso.
- Nuevo endpoint `GET /api/network-info` (grupo admin, mismo patrón que `/api/audit`) → `{lan_ip: string, port: string, enabled: bool}`. `lan_ip` se obtiene con el truco estándar de Go: abrir un `net.Dial("udp", "8.8.8.8:80")` (no envía tráfico real, solo hace que el SO resuelva qué interfaz usaría) y leer `LocalAddr()`; si falla (sin red en absoluto), devolver `lan_ip: ""` y el frontend muestra "No se detectó red local" en vez de romper.
- `frontend/app/settings/page.tsx` — nueva `ConfigRow` "Acceso desde celular (WiFi local)": toggle `lan_access_enabled` (se guarda con el mismo `PUT /setup` del formulario existente); debajo, cuando está activado, `GET /api/network-info` y mostrar `http://<lan_ip>:8080` en fuente monoespaciada grande + botón "Copiar"; texto de advertencia fijo: "Solo funciona dentro de la WiFi de este local. Asegúrate que tenga contraseña. Debes reiniciar la aplicación para que el cambio tome efecto."

**Regla de negocio:** N/A (flag de configuración + detección de red, sin aritmética).

**Scope — qué NO entra:**
- Código QR — el doc fuente lo sugiere, pero agregar una librería nueva solo para esto no se justifica frente a mostrar la IP en texto grande y copiable; si un cliente real lo pide, es un cambio pequeño y aislado en el mismo componente.
- Acceso remoto fuera de la LAN (tunnel/nube) — eso es K4 en el doc fuente, `v2` explícito, requiere 5+ días y contradice el posicionamiento offline.
- Hot-reload del bind sin reiniciar — limitación real de `http.ListenAndServe`, se documenta en la UI en vez de resolverse con complejidad adicional (ej. reverse proxy interno).

**Acceptance Criteria:**

**Configuración:**
- [ ] Toggle "Acceso desde celular" en Configuración, apagado por defecto — bases de datos existentes no cambian de comportamiento sin acción explícita del dueño.
- [ ] Con el toggle activado, se muestra la IP local + puerto y el aviso de reinicio.
- [ ] Si no se detecta ninguna red local (`lan_ip` vacío), se muestra "No se detectó red local" en vez de una IP vacía o un error crudo.
- [ ] Responsive: sección usable en 320px.

**Acceso real:**
- [ ] Con el toggle activado y la aplicación reiniciada, un celular conectado a la misma WiFi abre `http://<lan_ip>:8080` y ve la pantalla de login.
- [ ] Con el toggle desactivado (default), el servidor sigue escuchando solo en `localhost:8080` — comportamiento idéntico al actual.
- [ ] Login desde el celular funciona igual que desde el computador de caja (mismo JWT, mismos roles).

**DoD:**
- [ ] Migración 14 (compartida con US-053): columna `lan_access_enabled` — ver SQL exacto arriba.
- [ ] `GET /api/network-info` (grupo admin) → `{lan_ip, port, enabled}`.
- [ ] `GET /setup` y `PUT /setup` incluyen `lan_access_enabled`.
- [ ] Test (Go, `cmd/pos` o paquete extraído): `listenAddr(true) == "0.0.0.0:8080"`, `listenAddr(false) == "localhost:8080"`.
- [ ] Test (Go, `internal/handler`): `GET /api/network-info` sin sesión admin → 403; con sesión admin → 200 con `port: "8080"`.
- [ ] Test manual (no automatizable sin dos dispositivos en red): activar toggle, reiniciar `pos.exe`, confirmar acceso desde un celular real en la misma WiFi — **pendiente para Andrey**, mismo criterio que los checks de GitHub Actions de US-050 que no corren en este entorno.
- [ ] Build limpio (`go build ./...`, `npm run build`).

---

## Criterio de éxito del sprint

- Laura cierra caja, Don Carlos recibe en su WhatsApp (sin estar en la tienda) el resumen exacto de cómo fue el día — vendido por método, entradas, salidas, si sobró o faltó plata.
- Don Carlos, parado en su tienda con el celular, activa el acceso LAN una vez, reinicia la app, y desde ese momento entra al POS desde su celular en la misma WiFi para ver las ventas en vivo sin tocar el computador de la caja.

---

## Notas técnicas para el dev

- **US-053 y US-054 comparten la migración 14** — un solo archivo/función `migration14` con ambos `ALTER TABLE`, mismo criterio que SPRINT-10 aplicó a US-046/US-049.
- El botón de WhatsApp de US-053 y el `whatsappUrl` de `frontend/app/customers/page.tsx` deberían compartir el prefijo `57` y la lógica de limpiar dígitos (`replace(/\D/g, '')`) — vale la pena extraer un helper común `lib/whatsapp.ts` en vez de duplicarlo por tercera vez si aparece un cuarto caso de uso.
- `listenAddr()` en US-054 debe quedar como función pura exportada (no inline en `main()`) específicamente para poder testear sin levantar un servidor real — `main.go` hoy no tiene ningún test porque todo su contenido es orquestación; esta es la primera pieza de lógica ahí que vale la pena aislar.
- Advertir a Andrey en la demo: el toggle de US-054 exige reinicio manual del `.exe` — no es un bug, es una limitación de cómo `net/http` bindea el socket. Si en el futuro se quiere que el toggle aplique en caliente, la solución real es un segundo listener condicional o un proxy interno — no vale la pena para este sprint.

---

## Roadmap

Lo que queda por construir vive en [roadmap.md](roadmap.md), que es la única fuente. Este documento no reserva números de sprint futuros.

> Antes había aquí una tabla de roadmap propia. Se consolidó el 2026-09-21: las copias por sprint divergieron entre sí y llegaron a reclamar el mismo número de sprint para features distintas.
