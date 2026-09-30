# SPRINT-13 — Mesas, Pedidos y Pantalla de Barra

**Objetivo:** Que una cafetería pueda abrir una mesa, tomarle el pedido con sus notas ("sin azúcar, leche deslactosada"), que el barista lo vea en una tablet en la barra en lugar de un post-it, y que el cobro salga por el flujo de venta que ya existe.

**Fecha inicio:** (por definir)

**Replanificado:** 2026-09-29, tras cerrar SPRINT-14. Cuatro cambios verificados contra el código: (1) la migración 15 ya está tomada por `assistant_queries` (US-061), el módulo usa la **16**; (2) US-054 depende de R-55, que hoy impide servir el POS a otro dispositivo, y lo absorbe; (3) referencias `archivo:línea` actualizadas; (4) se quitaron las promesas a "SPRINT-14" (Regla 3: el destino de lo recortado es `roadmap.md`).

**Referencias:** `project-docs/design/2026-09-20-mesas-pedidos-kds.md` (diseño aprobado 2026-09-20), `project-docs/sprints/SPRINT-12-WhatsApp-Dueno-Acceso-LAN.md`

**Deuda técnica entrante:**
- US-034 — Factus: validación en sandbox (abierta desde SPRINT-07, requiere acceso externo) — **sigue abierta**, sin acción en este sprint.
- SPRINT-05 — Mobile Responsive (⬜ en la tabla de progreso) — **sigue abierta**. Las pantallas nuevas de este sprint sí nacen responsive; el resto de la app no se toca.
- US-042c, US-043, US-044, US-045 (SPRINT-09) — **siguen abiertas**, no relacionadas con el módulo de restaurante.
- US-053 — Resumen de cierre de caja por correo (SPRINT-12) — **sigue abierta**. Hay trabajo sin commitear en `internal/mailer/`; no entra aquí para no mezclarlo con el módulo nuevo.
- US-054 — Acceso al POS por WiFi local (SPRINT-12) — **completa en código y tests (2026-09-30)**, junto con R-55. Falta solo la prueba manual con dos dispositivos reales (AC-16 / T-M), pendiente de Andrey.
  **Corrección verificada (2026-09-29):** SPRINT-12 decía que `lan_access_enabled` iba en la migración 14; la creó la migración 18 dentro de US-055 (`internal/database/migrate.go:548`).
- **R-55 — el binario no servía el frontend embebido** — **cerrado por US-054 (2026-09-30).**

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-055 | Migración 16, modo restaurante y gestión de mesas | [x] Completa en código, tests y verificación en navegador (2026-09-29) | 2026-09-29 |
| US-054 | Acceso al POS desde el celular/tablet por WiFi local (deuda SPRINT-12) + R-55 servir el frontend desde el binario | [x] Completa en código y tests; pendiente prueba manual con dos dispositivos (2026-09-30) | 2026-09-30 |
| US-056 | Pedido por mesa: abrir, agregar ítems con nota, anular, cancelar | [x] Completa en código y tests; pendiente verificación manual T-19 del PO (2026-09-30) | 2026-09-30 |
| US-057 | Cobro del pedido reusando el flujo de venta | [ ] Pendiente | — |
| US-058 | Pantalla de barra (KDS) con estados, tiempos y alerta sonora | [ ] Pendiente | — |

**Esfuerzo total estimado: 8.5 días** (US-054 sube de 0.5 a 1 por absorber R-55) — por encima del objetivo de ~7. Línea de corte si la fecha aprieta: US-058 vuelve a `roadmap.md` como ítem nuevo. Advertencia honesta: sin US-058 no hay demo que resuelva el dolor del cliente (los post-its siguen), solo mesas y cuenta abierta. Preferible recortar por otro lado antes que por el KDS.

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-055 | Trae la migración 18 completa (incluida la columna de US-054) y el toggle que esconde el módulo. Todo lo demás depende de sus tablas. |
| **2** | US-054 | Necesita la columna de la migración 18. Toca el bind del servidor (`cmd/pos/main.go:41`) y el servidor de estáticos (R-55) — el único cambio con radio de impacto real, se hace temprano para probarlo con calma y no al final del sprint. |
| **3** | US-056 | El núcleo. Sin pedidos no hay nada que cobrar ni que mostrar en barra. |
| **4** | US-057 | Depende de que exista un pedido con ítems. Incluye la extracción del panel de pago que US-056 no necesita todavía. |
| **5** | US-058 | Lee lo que US-056 escribe. Va al final porque es la única US que se puede recortar sin dejar el sprint a medias a nivel de datos. |

---

## [x] US-055 — Migración 16, modo restaurante y gestión de mesas (2026-09-29)

**Como** dueño de la cafetería,
**quiero** registrar mis mesas y activar el modo restaurante,
**para** que el POS muestre mesas y barra solo si mi negocio las usa.

**Esfuerzo estimado:** 1.5 días. **Impacto:** Alto. Es la base de todo el módulo y el interruptor que evita que un cliente de tienda vea funciones que no le sirven.

**Estado verificado:**
- No existe ninguna tabla ni columna del módulo: `grep -rn "dining_tables\|order_items\|restaurant_mode\|lan_access" internal/ frontend/` no devuelve nada.
- Última migración de `dev`: **17** (SPRINT-16/17 tomaron la 16, `jwt_secret`, y la 17, catálogo público). El módulo usa la **18** (renumerada al mergear US-055, 2026-09-30).
- `business_config` es tabla singleton (id = 1), se lee y escribe en `internal/repository/setup.go` con la struct `BusinessConfig` (línea 7). El `UPSERT` de `Upsert` (`internal/repository/setup.go:83-92`) lista columna por columna: hay que agregar las nuevas ahí o no se persisten.
- El sidebar filtra ítems por `visibleItems` (`frontend/components/Sidebar.tsx:143`; los ítems viven en `NAV_ITEMS`, línea 16, con la bandera `adminOnly`); ahí entra la condición de modo restaurante.
- La config del negocio ya llega al frontend por contexto: `frontend/lib/business-config-context.tsx`.

**Contexto técnico:**
Migración 16 crea `dining_tables`, `orders` y `order_items` (SQL exacto en el DoD) y agrega dos columnas a `business_config`: `restaurant_mode` y `lan_access_enabled` (esta última es de US-054). Nuevo `internal/repository/dining_table.go` + `internal/handler/tables.go`, registrados en el grupo autenticado de `Routes()` con el CRUD bajo el grupo admin, mismo patrón que `suppliers` (`internal/handler/handler.go:183-188`). `BusinessConfig` y su `UPDATE` en `setup.go` suman los dos campos nuevos. Pantalla de administración de mesas dentro de Ajustes (`frontend/app/settings/page.tsx`), no una ruta aparte.

**Regla de negocio:**
```
Estado de una mesa (derivado, nunca almacenado):
  libre        → no existe orders con status='open' y ese table_id
  ocupada      → existe orders open, y al menos un ítem en pending|preparing
  por_cobrar   → existe orders open, y todos sus ítems (no anulados) están en ready|delivered
  Un pedido abierto sin ítems cuenta como 'ocupada'.

Borrado de mesa:
  con pedido abierto        → 409, no se borra
  sin pedido abierto        → borrado lógico (active = 0), nunca DELETE físico,
                              porque orders históricas la referencian.
```

**Scope — qué NO entra:**
- Plano visual arrastrable ni zonas con coordenadas. `zone` es solo texto para agrupar.
- Capacidad/comensales: la columna `seats` se crea pero no se usa en ninguna pantalla todavía.

**Acceptance Criteria:**

**Ajustes → Mesas (admin):**
- [ ] Existe un toggle "Modo restaurante" que persiste en `business_config.restaurant_mode`
- [ ] Con el toggle apagado, la sección de mesas no se muestra y el sidebar no tiene "Mesas" ni "Barra"
- [ ] Con el toggle encendido, se puede crear una mesa con nombre y zona opcional
- [ ] Nombre duplicado → error visible, no se crea
- [ ] Nombre vacío → error de validación
- [ ] Se puede editar nombre, zona y orden de una mesa
- [ ] Se puede desactivar una mesa sin pedido abierto
- [ ] Intentar borrar una mesa con pedido abierto → mensaje "La mesa tiene un pedido abierto" y no se borra
- [ ] Estado vacío: sin mesas creadas, se ve un texto guía y el botón de crear
- [ ] Responsive: la tabla de mesas funciona en 320px

**Sidebar:**
- [ ] Con modo restaurante activo aparecen "Mesas" y "Barra"; un usuario `cashier` los ve, no solo admin

**DoD:**
- [ ] Migración 16 (`migration18` + `{version: 18, up: migration18}` en `migrate.go`; no editar la 15):
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
ALTER TABLE business_config ADD COLUMN lan_access_enabled INTEGER NOT NULL DEFAULT 0;
```
- [ ] `GET /api/tables` (autenticado) → `[{ id, name, zone, seats, sort_order, active, state, open_order_id, open_total, opened_minutes }]` donde `state` es `libre|ocupada|por_cobrar`
- [ ] `POST /api/tables` (admin) → `{ id, name, zone, seats, sort_order, active }`
- [ ] `PUT /api/tables/{id}` (admin) → mesa actualizada
- [ ] `DELETE /api/tables/{id}` (admin) → 409 si tiene pedido abierto; si no, `active = 0`
- [ ] `restaurant_mode` y `lan_access_enabled` se leen en `GET /api/setup` y se escriben en `PUT /api/setup` (agregar al `UPSERT` de `internal/repository/setup.go:83-92` y a `BusinessConfig`, línea 7; el `INSERT` lista columnas explícitamente)
- [ ] Test: mesa con pedido `open` → `DELETE` responde 409 y la mesa sigue `active = 1`
- [ ] Test: mesa con 2 ítems `ready` y 0 pendientes → `state = "por_cobrar"`; con 1 ítem `pending` → `state = "ocupada"`; sin pedido → `state = "libre"`
- [ ] Test: `PUT /api/setup` con `restaurant_mode: true` → `GET /api/setup` lo devuelve `true` (verifica que la columna entró al UPSERT)
- [ ] Build limpio (`go build ./...`, `pnpm build` en `frontend/`)

### Criterios verificados (US-055, cierre 2026-09-29)

QA: OK, 0 bloqueantes. `go build`, `go test ./...`, `pnpm build`, `pnpm test` (45 tests) pasan; `pnpm lint` queda en 41 problemas, igual a la base. Los checkboxes de arriba quedan como estaban en el sprint original; el estado real es el de esta lista.

**Migración y datos**
- AC-1: CUANDO la base arranca en versión 17, el sistema aplica la migración 18 (3 tablas, 5 índices, CHECK de `type` y `status`) y registra la versión 16. Test: `internal/database/migrate_test.go` `TestMigration16_T1_TablesIndexesAndChecks`.
- AC-2: CUANDO se aplica la migración 18, `business_config` gana `restaurant_mode` y `lan_access_enabled`, ambas `INTEGER NOT NULL DEFAULT 0`. Test: `TestMigration16_T2_BusinessConfigDefaults`.
- AC-3: CUANDO un admin envía `PUT /api/setup` con `restaurant_mode` y/o `lan_access_enabled`, se persisten y `GET /api/setup` los devuelve; `false` o ausentes devuelven `false`. Tests: `internal/handler/setup_test.go` `TestSetup_T3_PersistsAndClearsFlags` (true/true y luego false/false: cubre INSERT y DO UPDATE SET), `TestSetup_T4_AbsentLanFlagDefaultsFalse`.

**API de mesas** (todos en `internal/handler/tables_test.go`)
- AC-5: `POST /api/tables` con nombre → 201 con `active=true`, `zone=""`, `seats=0`, `sort_order=0`. `TestTables_T5_CreateDefaultsAndValues`.
- AC-6: nombre vacío o solo espacios → 400 y no se crea. `TestTables_T6_EmptyNameRejected`.
- AC-7: nombre duplicado (también con la mesa desactivada) → 409 en español sin texto SQL. `TestTables_T7_DuplicateNameConflict`.
- AC-8: `PUT /api/tables/{id}` → 200 con la mesa actualizada; nombre de otra mesa → 409; id inexistente → 404. `TestTables_T8_Update`.
- AC-9: `DELETE` sin pedido abierto → 200, `active = 0`, la fila sigue; pedidos `closed` y `cancelled` no bloquean. `TestTables_T9_DeleteIsLogical`.
- AC-10: `DELETE` con pedido `open` → 409 "La mesa tiene un pedido abierto" y la mesa sigue `active = 1`. `TestTables_T10_DeleteBlockedByOpenOrder`.
- AC-11 y AC-12: `GET /api/tables` devuelve activas ordenadas por `sort_order` y `name`; sin mesas devuelve `[]`; sin pedido → `libre`. `TestTables_T11_ListOrderAndEmpty`.
- AC-13: estado derivado y total: café $4.500 x2 + torta $8.000 `ready` → `por_cobrar`, `$17.000`; con un ítem `pending` → `ocupada`; sin ítems → `ocupada`, `$0`; torta `void` no suma (`$9.000`); `delivered`+`ready` → `por_cobrar`; `delivered`+`preparing` → `ocupada`; pedido creado hace 25 min → `opened_minutes` entre 24 y 25. `TestTables_T12_DerivedStates`, `TestTables_T13_VoidAndMixedStatuses`, `TestTables_T14_OpenedMinutesUTC`.
- AC-14: `cashier` recibe 403 en `POST`/`PUT`/`DELETE /api/tables` y 200 en `GET /api/tables` y `GET /api/setup`; admin crea con 201. `TestTables_T15_Permissions`.

**Interfaz**
- AC-16 y AC-17 (sidebar): con `restaurant_mode` apagado no hay "Mesas" ni "Barra" para nadie; encendido las ven admin y `cashier` (este último sin `/settings`). `frontend/lib/nav.test.ts`, tres casos de `describe('filterNavItems (T-16)')`.
- AC-21: build, tests y lint sin problemas nuevos (QA, T-18).

**Verificación en navegador (T-17), 2026-09-29.** La QA de la vuelta 1 no pudo hacer smoke (puerto 8080 fijo en `cmd/pos/main.go:41`, ocupado). El orquestador la hizo después con Edge + playwright-core (`e2e.js`), binario del worktree, DB nueva, 320 px. Halló un BLOQUEANTE que la QA no vio: `SettingsPage` consumía `useBusinessConfig()` fuera del provider (`frontend/components/AppLayout.tsx:56`) y recibía `defaultConfig`, así que la sección Mesas no se renderizaba. Arreglado en b861090: `SettingsContent` (`frontend/app/settings/page.tsx:181`) va dentro de `<AppLayout>` (`:580-581`). QA vuelta 2: OK, 1 MENOR (`settings/page.tsx:105-121`: la carga de mesas está duplicada entre `load()` y un `useEffect`; sin efecto funcional).
- Verificado en navegador: sin modo restaurante no hay sección Mesas (AC-16); el toggle persiste y aparece la sección (AC-15, AC-17); estado vacío con texto guía y botón (AC-19); crear mesa; nombre duplicado y nombre vacío muestran el error del servidor y no cambian la lista (AC-18); desactivar; sin scroll horizontal a 320 px (AC-20); `restaurant_mode` sigue en `true` tras guardar el formulario (AC-4, parte `restaurant_mode`); el sidebar muestra "Mesas" y "Barra" (AC-17).
- No cubierto en navegador: AC-4 con `lan_access_enabled=1` previo en la DB (queda cubierto por lectura, `settings/page.tsx:235-236`, y por `TestSetup_T3_PersistsAndClearsFlags` a nivel API); desactivar una mesa con pedido abierto (AC-18, requiere pedido sembrado por SQL; sí lo cubre `TestTables_T10_DeleteBlockedByOpenOrder`); revisión de tema claro/oscuro y dispositivo real, a cargo de Andrey si lo desea.
- Ruido observado, ya cubierto: cargar `/settings` u otras rutas directo devolvía el dashboard; es R-55 (`web/static.go`, planeado en US-054). Patrón repetido en `CustomerDetailClient` registrado como R-61.

---

## [x] US-054 — Acceso al POS desde el celular/tablet por WiFi local (K1, deuda SPRINT-12) (2026-09-30)

**Como** dueño,
**quiero** entrar al POS desde el celular o una tablet conectada a mi WiFi,
**para** que el barista use el KDS en una tablet en la barra y yo vea las ventas sin estar en el computador.

**Esfuerzo estimado:** 1 día (0.5 del bind + 0.5 de R-55). **Impacto:** Alto en este sprint — es dependencia dura del KDS (US-058), no una comodidad.

**Estado verificado (al cerrar, 2026-09-30):**
- `lan_access_enabled` lo creó la migración 18 de US-055 (`internal/database/migrate.go:548`); esta US no agregó migración.
- El bind sale de `listenAddr(lan)` (`cmd/pos/main.go:72`), con `lan` leído de `business_config` al arrancar (`main.go:55-57`); el navegador se abre siempre en `browserURL()` (`main.go:80`).
- R-55 cerrado: `web/static.go` resuelve archivo -> `<ruta>.html` -> `customers/_.html`/`catalogo/_.html` -> 404 para `/_next/` y assets -> `index.html`; `frontend/lib/api.ts:3` usa `/api` relativo por defecto. CORS `:3000` (`internal/middleware/middleware.go:94-97`) sin cambio: solo aplica al dev.
- Se agregó `web/out/.gitkeep` para que `go build ./...` compile en un checkout limpio.

**Contexto técnico:**
`listenAddr(lanEnabled bool) string` como función exportada y pura en `cmd/pos/`, no inline en `main()`, específicamente para poder testearla sin levantar servidor. Devuelve `127.0.0.1:8080` con el toggle apagado y `0.0.0.0:8080` con el toggle encendido. `main()` lee `business_config.lan_access_enabled` al arrancar. En Ajustes, junto al toggle, se muestra la IP local detectada y la URL que hay que escribir en la tablet.

**Regla de negocio:**
```
lan_access_enabled = 0  → bind 127.0.0.1:8080  (solo esta máquina)
lan_access_enabled = 1  → bind 0.0.0.0:8080    (toda la red local)
El cambio aplica al reiniciar pos.exe. No es un bug: net/http bindea el
socket una sola vez al arrancar.
```

**Scope — qué NO entra:**
- HTTPS / certificados. Es red local.
- Aplicar el toggle en caliente sin reiniciar.

**Acceptance Criteria:**

**Ajustes:**
- [x] Toggle "Permitir acceso desde otros dispositivos de mi red WiFi", persiste en `business_config.lan_access_enabled`
- [x] Al activarlo se muestra la URL con la IP local detectada (ej. `http://192.168.1.15:8080`) y el aviso de que hay que reiniciar el programa
- [x] Si no se detecta IP local, se muestra un texto de ayuda en vez de una URL rota

**Acceso:**
- [x] Con el toggle apagado, `pos.exe` solo responde en `localhost`
- [ ] Con el toggle encendido y tras reiniciar, un celular en la misma WiFi abre el POS y puede iniciar sesión (AC-16, pendiente de Andrey)
- [ ] Responsive: el login y el POS abren usables en un celular de 320px (pendiente de verificar en celular real, con la prueba manual)

**DoD:**
- [x] Usa la columna `lan_access_enabled` de la migración 18 (US-055); esta US no crea migración
- [x] `listenAddr` exportada y pura en `cmd/pos/`
- [x] Test: `listenAddr(false) == "127.0.0.1:8080"` y `listenAddr(true) == "0.0.0.0:8080"`
- [x] R-55: `FileServer` quita la barra inicial y sirve `<ruta>.html` para rutas exportadas; `curl` de un chunk `/_next/static/chunks/*.js` devuelve `text/javascript` (el tipo que entrega `net/http`), no `text/html`
- [x] R-55: test de `FileServer` con un FS en memoria: `/`, `/login` (→ `login.html`), un `.js` existente y una ruta inexistente (→ `index.html`)
- [x] R-55: `api.ts` usa `/api` relativo por defecto; el dev con `:3000` sigue funcionando vía `NEXT_PUBLIC_API_URL`, documentado en `INSTALL.md`/`CLAUDE.md` si cambia
- [x] Test: detección de IP local devuelve una IPv4 privada (10.x / 172.16-31.x / 192.168.x) o cadena vacía, nunca `127.0.0.1` como sugerencia al usuario
- [ ] Test manual con dos dispositivos reales — **pendiente para Andrey**, no automatizable en este entorno
- [x] Build limpio (`go build ./...`, `pnpm build`)

---

**Criterios verificados (cierre 2026-09-30, QA vuelta 1 OK: 0 BLOQUEANTE, 3 MENOR):**
- AC-1/AC-2: MIENTRAS `lan_access_enabled` = 0 (o la config no se pueda leer) el sistema DEBE escuchar en `127.0.0.1:8080`; con 1, en `0.0.0.0:8080`. Test: `cmd/pos/main_test.go` `TestListenAddr_T1`.
- AC-3: el navegador se abre siempre en `http://localhost:8080`. Test: `cmd/pos/main_test.go` `TestBrowserURL_T2`.
- AC-4: el toggle persiste en `business_config.lan_access_enabled` y aplica al reiniciar. Test: `internal/handler/setup_test.go` `TestSetup_T4_AbsentLanFlagDefaultsFalse` (más el PUT/GET de US-055 en el mismo archivo).
- AC-6/AC-7/AC-8: la IP sugerida es solo IPv4 privada (10/8, 172.16/12, 192.168/16), nunca loopback ni 169.254.x; sin IP, Ajustes muestra ayuda. `GET /api/setup/lan-info` solo admin. Tests: `internal/netutil/lan_test.go` `TestPickPrivateIPv4_T4`; `internal/handler/setup_test.go` `TestSetup_US054_T5_LanInfo`.
- AC-9..AC-14: `FileServer` sirve `/` -> `index.html`; `/login` -> `login.html` sin redirigir (incluso con directorio homónimo); `/customers/<id>` y `/catalogo/<token>` -> `_.html`; asset inexistente o `/_next/*` -> 404; ruta desconocida -> `index.html`. Tests: `web/static_test.go` `TestFileServer_T6_RootServesIndex`, `_T7_HtmlExtensionWithoutRedirect`, `_T8_AssetContentType`, `_T9_DynamicRoutes`, `_T10_FallbackAndAsset404`.
- AC-15: sin `NEXT_PUBLIC_API_URL` la UI usa `/api`; con la variable, su valor. Test: `frontend/lib/api.test.ts` (`uses relative /api when NEXT_PUBLIC_API_URL is undefined`, `uses NEXT_PUBLIC_API_URL when defined`).
- AC-5/AC-17: toggle visible en Ajustes con URL o ayuda, aviso de reinicio y aviso de cambiar `admin/admin`. Solo revisado por lectura de `frontend/app/settings/page.tsx`; T-12 no se verificó en navegador.
- AC-16 / T-M: un celular real en la misma WiFi abre el POS e inicia sesión (320px). **Pendiente de Andrey**, no automatizable.
- MENOR aceptados sin ítem: import de `internal/netutil` dentro del grupo stdlib (`internal/handler/handler.go:6`) y doble línea en blanco antes de `getLanInfo` (`internal/handler/setup.go:64-65`); son de estilo. MENOR 3 (lint 42 vs 41): no es regresión de esta US, ver bitácora de `roadmap.md`.

---

## US-056 — Pedido por mesa: abrir, agregar ítems con nota, anular, cancelar

**Como** cajero de la cafetería,
**quiero** abrir una mesa y agregarle ítems con sus indicaciones,
**para** dejar de escribir el pedido en un post-it y que la cuenta del cliente quede registrada.

**Esfuerzo estimado:** 2.5 días. **Impacto:** Alto. Es el núcleo del módulo.

**Estado verificado:**
- El POS actual (`frontend/app/pos/page.tsx`, 1624 líneas) guarda carritos aparcados en `localStorage` (`PARKED_CARTS_KEY`, línea 69). Es una cuenta abierta local a ese navegador: el barista en su tablet nunca la ve. Con `restaurant_mode` activo ese botón se oculta.
- Componentes reutilizables ya existentes en esa página: `ProductCard` (línea 152) y `CartItemRow` (línea 200). Se extraen o se replican deliberadamente; ver Notas técnicas.
- `repository.InsertAuditLog(ctx, execer, userID, action, entity, entityID, details)` (`internal/repository/audit.go:21`) recibe un `execer`, así que funciona dentro de una transacción.

**Contexto técnico:**
Nuevo `internal/repository/order.go` (+ `order_test.go`), `internal/service/order.go`, `internal/handler/order.go`, rutas en el grupo autenticado. Frontend: `frontend/app/mesas/page.tsx` (grid de mesas) y `frontend/app/mesas/[orderId]/page.tsx` (pedido). Ningún cambio en ventas, caja, reportes ni facturación: este sprint no escribe en `sales` hasta US-057.

**Regla de negocio:**
```
Totales del pedido (sin impuestos ni descuento global en v1):
  linea.subtotal = quantity * unit_price
  pedido.total   = suma de subtotales de ítems con status != 'void'

unit_price se copia del producto al agregar el ítem (snapshot).
Cambiar el precio del producto después NO altera pedidos ya abiertos.

Anulación de ítem:
  status = 'void' (nunca DELETE físico) + registro en audit_log
  action='order_item_void', entity='order_item', entity_id=<item id>

Cancelación de pedido:
  solo si status='open'. Pasa a 'cancelled' + audit_log
  action='order_cancel', entity='order', entity_id=<order id>, details=<motivo>
  Un pedido 'closed' no se puede cancelar (ya es una venta: se anula la venta).

Toda escritura de ítem toca orders.updated_at (misma transacción).
```

**Scope — qué NO entra:**
- Pedido para llevar con turno (R-01 en `roadmap.md`). En este sprint `type` siempre es `'table'`; la columna ya acepta `'takeaway'`.
- Modificadores con precio. La variación va en `note`, texto libre.
- Mover ítems entre mesas, dividir cuenta, propina.

**Acceptance Criteria:**

**`/mesas` (grid):**
- [ ] Muestra todas las mesas activas con su estado por color: libre / ocupada / por cobrar
- [ ] Una mesa ocupada muestra el total acumulado y los minutos desde que se abrió
- [ ] Tocar una mesa libre abre un pedido nuevo y navega a su pantalla
- [ ] Tocar una mesa ocupada navega al pedido existente (no crea uno segundo)
- [ ] Estado vacío: sin mesas creadas, enlace a Ajustes → Mesas
- [ ] Responsive: el grid funciona en 320px (una columna)

**`/mesas/[orderId]` (pedido):**
- [ ] Buscador y grid de productos; tocar un producto lo agrega al pedido
- [ ] Cada línea permite editar cantidad y escribir una nota de texto libre
- [ ] El ítem se guarda en el servidor al agregarlo, sin botón "enviar a cocina"
- [ ] Cada línea muestra su estado (pendiente / preparando / listo)
- [ ] Anular una línea pide confirmación y la deja tachada, no la borra de la vista
- [ ] Cancelar el pedido pide motivo y libera la mesa
- [ ] El total se recalcula al instante con cada cambio
- [ ] Error de red al agregar un ítem → mensaje visible y la línea no queda fantasma en la UI
- [ ] Responsive: en 320px el catálogo y el pedido se alternan por pestañas, como ya hace el POS (`mobileTab`)

**POS existente:**
- [ ] Con `restaurant_mode` activo, el botón de carritos guardados no se muestra
- [ ] Con `restaurant_mode` apagado, el POS se comporta exactamente como hoy

**DoD:**
- [ ] Sin migración nueva (usa las tablas de la migración 18)
- [ ] `POST /api/orders` (autenticado) → `{ id, type, table_id, status, created_at }` · 409 si la mesa ya tiene pedido abierto
- [ ] `GET /api/orders?status=open` (autenticado) → `[{ id, table_id, table_name, total, items_pending, opened_minutes }]`
- [ ] `GET /api/orders/{id}` (autenticado) → `{ id, type, table_id, status, total, items: [{ id, product_id, product_name, quantity, unit_price, subtotal, note, status }] }`
- [ ] `POST /api/orders/{id}/items` (autenticado) → ítem creado · 409 si el pedido no está `open`
- [ ] `PATCH /api/orders/{id}/items/{itemId}` (autenticado) → acepta `quantity`, `note`, `status`
- [ ] `DELETE /api/orders/{id}/items/{itemId}` (autenticado) → `status='void'` + fila en `audit_log`
- [ ] `POST /api/orders/{id}/cancel` (autenticado) → `{reason}` obligatorio, pedido `cancelled` + fila en `audit_log`
- [ ] Test: pedido con 2 ítems (café $4.500 x2 y torta $8.000 x1) → total 17.000; se anula la torta → total 9.000 y el ítem sigue en la tabla con `status='void'`
- [ ] Test: abrir pedido en una mesa que ya tiene uno abierto → 409 y sigue habiendo exactamente 1 pedido abierto para esa mesa
- [ ] Test: cancelar pedido con ítems → `orders.status='cancelled'`, se inserta 1 fila en `audit_log` con el `user_id` correcto, y la mesa vuelve a `state='libre'`
- [ ] Test: agregar ítem a un pedido `closed` → 409
- [ ] Build limpio (`go build ./...`, `pnpm build`)

**Estado: [x] completa en código y tests (2026-09-30); verificación manual T-19 pendiente del PO.** Los checkboxes de UI de arriba (`/mesas`, `/mesas/[orderId]`, POS) quedan sin marcar hasta esa prueba; build, tests backend y vitest pasaron en QA.

**Criterios verificados (cierre 2026-09-30, QA vuelta 1 OK: 0 BLOQUEANTE, 4 MENOR):**
- AC-1/AC-3: CUANDO un autenticado hace `POST /api/orders` con `table_id` de mesa activa sin pedido abierto, el sistema DEBE crear un pedido `type='table'`, `open`, con el `user_id` del token (201); `table_id` inexistente o desactivada → 404, falta/inválido o `type` ≠ `table` → 400, sin crear fila. Tests: `internal/handler/order_test.go` `TestOrder_T11_OpenValidation`, `TestOrder_T2_SecondOpenConflicts`.
- AC-2: SI la mesa ya tiene pedido `open`, ENTONCES 409, incluso en concurrencia (INSERT condicional atómico, sin migración); `SQLITE_BUSY` → 409 "reintenta", nunca 500. Tests: `internal/repository/order_test.go` `TestOrder_T3_ConcurrentOpenSingleWinner`, `TestOrder_T15_InactiveTableAndOpenThenDeactivate`; `internal/handler/order_test.go` `TestOrder_T22_BusyMapsTo409`.
- AC-4/AC-5/AC-6/AC-7/AC-7b: agregar ítem con snapshot de precio; `quantity` entero > 0 (0, negativos, `0.5`, `2.0` → 400); solo con pedido `open` (si no 409); `PATCH` de `quantity`/`note`/`status` (`pending|preparing|ready|delivered`, `void` → 400); editable en `ready`/`delivered`. Ej.: café $4.500 x2 → tras `PATCH quantity=3` subtotal $13.500. Tests: `TestOrder_T1_AddItemsTotalAndVoid`, `T6_ClosedOrderRejectsItemChanges`, `T7_ValidationOfQuantityStatusProduct`, `T8_PriceSnapshot`, `T10_PatchQuantityAndNote`, `T20_EditReadyAndDeliveredItems`.
- AC-7c: `ready_at` se fija al pasar a `ready`, no se sobrescribe al reenviar y vuelve a NULL al salir de `ready`. Tests: `TestOrder_T21_ReadyAtSetAndPreserved`, `T21b_ReadyAtClearedWhenLeavingReady`.
- AC-8: `DELETE` de ítem → `status='void'` + `audit_log order_item_void` con `details` JSON de 9 claves (`status` = estado previo), toca `updated_at`; ya anulado → 409. Pedido con café x2 ($9.000) y torta $8.000: total $17.000, tras anular la torta $9.000. Tests: `TestOrder_T1_AddItemsTotalAndVoid`, `T1b_VoidDetailsStatusIsPreVoidState`, `T9_TouchesUpdatedAt`.
- AC-9/AC-10/AC-11: `POST /cancel` con motivo no vacío → `cancelled`, `closed_at`, `audit_log order_cancel`, ítems conservados, mesa `libre`; motivo vacío → 400; no `open` → 409. Tests: `TestOrder_T4_CancelOrder`, `T5_CancelRejections`.
- AC-12/AC-13: `GET /orders/{id}` con total sin `void` e `items` incluyendo `void`; `GET /orders?status=open` con `items_pending` y `opened_minutes`, `[]` si vacío, `status` inválido → 400. Tests: `TestOrder_T1_AddItemsTotalAndVoid`, `T12_ListOpen`.
- AC-14: cajero y admin operan pedidos; sin token 401. Test: `TestOrder_T13_CashierAllowedAnonymousRejected`.
- AC-15 (R-59): `Deactivate` atómico, 409 con pedido abierto. Test: `internal/repository/dining_table_test.go` `TestDiningTable_T14_DeactivateWithOpenOrder`; regresión de US-055 en `internal/handler/tables_test.go`.
- AC-16/AC-24: CORS con `PATCH` y `api.patch`. Tests: `internal/middleware/middleware_test.go` `TestCORS_T16_AllowsPatch`; `frontend/lib/api.test.ts` (`T-18: sends PATCH with JSON body`).
- AC-22: `/mesas/12` se sirve con `mesas/_.html`. Test: `web/static_test.go` `TestFileServer_T9_DynamicRoutes`.
- AC-17..AC-21, AC-23 (UI): verificados solo por `pnpm build` (genera `/mesas` y `/mesas/_`) y lectura de código; `showParked = !restaurant_mode` en `frontend/app/pos/page.tsx:622,1030,1141,1155`. **T-19 (manual a 320px y 1280px) PENDIENTE del PO**: grid por estado, tocar ocupada no duplica, backend apagado sin línea fantasma, anulación tachada, cancelación con motivo, y con `restaurant_mode`=1 sin chips de espera en el POS.
- Fuera de alcance y registrado: refresco automático (R-63) y 4 MENOR de QA (R-64, R-65, R-66; ver `roadmap.md`). `TestDashboardToday_T20_Golden` falla en `dev` por fecha (preexistente, R-67).

---

## US-057 — Cobro del pedido reusando el flujo de venta

**Como** cajero,
**quiero** cobrar la mesa con los mismos medios de pago de siempre,
**para** que la venta quede en caja, en los reportes y con su ticket, sin registrar nada dos veces.

**Esfuerzo estimado:** 1.5 días. **Impacto:** Alto. Es lo que conecta el módulo nuevo con el negocio real.

**Estado verificado:**
- `createSale` (`internal/handler/sale.go:15-48`) calcula subtotales de ítems, arma `model.Sale` y llama `h.sales.Create`; después dispara la facturación electrónica en goroutine (`internal/handler/sale.go:44-46`).
- `SaleService.Create` (`internal/service/sale.go:39`) es el punto único de negocio; el repositorio (`internal/repository/sale.go:37`) hace en una sola transacción: venta + `sale_payments` + `sale_items` + stock + movimiento de caja.
- El panel de pago vive dentro de `POSPage` con su propio estado: `payMethod`, `extraPayments`, `firstAmountInput`, `cashReceived` (`frontend/app/pos/page.tsx:652-658`), y la lógica de pagos mixtos en `paymentLines` / `addPaymentLine` (líneas 899-916). Los cálculos ya están fuera, en `frontend/lib/saleCalc.ts`, con tests.

**Contexto técnico:**
El handler de checkout construye un `model.Sale` desde los ítems no anulados del pedido y llama el mismo `SaleService.Create`; no se duplica una línea de lógica de cobro. Cerrar el pedido (`status='closed'`, `sale_id`, `closed_at`) ocurre en la misma transacción que crea la venta, o el pedido queda abierto. Frontend: se extrae el panel de pago a `frontend/components/SalePaymentPanel.tsx` y lo consumen el POS y la pantalla de pedido. Los cálculos siguen viniendo de `lib/saleCalc.ts` — la extracción mueve UI y estado, no aritmética.

**Regla de negocio:**
```
Al cobrar:
  items considerados = order_items con status != 'void'
  sale.subtotal      = suma de (quantity * unit_price)
  sale.total         = subtotal - descuento + impuesto   (igual que hoy en createSale)
  cada order_item    → un sale_item con el MISMO unit_price (snapshot del pedido)

Atomicidad: venta + cierre del pedido en una sola transacción.
  Si la venta falla → orders.status sigue 'open', no hay venta parcial.

Pedido sin ítems válidos (vacío o todo anulado) → 400, no se crea venta.
El stock se descuenta aquí, no al tomar el pedido.
```

**Scope — qué NO entra:**
- Propina y división de cuenta (R-12 en `roadmap.md`).
- Descuento global por pedido: se usa el mismo campo de descuento de la venta, sin UI nueva por línea.

**Acceptance Criteria:**

**Cobro desde el pedido:**
- [ ] Botón "Cobrar" abre el mismo panel de pago del POS (efectivo, transferencia, tarjeta, crédito y mixto)
- [ ] Efectivo muestra cambio con la misma lógica de hoy
- [ ] Crédito exige cliente y respeta el límite de crédito igual que el POS
- [ ] Al confirmar se muestra el ticket con el mismo componente de siempre
- [ ] Terminado el cobro, la mesa queda libre en `/mesas`
- [ ] Cobrar un pedido sin ítems → botón deshabilitado y, si se fuerza por API, 400
- [ ] Error al cobrar → mensaje visible, el pedido sigue abierto y no se creó venta

**POS existente (regresión):**
- [ ] El POS cobra exactamente igual que antes tras la extracción del panel: efectivo, mixto con 2 y 3 líneas, crédito
- [ ] Responsive: el panel extraído sigue funcionando en 320px en ambas pantallas

**Cierre de caja:**
- [ ] Cerrar caja con pedidos abiertos muestra el aviso "Hay N pedidos sin cobrar" y **permite** cerrar igual

**DoD:**
- [ ] Sin migración nueva
- [ ] `POST /api/orders/{id}/checkout` (autenticado) → mismo body que `POST /api/sales` (`{ payments: [{method, amount, reference}], customer_id?, discount?, tax?, buyer_email? }`) → `{ order_id, sale_id, total }`
- [ ] El checkout llama `SaleService.Create`; no se reimplementa stock, caja ni pagos
- [ ] `frontend/components/SalePaymentPanel.tsx` existe y es consumido por `app/pos/page.tsx` y `app/mesas/[orderId]/page.tsx`
- [ ] Test: pedido con café $4.500 x2 + torta $8.000 → checkout con efectivo $20.000 → venta total 17.000, cambio 3.000, `orders.status='closed'` con `sale_id` poblado, 2 filas en `sale_items`
- [ ] Test: pedido con 3 ítems, uno anulado → la venta tiene 2 `sale_items` y el total excluye el anulado
- [ ] Test: checkout de pedido sin ítems válidos → 400 y `COUNT(*) FROM sales` no cambia
- [ ] Test: checkout con pago mixto $10.000 efectivo + $7.000 transferencia sobre total 17.000 → 2 filas en `sale_payments`, `payment_method='mixed'`, saldo 0
- [ ] Build limpio (`go build ./...`, `pnpm build`, `pnpm test` para los tests de `saleCalc`)

---

## US-058 — Pantalla de barra (KDS) con estados, tiempos y alerta sonora

**Como** barista,
**quiero** ver en una pantalla lo que tengo que preparar y marcarlo listo,
**para** dejar de trabajar con post-its y que nadie tenga que preguntarme si ya salió.

**Esfuerzo estimado:** 2 días. **Impacto:** Alto. Es la US que resuelve el dolor que el cliente describió.

**Estado verificado:**
- No existe ninguna pantalla ni endpoint de cocina/barra en el repo.
- No hay SSE ni WebSocket en el backend hoy: todo es request/response con Chi (`internal/handler/handler.go`).
- `frontend/hooks/useIsMobile.ts` es el único hook propio; el polling se implementa con `useEffect` + `setInterval`, sin librería nueva.

**Contexto técnico:**
Nuevo `GET /api/kds/orders` que devuelve los ítems en `pending|preparing|ready` de pedidos `open` **y** `closed` (el prepago se cobra antes de prepararse), con `server_time` para calcular los minutos sin depender del reloj de la tablet. Nueva página `frontend/app/barra/page.tsx` con polling de 3 s. El avance de estado reusa el `PATCH /api/orders/{id}/items/{itemId}` de US-056; esta US no crea endpoints de escritura.

**Regla de negocio:**
```
Columnas: pending → preparing → ready. Un toque avanza al siguiente estado.
'delivered' se marca desde el pedido (US-056), no desde la barra.
Los ítems 'void' nunca aparecen en la barra; si se anula uno ya mostrado,
desaparece en el siguiente poll.

minutos = (server_time - item.created_at) / 60
  < 5 min   → normal
  5–10 min  → advertencia (ámbar)
  > 10 min  → alerta (rojo)

Sonido: suena una vez por poll si aparecieron ids de ítem no vistos antes.
Nunca suena en el primer poll tras abrir la pantalla (si no, suena con
todo el backlog del día).
```

**Scope — qué NO entra:**
- SSE / WebSocket. El polling de 3 s se cambia después sin tocar el contrato de la API.
- Impresión de comanda en térmica.
- Vistas separadas por estación (barra / cocina).

**Acceptance Criteria:**

**`/barra`:**
- [ ] Tres columnas: Pendiente, Preparando, Listo
- [ ] Cada tarjeta muestra mesa (o número de turno cuando exista), los ítems con cantidad, y la nota resaltada visualmente
- [ ] Cada tarjeta muestra los minutos transcurridos, con color según los umbrales de la regla
- [ ] Un toque en el ítem lo avanza de estado y el cambio se ve en menos de 3 s en otra pantalla abierta
- [ ] La lista se refresca sola cada 3 s sin recargar la página ni perder el scroll
- [ ] Botón "Activar sonido" al entrar; después de tocarlo, un ítem nuevo dispara el sonido
- [ ] No suena en el primer refresco al abrir la pantalla
- [ ] Un ítem anulado desde el pedido desaparece de la barra en el siguiente refresco
- [ ] Estado vacío: sin pedidos pendientes, mensaje grande "Todo al día"
- [ ] Si el servidor no responde, se muestra "Sin conexión" y al volver la red se reanuda sola sin recargar
- [ ] Legibilidad: tipografía y botones usables en una tablet a un metro de distancia
- [ ] Responsive: en 320px las tres columnas se apilan

**DoD:**
- [ ] Sin migración nueva
- [ ] `GET /api/kds/orders` (autenticado) → `{ server_time, orders: [{ order_id, type, table_name, turn_number, created_at, items: [{ id, product_name, quantity, note, status, created_at }] }] }`
- [ ] Devuelve ítems de pedidos `open` y `closed`; excluye `void`, `delivered` y pedidos `cancelled`
- [ ] El avance de estado usa el `PATCH` de US-056, sin endpoint nuevo
- [ ] Test: pedido **cobrado** (`status='closed'`) con un ítem `pending` → aparece en la respuesta del KDS (regresión del caso prepago)
- [ ] Test: pedido con 3 ítems (1 pending, 1 ready, 1 void) → la respuesta trae 2 ítems y ninguno es el anulado
- [ ] Test: `PATCH` de ítem a `ready` → `ready_at` queda poblado y `orders.updated_at` se actualiza
- [ ] Build limpio (`go build ./...`, `pnpm build`)

---

## Criterio de éxito del sprint

- El dueño activa el modo restaurante, crea 8 mesas y permite el acceso por WiFi en menos de 5 minutos.
- El cajero abre la mesa 3, agrega dos capuchinos con la nota "leche deslactosada" y una torta, y el barista los ve en la tablet en menos de 3 segundos sin que nadie escriba un papel.
- El barista marca listo un capuchino y el cajero lo ve en la mesa; la torta sigue pendiente en la barra.
- El cajero cobra la mesa con pago mixto, sale el ticket de siempre, la venta aparece en el cierre de caja del día y la mesa queda libre.

---

## Notas técnicas para el dev

- **US-056 y US-057 comparten el catálogo de productos** con `app/pos/page.tsx`: `ProductCard` (línea 152) y el buscador. Extraer `ProductCard` a `components/` antes de copiarlo; ya va a ser su tercer uso.
- **La extracción de `SalePaymentPanel` (US-057) es la única refactorización autorizada del sprint.** Hacerla en un commit aparte del checkout, y probar el POS antes de seguir: si se mezcla con la lógica nueva y algo se rompe, no se sabe cuál de las dos fue.
- `repository.InsertAuditLog` acepta un `execer` (`internal/repository/audit.go:21`), así que la anulación de ítem y la cancelación de pedido escriben la auditoría **dentro** de la misma transacción. Fuera de ella, un rollback deja una auditoría de algo que nunca pasó.
- `turn_number` no se usa en este sprint pero la columna se crea en la migración 18: SQLite no permite agregar columnas con `UNIQUE` después, y agregarla ahora evita otra migración solo para eso.
- Fechas en SQLite: usar `date('now','localtime')` para cualquier lógica de "hoy". `CURRENT_TIMESTAMP` guarda UTC y en Colombia (UTC-5) parte el día a las 7 p.m.
- El `UPSERT` de `internal/repository/setup.go:83-92` lista las columnas una por una. Cada columna nueva de `business_config` hay que agregarla ahí o se guarda en silencio sin efecto — es un error fácil de no notar hasta la demo.
- `orders.updated_at` no se actualiza solo: tocarlo explícitamente en toda escritura de ítem, o el KDS mostrará datos viejos.

---

## Roadmap

Lo que queda por construir vive en [roadmap.md](roadmap.md), que es la única fuente. Este documento no reserva números de sprint futuros.

> Antes había aquí una tabla de roadmap propia. Se consolidó el 2026-09-21: las copias por sprint divergieron entre sí y llegaron a reclamar el mismo número de sprint para features distintas.
