# SPRINT-18 — Estabilización y Bloqueadores de Producción

**Objetivo:** Eliminar los 4 bloqueadores críticos (seguridad, consistencia de datos, acceso estático y sesión) que impiden entregar el binario a un cliente real. Este sprint no introduce features de negocio nuevas; deja la fundación sólida para que el sistema no se caiga ni comprometa información en el primer uso.

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/roadmap.md`, `project-docs/gaps.md`

**Deuda técnica entrante:**
- R-55: El binario no sirve el frontend embebido: página en blanco en `:8080`. -> **Se cierra en US-075**.
- R-57: `GET /setup` expone `owner_email_app_password` a cualquier usuario autenticado. -> **Se cierra en US-076**.
- R-49: Redirigir a `/login` cuando cualquier petición responde 401. -> **Se cierra en US-077**.
- R-56: Las llaves foráneas de SQLite no se aplican. -> **Se cierra en US-078**.
- R-35: 42 problemas de linting de React (`set-state-in-effect`, `any`). -> **Se cierra en US-079**.
- Gaps (POS Prodcheck): Soporte nativo ESC/POS para impresoras térmicas. -> **Se cierra en US-080**.
- E1/K1 (US-054 parcial): Pruebas de concurrencia Multi-caja en LAN. -> **Se cierra en US-081**.

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-075 | Binario sirve SPA Next.js correctamente (R-55) | ⬜ Pendiente | |
| US-076 | Ocultar contraseña App Email en GET (R-57) | ⬜ Pendiente | |
| US-077 | Redirección forzada al vencer sesión (R-49) | ⬜ Pendiente | |
| US-078 | Activar Foreign Keys en SQLite (R-56) | ⬜ Pendiente | |
| US-079 | Corrección 42 Errores de Linting (R-35) | ⬜ Pendiente | |
| US-080 | Impresión Nativa ESC/POS 58/80mm | ⬜ Pendiente | |
| US-081 | Concurrencia y Estabilidad Multi-Caja LAN | ⬜ Pendiente | |

**Esfuerzo total estimado: ~6 días**

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-075 | Desbloquea probar el resto de las tareas desde el binario real, en vez del server de desarrollo. |
| **2** | US-076 | Es el cambio más crítico de seguridad y muy rápido (esconder un campo en un JSON). |
| **3** | US-077 | Mejora la experiencia de pruebas de todos, evitando las "pantallas rotas" cuando caduca el token. |
| **4** | US-078 | Cambio de driver de base de datos que podría destapar data huérfana en fixtures, ideal dejarlo para el final y correr todos los tests. |
| **5** | US-079 | Paga la deuda de UI (evitando parpadeos y re-renders) antes de tocar lógica pesada de concurrencia. |
| **6** | US-080 | El módulo de impresión es autocontenido y de alto valor comercial, requiere pruebas con hardware/emuladores. |
| **7** | US-081 | Prueba integral de concurrencia LAN, requiere tener todo el sistema estable antes de simular estrés con 2+ dispositivos. |

---

## US-075 — Binario sirve SPA Next.js correctamente (R-55)

**Como** dueño del negocio,
**quiero** hacer doble clic en el programa y que la interfaz funcione al instante,
**para** poder usar el sistema sin necesitar la consola de Node o estar conectado a internet.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Crítico (El software no abre).

**Contexto técnico:**
- `web/static.go` monta un `http.FileServer` con `embed.FS`.
- Ya hace un `TrimPrefix` de `/` para que `io/fs.ValidPath` no falle (mitad del fix).
- Falta: las exportaciones estáticas de Next sin `.html` al final devuelven 404 en `frontendFS.Open(cleanPath)`, el sistema cae al fallback y sirve `index.html`, rompiéndose las rutas que Next exportó como `<ruta>.html` (por ej. `/login.html`).
- `frontend/lib/api.ts:3` tiene la URL base `/api`, pero debe asegurarse de no requerir CORS estricto de desarrollo o ajustarlo.
- `internal/middleware/middleware.go:96` bloquea llamadas si no vienen de `:3000`. 

**Regla de negocio:**
- Si la URL no es `/api/...`, intenta servir `ruta`, luego `ruta.html`, y finalmente cae a `index.html`. 
- El middleware CORS debe permitir el origen del binario (mismo dominio/puerto) o estar apagado para peticiones locales donde frontend/backend conviven en el mismo puerto.

**Scope — qué NO entra:**
- Modificar el flujo de login.

**Acceptance Criteria:**
- [ ] Compilar el binario (`make build` o equivalente), ejecutarlo y abrir `http://localhost:8080/login`.
- [ ] La pantalla de login carga sin errores 404 de archivos CSS o JS.
- [ ] Hacer login y ver que la redirección a `/dashboard` funciona y llama a la API.

**DoD:**
- Ajuste de `frontendFS.Open` en `web/static.go` intentando `.html` si falla la ruta base.
- Middleware CORS ajustado en `middleware.go` para funcionar en prod (mismo host).
- Build limpio comprobado.

---

## US-076 — Ocultar contraseña App Email en GET (R-57)

**Como** cajero,
**no debería** poder ver las credenciales de correo de la tienda,
**para** evitar el robo de la contraseña de aplicación de Gmail del dueño.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Alto (Fuga de secreto).

**Contexto técnico:**
- `internal/handler/setup.go` tiene `getSetup` que responde todo el `repository.BusinessConfig`, incluyendo `OwnerEmailAppPassword`.
- Este endpoint está protegido solo por `Auth` (cajeros tienen acceso).

**Regla de negocio:**
- El GET `/setup` no debe incluir el `owner_email_app_password`. Puedes serializarlo vacío, o crear un DTO de respuesta que lo omita. 
- Puedes agregar un boolean `has_email_app_password: true` para que la UI sepa que ya está configurado.
- El PUT `/setup` de actualización (que usa upsert) no debe borrar la contraseña existente si el payload la omite, a menos que se quiera explícitamente.

**Acceptance Criteria:**
- [ ] Un usuario logueado con rol `cashier` inspecciona la red en `/setup` y no ve el campo `owner_email_app_password`.
- [ ] Actualizar el setup desde la UI como admin no destruye la contraseña guardada previamente (se mantiene la configuración del correo).

**DoD:**
- Modificar la respuesta en `getSetup`.
- Ajustar lógica en `updateSetup` o `repo.Upsert` para ignorar la sobreescritura de password en blanco.
- 1 test unitario de que `GET /setup` no filtra la credencial.

---

## US-077 — Redirección forzada al vencer sesión (R-49)

**Como** usuario,
**quiero** que si mi sesión caducó me envíen al login,
**para** no enfrentarme a pantallas vacías, errores crípticos o el sistema bloqueado.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Medio-Alto (UX y soporte).

**Contexto técnico:**
- En `frontend/lib/api.ts`, si `res.ok` es falso, simplemente se hace `throw new Error(...)`.
- Los componentes capturan esto y muestran un error genérico o fallan.

**Regla de negocio:**
- Si un request fetch retorna `HTTP 401 Unauthorized`, limpiar la sesión (localStorage, cookies o la función `logout` existente).
- Redirigir a `/login` en el lado del cliente (ej. `window.location.href = '/login'`).

**Acceptance Criteria:**
- [ ] Entrar al POS con sesión activa.
- [ ] Borrar el JWT de localStorage (o esperar 24h).
- [ ] Hacer clic en cualquier vista (Inventario, Dashboard, etc).
- [ ] Ser redirigido instantáneamente a `/login`.
- [ ] La app no debe entrar en un bucle infinito si la página de `/login` hace un fetch que devuelve 401.

**DoD:**
- `frontend/lib/api.ts` ajustado.
- Verificación de que `window.location.pathname !== '/login'` antes de redirigir (opcional pero recomendado).

---

## US-078 — Activar Foreign Keys en SQLite (R-56)

**Como** administrador del sistema,
**quiero** que la base de datos no me deje borrar entidades atadas (como un producto con ventas),
**para** que no se rompan mis reportes de cierres de caja en el futuro.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Alto (Integridad de datos).

**Contexto técnico:**
- `internal/database/database.go` usa `?_foreign_keys=on`, lo cual es sintaxis para `mattn/go-sqlite3` pero ignorado por `modernc.org/sqlite`.
- La sintaxis de `modernc` es `_pragma=foreign_keys(1)`.

**Regla de negocio:**
- Habilitar `PRAGMA foreign_keys = ON` correctamente.
- OJO: Esto puede romper tests existentes si insertan facturas usando `register_id` falsos o no insertados previamente. Se debe ejecutar todos los tests y arreglar los fixtures huérfanos.

**Acceptance Criteria:**
- [ ] Ejecutar todos los tests del backend (`go test ./...`) y que pasen con las FK activadas.
- [ ] Si se intenta hacer un `DELETE` de un `register` con ventas mediante una sentencia SQL directa (o un handler), SQLite debe disparar un error de FK constraint.

**DoD:**
- DSN en `database.go` ajustado.
- Fixtures de tests saneados (ningún test inserta data huérfana).
- Los pipelines/build pasan limpios.

---

## US-079 — Corrección 42 Errores de Linting (R-35)

**Como** desarrollador y usuario final,
**quiero** que el código de React respete las reglas de hooks (`set-state-in-effect`),
**para** que la interfaz no haga re-renders innecesarios que sientan la app "pesada" o con parpadeos.

**Esfuerzo estimado:** 1 día. **Impacto:** Medio (Rendimiento UX / Calidad de código).

**Contexto técnico:**
- El comando `pnpm run lint` arroja 42 errores, principalmente en `AppLayout.tsx`, `useIsMobile.ts`, `ThemeProvider.tsx`.
- El error común es actualizar un estado de React sincrónicamente dentro de un `useEffect` (`react-hooks/set-state-in-effect`).

**Regla de negocio:**
- Mover las lecturas iniciales (como el tema actual o el matchMedia) al estado inicial del `useState` (o inicializador perezoso `useState(() => ...)`), o usar un manejador de eventos asíncrono para que no colapse el render de React 18+.
- Eliminar los tipos `any` injustificados (tipar como `unknown` o la interfaz adecuada).

**Acceptance Criteria:**
- [ ] Ejecutar `pnpm run lint` en `/frontend` debe devolver cero errores y cero warnings.
- [ ] Navegar por la app y verificar que no haya parpadeos al abrir el menú lateral (AppLayout) o al cambiar de modo claro a oscuro (ThemeProvider).

**DoD:**
- Modificaciones a los hooks problemáticos.
- CI/Build limpio (`pnpm lint` && `pnpm build`).

---

## US-080 — Impresión Nativa ESC/POS 58/80mm

**Como** cajero o mesero,
**quiero** que el ticket de venta o la comanda salgan instantáneamente de mi impresora térmica al cobrar,
**para** no tener que hacer clic en la ventana de impresión de Windows ni configurar márgenes en el navegador.

**Esfuerzo estimado:** 2 días. **Impacto:** Muy Alto (Feature crítica comercial).

**Contexto técnico:**
- Actualmente se usa el diálogo de impresión del navegador (`window.print()`).
- Para ESC/POS, Go deberá comunicarse por red (LAN a la IP de la impresora) o mediante conexión USB/Serial (usando una librería de comandos térmicos como `github.com/alexbrainman/printer` en Windows o impresión raw genérica).

**Regla de negocio:**
- Se debe añadir un endpoint en el backend (`POST /api/printer/print-ticket/:sale_id`).
- La configuración del negocio (Settings) debe permitir elegir el método de impresión: `[Browser, ESC/POS Local, ESC/POS Red]`.
- Si es red, guardar IP y puerto. Si es local, guardar el nombre de la impresora compartida de Windows.

**Acceptance Criteria:**
- [ ] En la configuración del POS, el usuario puede habilitar "Impresión Directa".
- [ ] Al hacer un cobro o reimprimir un ticket, el backend despacha el comando ESC/POS puro.
- [ ] La impresora saca el papel inmediatamente sin pop-ups en la pantalla.
- [ ] Se imprimen correctamente el logo (si es posible), el encabezado, los ítems y el pie de página de la configuración comercial.

**DoD:**
- API de impresión ESC/POS (`internal/service/printer.go`).
- Interfaz en frontend para configurar la impresora.
- Test unitario simulando el generador de comandos térmicos.

---

## US-081 — Concurrencia y Estabilidad Multi-Caja LAN

**Como** dueño del restaurante o tienda,
**quiero** que varios empleados puedan cobrar y usar el sistema al mismo tiempo desde distintos celulares/tablets en la misma red WiFi,
**para** agilizar la atención sin que se pisen los datos o rompan las cuentas.

**Esfuerzo estimado:** 1 día. **Impacto:** Alto (Escalabilidad de operación real).

**Contexto técnico:**
- US-054 habilitó el acceso LAN. SQLite está configurado con `WAL` (Write-Ahead Logging) y un timeout de `5000ms`, lo cual soporta lectura/escritura concurrente decentemente.
- Se debe validar en un entorno real que no haya fallos de locking (`database is locked`) bajo carga simultánea.

**Regla de negocio:**
- Si dos cajas descuentan del mismo producto, la lógica debe manejar bien las transacciones (ya que el servicio lo hace en una Tx, esto debería estar cubierto, pero se debe validar el frontend).
- Reforzar el manejo de errores: Si el frontend recibe `HTTP 503` o `locked`, debe reintentar automáticamente (política de retry en `api.ts`).

**Acceptance Criteria:**
- [ ] Cargar el POS en una PC y en 2 celulares a través de la IP LAN.
- [ ] Simular dos cobros de diferentes mesas/clientes en el exacto mismo segundo.
- [ ] Validar que ninguna petición muera con un crash blanco; si hay colisión, un *toast* avisa que hubo un error o la app lo reintenta invisiblemente.
- [ ] Revisar el cierre de caja para confirmar que todos los montos de ambas sesiones coinciden perfectamente sin registros fantasma.

**DoD:**
- Añadir retry-logic en `frontend/lib/api.ts` para requests fallidos por locking temporal.
- Validaciones en consola de que no saltan panics en Go al manejar transacciones concurrentes.

---

## Criterio de éxito del sprint
- El binario Windows `.exe` puede lanzarse y usarse al 100% en localhost:8080 sin NodeJS.
- Ningún cajero puede robarse la clave del email.
- Las sesiones vencidas redirigen limpiamente.
- La base de datos es robusta contra eliminación accidental de datos conectados.
- La base de código de React está libre de warnings y deuda de linting.
- El POS es capaz de escupir tickets térmicos instantáneos sin intervención del navegador.
- El sistema no se cae ni se corrompe cuando hay 3 empleados cobrando a la vez.

## Notas técnicas para el dev
- Sobre US-076: Revisa si el struct `BusinessConfig` ya se usa directamente como request y response. Es recomendable separarlos o vaciar el campo en la respuesta y comprobar en el PUT si viene vacío para mantener el valor anterior haciendo un merge antes del `Upsert`.
