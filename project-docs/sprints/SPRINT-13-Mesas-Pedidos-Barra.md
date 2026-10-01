# SPRINT-13 â€” Mesas, Pedidos y Pantalla de Barra

**Objetivo:** Que una cafeterÃ­a pueda abrir una mesa, tomarle el pedido con sus notas ("sin azÃºcar, leche deslactosada"), que el barista lo vea en una tablet en la barra en lugar de un post-it, y que el cobro salga por el flujo de venta que ya existe.

**Fecha inicio:** (por definir)

**Replanificado:** 2026-09-29, tras cerrar SPRINT-14. Cuatro cambios verificados contra el cÃ³digo: (1) la migraciÃ³n 15 ya estÃ¡ tomada por `assistant_queries` (US-061), el mÃ³dulo usa la **16**; (2) US-054 depende de R-55, que hoy impide servir el POS a otro dispositivo, y lo absorbe; (3) referencias `archivo:lÃ­nea` actualizadas; (4) se quitaron las promesas a "SPRINT-14" (Regla 3: el destino de lo recortado es `roadmap.md`).

**Referencias:** `project-docs/design/2026-09-20-mesas-pedidos-kds.md` (diseÃ±o aprobado 2026-09-20), `project-docs/sprints/SPRINT-12-WhatsApp-Dueno-Acceso-LAN.md`

**Deuda tÃ©cnica entrante:**
- US-034 â€” Factus: validaciÃ³n en sandbox (abierta desde SPRINT-07, requiere acceso externo) â€” **sigue abierta**, sin acciÃ³n en este sprint.
- SPRINT-05 â€” Mobile Responsive (â¬œ en la tabla de progreso) â€” **sigue abierta**. Las pantallas nuevas de este sprint sÃ­ nacen responsive; el resto de la app no se toca.
- US-042c, US-043, US-044, US-045 (SPRINT-09) â€” **siguen abiertas**, no relacionadas con el mÃ³dulo de restaurante.
- US-053 â€” Resumen de cierre de caja por correo (SPRINT-12) â€” **sigue abierta**. Hay trabajo sin commitear en `internal/mailer/`; no entra aquÃ­ para no mezclarlo con el mÃ³dulo nuevo.
- US-054 â€” Acceso al POS por WiFi local (SPRINT-12) â€” **completa en cÃ³digo y tests (2026-09-30)**, junto con R-55. Falta solo la prueba manual con dos dispositivos reales (AC-16 / T-M), pendiente de Andrey.
  **CorrecciÃ³n verificada (2026-09-29):** SPRINT-12 decÃ­a que `lan_access_enabled` iba en la migraciÃ³n 14; la creÃ³ la migraciÃ³n 18 dentro de US-055 (`internal/database/migrate.go:548`).
- **R-55 â€” el binario no servÃ­a el frontend embebido** â€” **cerrado por US-054 (2026-09-30).**

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-055 | MigraciÃ³n 16, modo restaurante y gestiÃ³n de mesas | [x] Completa en cÃ³digo, tests y verificaciÃ³n en navegador (2026-09-29) | 2026-09-29 |
| US-054 | Acceso al POS desde el celular/tablet por WiFi local (deuda SPRINT-12) + R-55 servir el frontend desde el binario | [x] Completa en cÃ³digo y tests; pendiente prueba manual con dos dispositivos (2026-09-30) | 2026-09-30 |
| US-056 | Pedido por mesa: abrir, agregar Ã­tems con nota, anular, cancelar | [x] Completa en cÃ³digo y tests; pendiente verificaciÃ³n manual T-19 del PO (2026-09-30) | 2026-09-30 |
| US-057 | Cobro del pedido reusando el flujo de venta | [x] Completa en código y tests (2026-10-01) | 2026-10-01 |
| US-058 | Pantalla de barra (KDS) con estados, tiempos y alerta sonora | [x] Completa en código y tests (2026-10-01) | 2026-10-01 |

**Esfuerzo total estimado: 8.5 dÃ­as** (US-054 sube de 0.5 a 1 por absorber R-55) â€” por encima del objetivo de ~7. LÃ­nea de corte si la fecha aprieta: US-058 vuelve a `roadmap.md` como Ã­tem nuevo. Advertencia honesta: sin US-058 no hay demo que resuelva el dolor del cliente (los post-its siguen), solo mesas y cuenta abierta. Preferible recortar por otro lado antes que por el KDS.

---

## Orden de implementaciÃ³n recomendado

| Orden | US | RazÃ³n |
|-------|----|-------|
| **1** | US-055 | Trae la migraciÃ³n 18 completa (incluida la columna de US-054) y el toggle que esconde el mÃ³dulo. Todo lo demÃ¡s depende de sus tablas. |
| **2** | US-054 | Necesita la columna de la migraciÃ³n 18. Toca el bind del servidor (`cmd/pos/main.go:41`) y el servidor de estÃ¡ticos (R-55) â€” el Ãºnico cambio con radio de impacto real, se hace temprano para probarlo con calma y no al final del sprint. |
| **3** | US-056 | El nÃºcleo. Sin pedidos no hay nada que cobrar ni que mostrar en barra. |
| **4** | US-057 | Depende de que exista un pedido con Ã­tems. Incluye la extracciÃ³n del panel de pago que US-056 no necesita todavÃ­a. |
| **5** | US-058 | Lee lo que US-056 escribe. Va al final porque es la Ãºnica US que se puede recortar sin dejar el sprint a medias a nivel de datos. |

---

## [x] US-055 â€” MigraciÃ³n 16, modo restaurante y gestiÃ³n de mesas (2026-09-29)

**Como** dueÃ±o de la cafeterÃ­a,
**quiero** registrar mis mesas y activar el modo restaurante,
**para** que el POS muestre mesas y barra solo si mi negocio las usa.

**Esfuerzo estimado:** 1.5 dÃ­as. **Impacto:** Alto. Es la base de todo el mÃ³dulo y el interruptor que evita que un cliente de tienda vea funciones que no le sirven.

**Estado verificado:**
- No existe ninguna tabla ni columna del mÃ³dulo: `grep -rn "dining_tables\|order_items\|restaurant_mode\|lan_access" internal/ frontend/` no devuelve nada.
- Ãšltima migraciÃ³n de `dev`: **17** (SPRINT-16/17 tomaron la 16, `jwt_secret`, y la 17, catÃ¡logo pÃºblico). El mÃ³dulo usa la **18** (renumerada al mergear US-055, 2026-09-30).
- `business_config` es tabla singleton (id = 1), se lee y escribe en `internal/repository/setup.go` con la struct `BusinessConfig` (lÃ­nea 7). El `UPSERT` de `Upsert` (`internal/repository/setup.go:83-92`) lista columna por columna: hay que agregar las nuevas ahÃ­ o no se persisten.
- El sidebar filtra Ã­tems por `visibleItems` (`frontend/components/Sidebar.tsx:143`; los Ã­tems viven en `NAV_ITEMS`, lÃ­nea 16, con la bandera `adminOnly`); ahÃ­ entra la condiciÃ³n de modo restaurante.
- La config del negocio ya llega al frontend por contexto: `frontend/lib/business-config-context.tsx`.

**Contexto tÃ©cnico:**
MigraciÃ³n 16 crea `dining_tables`, `orders` y `order_items` (SQL exacto en el DoD) y agrega dos columnas a `business_config`: `restaurant_mode` y `lan_access_enabled` (esta Ãºltima es de US-054). Nuevo `internal/repository/dining_table.go` + `internal/handler/tables.go`, registrados en el grupo autenticado de `Routes()` con el CRUD bajo el grupo admin, mismo patrÃ³n que `suppliers` (`internal/handler/handler.go:183-188`). `BusinessConfig` y su `UPDATE` en `setup.go` suman los dos campos nuevos. Pantalla de administraciÃ³n de mesas dentro de Ajustes (`frontend/app/settings/page.tsx`), no una ruta aparte.

**Regla de negocio:**
```
Estado de una mesa (derivado, nunca almacenado):
  libre        â†’ no existe orders con status='open' y ese table_id
  ocupada      â†’ existe orders open, y al menos un Ã­tem en pending|preparing
  por_cobrar   â†’ existe orders open, y todos sus Ã­tems (no anulados) estÃ¡n en ready|delivered
  Un pedido abierto sin Ã­tems cuenta como 'ocupada'.

Borrado de mesa:
  con pedido abierto        â†’ 409, no se borra
  sin pedido abierto        â†’ borrado lÃ³gico (active = 0), nunca DELETE fÃ­sico,
                              porque orders histÃ³ricas la referencian.
```

**Scope â€” quÃ© NO entra:**
- Plano visual arrastrable ni zonas con coordenadas. `zone` es solo texto para agrupar.
- Capacidad/comensales: la columna `seats` se crea pero no se usa en ninguna pantalla todavÃ­a.

**Acceptance Criteria:**

**Ajustes â†’ Mesas (admin):**
- [ ] Existe un toggle "Modo restaurante" que persiste en `business_config.restaurant_mode`
- [ ] Con el toggle apagado, la secciÃ³n de mesas no se muestra y el sidebar no tiene "Mesas" ni "Barra"
- [ ] Con el toggle encendido, se puede crear una mesa con nombre y zona opcional
- [ ] Nombre duplicado â†’ error visible, no se crea
- [ ] Nombre vacÃ­o â†’ error de validaciÃ³n
- [ ] Se puede editar nombre, zona y orden de una mesa
- [ ] Se puede desactivar una mesa sin pedido abierto
- [ ] Intentar borrar una mesa con pedido abierto â†’ mensaje "La mesa tiene un pedido abierto" y no se borra
- [ ] Estado vacÃ­o: sin mesas creadas, se ve un texto guÃ­a y el botÃ³n de crear
- [ ] Responsive: la tabla de mesas funciona en 320px

**Sidebar:**
- [ ] Con modo restaurante activo aparecen "Mesas" y "Barra"; un usuario `cashier` los ve, no solo admin

**DoD:**
- [ ] MigraciÃ³n 16 (`migration18` + `{version: 18, up: migration18}` en `migrate.go`; no editar la 15):
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
- [ ] `GET /api/tables` (autenticado) â†’ `[{ id, name, zone, seats, sort_order, active, state, open_order_id, open_total, opened_minutes }]` donde `state` es `libre|ocupada|por_cobrar`
- [ ] `POST /api/tables` (admin) â†’ `{ id, name, zone, seats, sort_order, active }`
- [ ] `PUT /api/tables/{id}` (admin) â†’ mesa actualizada
- [ ] `DELETE /api/tables/{id}` (admin) â†’ 409 si tiene pedido abierto; si no, `active = 0`
- [ ] `restaurant_mode` y `lan_access_enabled` se leen en `GET /api/setup` y se escriben en `PUT /api/setup` (agregar al `UPSERT` de `internal/repository/setup.go:83-92` y a `BusinessConfig`, lÃ­nea 7; el `INSERT` lista columnas explÃ­citamente)
- [ ] Test: mesa con pedido `open` â†’ `DELETE` responde 409 y la mesa sigue `active = 1`
- [ ] Test: mesa con 2 Ã­tems `ready` y 0 pendientes â†’ `state = "por_cobrar"`; con 1 Ã­tem `pending` â†’ `state = "ocupada"`; sin pedido â†’ `state = "libre"`
- [ ] Test: `PUT /api/setup` con `restaurant_mode: true` â†’ `GET /api/setup` lo devuelve `true` (verifica que la columna entrÃ³ al UPSERT)
- [ ] Build limpio (`go build ./...`, `pnpm build` en `frontend/`)

### Criterios verificados (US-055, cierre 2026-09-29)

QA: OK, 0 bloqueantes. `go build`, `go test ./...`, `pnpm build`, `pnpm test` (45 tests) pasan; `pnpm lint` queda en 41 problemas, igual a la base. Los checkboxes de arriba quedan como estaban en el sprint original; el estado real es el de esta lista.

**MigraciÃ³n y datos**
- AC-1: CUANDO la base arranca en versiÃ³n 17, el sistema aplica la migraciÃ³n 18 (3 tablas, 5 Ã­ndices, CHECK de `type` y `status`) y registra la versiÃ³n 16. Test: `internal/database/migrate_test.go` `TestMigration16_T1_TablesIndexesAndChecks`.
- AC-2: CUANDO se aplica la migraciÃ³n 18, `business_config` gana `restaurant_mode` y `lan_access_enabled`, ambas `INTEGER NOT NULL DEFAULT 0`. Test: `TestMigration16_T2_BusinessConfigDefaults`.
- AC-3: CUANDO un admin envÃ­a `PUT /api/setup` con `restaurant_mode` y/o `lan_access_enabled`, se persisten y `GET /api/setup` los devuelve; `false` o ausentes devuelven `false`. Tests: `internal/handler/setup_test.go` `TestSetup_T3_PersistsAndClearsFlags` (true/true y luego false/false: cubre INSERT y DO UPDATE SET), `TestSetup_T4_AbsentLanFlagDefaultsFalse`.

**API de mesas** (todos en `internal/handler/tables_test.go`)
- AC-5: `POST /api/tables` con nombre â†’ 201 con `active=true`, `zone=""`, `seats=0`, `sort_order=0`. `TestTables_T5_CreateDefaultsAndValues`.
- AC-6: nombre vacÃ­o o solo espacios â†’ 400 y no se crea. `TestTables_T6_EmptyNameRejected`.
- AC-7: nombre duplicado (tambiÃ©n con la mesa desactivada) â†’ 409 en espaÃ±ol sin texto SQL. `TestTables_T7_DuplicateNameConflict`.
- AC-8: `PUT /api/tables/{id}` â†’ 200 con la mesa actualizada; nombre de otra mesa â†’ 409; id inexistente â†’ 404. `TestTables_T8_Update`.
- AC-9: `DELETE` sin pedido abierto â†’ 200, `active = 0`, la fila sigue; pedidos `closed` y `cancelled` no bloquean. `TestTables_T9_DeleteIsLogical`.
- AC-10: `DELETE` con pedido `open` â†’ 409 "La mesa tiene un pedido abierto" y la mesa sigue `active = 1`. `TestTables_T10_DeleteBlockedByOpenOrder`.
- AC-11 y AC-12: `GET /api/tables` devuelve activas ordenadas por `sort_order` y `name`; sin mesas devuelve `[]`; sin pedido â†’ `libre`. `TestTables_T11_ListOrderAndEmpty`.
- AC-13: estado derivado y total: cafÃ© $4.500 x2 + torta $8.000 `ready` â†’ `por_cobrar`, `$17.000`; con un Ã­tem `pending` â†’ `ocupada`; sin Ã­tems â†’ `ocupada`, `$0`; torta `void` no suma (`$9.000`); `delivered`+`ready` â†’ `por_cobrar`; `delivered`+`preparing` â†’ `ocupada`; pedido creado hace 25 min â†’ `opened_minutes` entre 24 y 25. `TestTables_T12_DerivedStates`, `TestTables_T13_VoidAndMixedStatuses`, `TestTables_T14_OpenedMinutesUTC`.
- AC-14: `cashier` recibe 403 en `POST`/`PUT`/`DELETE /api/tables` y 200 en `GET /api/tables` y `GET /api/setup`; admin crea con 201. `TestTables_T15_Permissions`.

**Interfaz**
- AC-16 y AC-17 (sidebar): con `restaurant_mode` apagado no hay "Mesas" ni "Barra" para nadie; encendido las ven admin y `cashier` (este Ãºltimo sin `/settings`). `frontend/lib/nav.test.ts`, tres casos de `describe('filterNavItems (T-16)')`.
- AC-21: build, tests y lint sin problemas nuevos (QA, T-18).

**VerificaciÃ³n en navegador (T-17), 2026-09-29.** La QA de la vuelta 1 no pudo hacer smoke (puerto 8080 fijo en `cmd/pos/main.go:41`, ocupado). El orquestador la hizo despuÃ©s con Edge + playwright-core (`e2e.js`), binario del worktree, DB nueva, 320 px. HallÃ³ un BLOQUEANTE que la QA no vio: `SettingsPage` consumÃ­a `useBusinessConfig()` fuera del provider (`frontend/components/AppLayout.tsx:56`) y recibÃ­a `defaultConfig`, asÃ­ que la secciÃ³n Mesas no se renderizaba. Arreglado en b861090: `SettingsContent` (`frontend/app/settings/page.tsx:181`) va dentro de `<AppLayout>` (`:580-581`). QA vuelta 2: OK, 1 MENOR (`settings/page.tsx:105-121`: la carga de mesas estÃ¡ duplicada entre `load()` y un `useEffect`; sin efecto funcional).
- Verificado en navegador: sin modo restaurante no hay secciÃ³n Mesas (AC-16); el toggle persiste y aparece la secciÃ³n (AC-15, AC-17); estado vacÃ­o con texto guÃ­a y botÃ³n (AC-19); crear mesa; nombre duplicado y nombre vacÃ­o muestran el error del servidor y no cambian la lista (AC-18); desactivar; sin scroll horizontal a 320 px (AC-20); `restaurant_mode` sigue en `true` tras guardar el formulario (AC-4, parte `restaurant_mode`); el sidebar muestra "Mesas" y "Barra" (AC-17).
- No cubierto en navegador: AC-4 con `lan_access_enabled=1` previo en la DB (queda cubierto por lectura, `settings/page.tsx:235-236`, y por `TestSetup_T3_PersistsAndClearsFlags` a nivel API); desactivar una mesa con pedido abierto (AC-18, requiere pedido sembrado por SQL; sÃ­ lo cubre `TestTables_T10_DeleteBlockedByOpenOrder`); revisiÃ³n de tema claro/oscuro y dispositivo real, a cargo de Andrey si lo desea.
- Ruido observado, ya cubierto: cargar `/settings` u otras rutas directo devolvÃ­a el dashboard; es R-55 (`web/static.go`, planeado en US-054). PatrÃ³n repetido en `CustomerDetailClient` registrado como R-61.

---

## [x] US-054 â€” Acceso al POS desde el celular/tablet por WiFi local (K1, deuda SPRINT-12) (2026-09-30)

**Como** dueÃ±o,
**quiero** entrar al POS desde el celular o una tablet conectada a mi WiFi,
**para** que el barista use el KDS en una tablet en la barra y yo vea las ventas sin estar en el computador.

**Esfuerzo estimado:** 1 dÃ­a (0.5 del bind + 0.5 de R-55). **Impacto:** Alto en este sprint â€” es dependencia dura del KDS (US-058), no una comodidad.

**Estado verificado (al cerrar, 2026-09-30):**
- `lan_access_enabled` lo creÃ³ la migraciÃ³n 18 de US-055 (`internal/database/migrate.go:548`); esta US no agregÃ³ migraciÃ³n.
- El bind sale de `listenAddr(lan)` (`cmd/pos/main.go:72`), con `lan` leÃ­do de `business_config` al arrancar (`main.go:55-57`); el navegador se abre siempre en `browserURL()` (`main.go:80`).
- R-55 cerrado: `web/static.go` resuelve archivo -> `<ruta>.html` -> `customers/_.html`/`catalogo/_.html` -> 404 para `/_next/` y assets -> `index.html`; `frontend/lib/api.ts:3` usa `/api` relativo por defecto. CORS `:3000` (`internal/middleware/middleware.go:94-97`) sin cambio: solo aplica al dev.
- Se agregÃ³ `web/out/.gitkeep` para que `go build ./...` compile en un checkout limpio.

**Contexto tÃ©cnico:**
`listenAddr(lanEnabled bool) string` como funciÃ³n exportada y pura en `cmd/pos/`, no inline en `main()`, especÃ­ficamente para poder testearla sin levantar servidor. Devuelve `127.0.0.1:8080` con el toggle apagado y `0.0.0.0:8080` con el toggle encendido. `main()` lee `business_config.lan_access_enabled` al arrancar. En Ajustes, junto al toggle, se muestra la IP local detectada y la URL que hay que escribir en la tablet.

**Regla de negocio:**
```
lan_access_enabled = 0  â†’ bind 127.0.0.1:8080  (solo esta mÃ¡quina)
lan_access_enabled = 1  â†’ bind 0.0.0.0:8080    (toda la red local)
El cambio aplica al reiniciar pos.exe. No es un bug: net/http bindea el
socket una sola vez al arrancar.
```

**Scope â€” quÃ© NO entra:**
- HTTPS / certificados. Es red local.
- Aplicar el toggle en caliente sin reiniciar.

**Acceptance Criteria:**

**Ajustes:**
- [x] Toggle "Permitir acceso desde otros dispositivos de mi red WiFi", persiste en `business_config.lan_access_enabled`
- [x] Al activarlo se muestra la URL con la IP local detectada (ej. `http://192.168.1.15:8080`) y el aviso de que hay que reiniciar el programa
- [x] Si no se detecta IP local, se muestra un texto de ayuda en vez de una URL rota

**Acceso:**
- [x] Con el toggle apagado, `pos.exe` solo responde en `localhost`
- [ ] Con el toggle encendido y tras reiniciar, un celular en la misma WiFi abre el POS y puede iniciar sesiÃ³n (AC-16, pendiente de Andrey)
- [ ] Responsive: el login y el POS abren usables en un celular de 320px (pendiente de verificar en celular real, con la prueba manual)

**DoD:**
- [x] Usa la columna `lan_access_enabled` de la migraciÃ³n 18 (US-055); esta US no crea migraciÃ³n
- [x] `listenAddr` exportada y pura en `cmd/pos/`
- [x] Test: `listenAddr(false) == "127.0.0.1:8080"` y `listenAddr(true) == "0.0.0.0:8080"`
- [x] R-55: `FileServer` quita la barra inicial y sirve `<ruta>.html` para rutas exportadas; `curl` de un chunk `/_next/static/chunks/*.js` devuelve `text/javascript` (el tipo que entrega `net/http`), no `text/html`
- [x] R-55: test de `FileServer` con un FS en memoria: `/`, `/login` (â†’ `login.html`), un `.js` existente y una ruta inexistente (â†’ `index.html`)
- [x] R-55: `api.ts` usa `/api` relativo por defecto; el dev con `:3000` sigue funcionando vÃ­a `NEXT_PUBLIC_API_URL`, documentado en `INSTALL.md`/`CLAUDE.md` si cambia
- [x] Test: detecciÃ³n de IP local devuelve una IPv4 privada (10.x / 172.16-31.x / 192.168.x) o cadena vacÃ­a, nunca `127.0.0.1` como sugerencia al usuario
- [ ] Test manual con dos dispositivos reales â€” **pendiente para Andrey**, no automatizable en este entorno
- [x] Build limpio (`go build ./...`, `pnpm build`)

---

**Criterios verificados (cierre 2026-09-30, QA vuelta 1 OK: 0 BLOQUEANTE, 3 MENOR):**
- AC-1/AC-2: MIENTRAS `lan_access_enabled` = 0 (o la config no se pueda leer) el sistema DEBE escuchar en `127.0.0.1:8080`; con 1, en `0.0.0.0:8080`. Test: `cmd/pos/main_test.go` `TestListenAddr_T1`.
- AC-3: el navegador se abre siempre en `http://localhost:8080`. Test: `cmd/pos/main_test.go` `TestBrowserURL_T2`.
- AC-4: el toggle persiste en `business_config.lan_access_enabled` y aplica al reiniciar. Test: `internal/handler/setup_test.go` `TestSetup_T4_AbsentLanFlagDefaultsFalse` (mÃ¡s el PUT/GET de US-055 en el mismo archivo).
- AC-6/AC-7/AC-8: la IP sugerida es solo IPv4 privada (10/8, 172.16/12, 192.168/16), nunca loopback ni 169.254.x; sin IP, Ajustes muestra ayuda. `GET /api/setup/lan-info` solo admin. Tests: `internal/netutil/lan_test.go` `TestPickPrivateIPv4_T4`; `internal/handler/setup_test.go` `TestSetup_US054_T5_LanInfo`.
- AC-9..AC-14: `FileServer` sirve `/` -> `index.html`; `/login` -> `login.html` sin redirigir (incluso con directorio homÃ³nimo); `/customers/<id>` y `/catalogo/<token>` -> `_.html`; asset inexistente o `/_next/*` -> 404; ruta desconocida -> `index.html`. Tests: `web/static_test.go` `TestFileServer_T6_RootServesIndex`, `_T7_HtmlExtensionWithoutRedirect`, `_T8_AssetContentType`, `_T9_DynamicRoutes`, `_T10_FallbackAndAsset404`.
- AC-15: sin `NEXT_PUBLIC_API_URL` la UI usa `/api`; con la variable, su valor. Test: `frontend/lib/api.test.ts` (`uses relative /api when NEXT_PUBLIC_API_URL is undefined`, `uses NEXT_PUBLIC_API_URL when defined`).
- AC-5/AC-17: toggle visible en Ajustes con URL o ayuda, aviso de reinicio y aviso de cambiar `admin/admin`. Solo revisado por lectura de `frontend/app/settings/page.tsx`; T-12 no se verificÃ³ en navegador.
- AC-16 / T-M: un celular real en la misma WiFi abre el POS e inicia sesiÃ³n (320px). **Pendiente de Andrey**, no automatizable.
- MENOR aceptados sin Ã­tem: import de `internal/netutil` dentro del grupo stdlib (`internal/handler/handler.go:6`) y doble lÃ­nea en blanco antes de `getLanInfo` (`internal/handler/setup.go:64-65`); son de estilo. MENOR 3 (lint 42 vs 41): no es regresiÃ³n de esta US, ver bitÃ¡cora de `roadmap.md`.

---

## US-056 â€” Pedido por mesa: abrir, agregar Ã­tems con nota, anular, cancelar

**Como** cajero de la cafeterÃ­a,
**quiero** abrir una mesa y agregarle Ã­tems con sus indicaciones,
**para** dejar de escribir el pedido en un post-it y que la cuenta del cliente quede registrada.

**Esfuerzo estimado:** 2.5 dÃ­as. **Impacto:** Alto. Es el nÃºcleo del mÃ³dulo.

**Estado verificado:**
- El POS actual (`frontend/app/pos/page.tsx`, 1624 lÃ­neas) guarda carritos aparcados en `localStorage` (`PARKED_CARTS_KEY`, lÃ­nea 69). Es una cuenta abierta local a ese navegador: el barista en su tablet nunca la ve. Con `restaurant_mode` activo ese botÃ³n se oculta.
- Componentes reutilizables ya existentes en esa pÃ¡gina: `ProductCard` (lÃ­nea 152) y `CartItemRow` (lÃ­nea 200). Se extraen o se replican deliberadamente; ver Notas tÃ©cnicas.
- `repository.InsertAuditLog(ctx, execer, userID, action, entity, entityID, details)` (`internal/repository/audit.go:21`) recibe un `execer`, asÃ­ que funciona dentro de una transacciÃ³n.

**Contexto tÃ©cnico:**
Nuevo `internal/repository/order.go` (+ `order_test.go`), `internal/service/order.go`, `internal/handler/order.go`, rutas en el grupo autenticado. Frontend: `frontend/app/mesas/page.tsx` (grid de mesas) y `frontend/app/mesas/[orderId]/page.tsx` (pedido). NingÃºn cambio en ventas, caja, reportes ni facturaciÃ³n: este sprint no escribe en `sales` hasta US-057.

**Regla de negocio:**
```
Totales del pedido (sin impuestos ni descuento global en v1):
  linea.subtotal = quantity * unit_price
  pedido.total   = suma de subtotales de Ã­tems con status != 'void'

unit_price se copia del producto al agregar el Ã­tem (snapshot).
Cambiar el precio del producto despuÃ©s NO altera pedidos ya abiertos.

AnulaciÃ³n de Ã­tem:
  status = 'void' (nunca DELETE fÃ­sico) + registro en audit_log
  action='order_item_void', entity='order_item', entity_id=<item id>

CancelaciÃ³n de pedido:
  solo si status='open'. Pasa a 'cancelled' + audit_log
  action='order_cancel', entity='order', entity_id=<order id>, details=<motivo>
  Un pedido 'closed' no se puede cancelar (ya es una venta: se anula la venta).

Toda escritura de Ã­tem toca orders.updated_at (misma transacciÃ³n).
```

**Scope â€” quÃ© NO entra:**
- Pedido para llevar con turno (R-01 en `roadmap.md`). En este sprint `type` siempre es `'table'`; la columna ya acepta `'takeaway'`.
- Modificadores con precio. La variaciÃ³n va en `note`, texto libre.
- Mover Ã­tems entre mesas, dividir cuenta, propina.

**Acceptance Criteria:**

**`/mesas` (grid):**
- [ ] Muestra todas las mesas activas con su estado por color: libre / ocupada / por cobrar
- [ ] Una mesa ocupada muestra el total acumulado y los minutos desde que se abriÃ³
- [ ] Tocar una mesa libre abre un pedido nuevo y navega a su pantalla
- [ ] Tocar una mesa ocupada navega al pedido existente (no crea uno segundo)
- [ ] Estado vacÃ­o: sin mesas creadas, enlace a Ajustes â†’ Mesas
- [ ] Responsive: el grid funciona en 320px (una columna)

**`/mesas/[orderId]` (pedido):**
- [ ] Buscador y grid de productos; tocar un producto lo agrega al pedido
- [ ] Cada lÃ­nea permite editar cantidad y escribir una nota de texto libre
- [ ] El Ã­tem se guarda en el servidor al agregarlo, sin botÃ³n "enviar a cocina"
- [ ] Cada lÃ­nea muestra su estado (pendiente / preparando / listo)
- [ ] Anular una lÃ­nea pide confirmaciÃ³n y la deja tachada, no la borra de la vista
- [ ] Cancelar el pedido pide motivo y libera la mesa
- [ ] El total se recalcula al instante con cada cambio
- [ ] Error de red al agregar un Ã­tem â†’ mensaje visible y la lÃ­nea no queda fantasma en la UI
- [ ] Responsive: en 320px el catÃ¡logo y el pedido se alternan por pestaÃ±as, como ya hace el POS (`mobileTab`)

**POS existente:**
- [ ] Con `restaurant_mode` activo, el botÃ³n de carritos guardados no se muestra
- [ ] Con `restaurant_mode` apagado, el POS se comporta exactamente como hoy

**DoD:**
- [ ] Sin migraciÃ³n nueva (usa las tablas de la migraciÃ³n 18)
- [ ] `POST /api/orders` (autenticado) â†’ `{ id, type, table_id, status, created_at }` Â· 409 si la mesa ya tiene pedido abierto
- [ ] `GET /api/orders?status=open` (autenticado) â†’ `[{ id, table_id, table_name, total, items_pending, opened_minutes }]`
- [ ] `GET /api/orders/{id}` (autenticado) â†’ `{ id, type, table_id, status, total, items: [{ id, product_id, product_name, quantity, unit_price, subtotal, note, status }] }`
- [ ] `POST /api/orders/{id}/items` (autenticado) â†’ Ã­tem creado Â· 409 si el pedido no estÃ¡ `open`
- [ ] `PATCH /api/orders/{id}/items/{itemId}` (autenticado) â†’ acepta `quantity`, `note`, `status`
- [ ] `DELETE /api/orders/{id}/items/{itemId}` (autenticado) â†’ `status='void'` + fila en `audit_log`
- [ ] `POST /api/orders/{id}/cancel` (autenticado) â†’ `{reason}` obligatorio, pedido `cancelled` + fila en `audit_log`
- [ ] Test: pedido con 2 Ã­tems (cafÃ© $4.500 x2 y torta $8.000 x1) â†’ total 17.000; se anula la torta â†’ total 9.000 y el Ã­tem sigue en la tabla con `status='void'`
- [ ] Test: abrir pedido en una mesa que ya tiene uno abierto â†’ 409 y sigue habiendo exactamente 1 pedido abierto para esa mesa
- [ ] Test: cancelar pedido con Ã­tems â†’ `orders.status='cancelled'`, se inserta 1 fila en `audit_log` con el `user_id` correcto, y la mesa vuelve a `state='libre'`
- [ ] Test: agregar Ã­tem a un pedido `closed` â†’ 409
- [ ] Build limpio (`go build ./...`, `pnpm build`)

**Estado: [x] completa en cÃ³digo y tests (2026-09-30); verificaciÃ³n manual T-19 pendiente del PO.** Los checkboxes de UI de arriba (`/mesas`, `/mesas/[orderId]`, POS) quedan sin marcar hasta esa prueba; build, tests backend y vitest pasaron en QA.

**Criterios verificados (cierre 2026-09-30, QA vuelta 1 OK: 0 BLOQUEANTE, 4 MENOR):**
- AC-1/AC-3: CUANDO un autenticado hace `POST /api/orders` con `table_id` de mesa activa sin pedido abierto, el sistema DEBE crear un pedido `type='table'`, `open`, con el `user_id` del token (201); `table_id` inexistente o desactivada â†’ 404, falta/invÃ¡lido o `type` â‰  `table` â†’ 400, sin crear fila. Tests: `internal/handler/order_test.go` `TestOrder_T11_OpenValidation`, `TestOrder_T2_SecondOpenConflicts`.
- AC-2: SI la mesa ya tiene pedido `open`, ENTONCES 409, incluso en concurrencia (INSERT condicional atÃ³mico, sin migraciÃ³n); `SQLITE_BUSY` â†’ 409 "reintenta", nunca 500. Tests: `internal/repository/order_test.go` `TestOrder_T3_ConcurrentOpenSingleWinner`, `TestOrder_T15_InactiveTableAndOpenThenDeactivate`; `internal/handler/order_test.go` `TestOrder_T22_BusyMapsTo409`.
- AC-4/AC-5/AC-6/AC-7/AC-7b: agregar Ã­tem con snapshot de precio; `quantity` entero > 0 (0, negativos, `0.5`, `2.0` â†’ 400); solo con pedido `open` (si no 409); `PATCH` de `quantity`/`note`/`status` (`pending|preparing|ready|delivered`, `void` â†’ 400); editable en `ready`/`delivered`. Ej.: cafÃ© $4.500 x2 â†’ tras `PATCH quantity=3` subtotal $13.500. Tests: `TestOrder_T1_AddItemsTotalAndVoid`, `T6_ClosedOrderRejectsItemChanges`, `T7_ValidationOfQuantityStatusProduct`, `T8_PriceSnapshot`, `T10_PatchQuantityAndNote`, `T20_EditReadyAndDeliveredItems`.
- AC-7c: `ready_at` se fija al pasar a `ready`, no se sobrescribe al reenviar y vuelve a NULL al salir de `ready`. Tests: `TestOrder_T21_ReadyAtSetAndPreserved`, `T21b_ReadyAtClearedWhenLeavingReady`.
- AC-8: `DELETE` de Ã­tem â†’ `status='void'` + `audit_log order_item_void` con `details` JSON de 9 claves (`status` = estado previo), toca `updated_at`; ya anulado â†’ 409. Pedido con cafÃ© x2 ($9.000) y torta $8.000: total $17.000, tras anular la torta $9.000. Tests: `TestOrder_T1_AddItemsTotalAndVoid`, `T1b_VoidDetailsStatusIsPreVoidState`, `T9_TouchesUpdatedAt`.
- AC-9/AC-10/AC-11: `POST /cancel` con motivo no vacÃ­o â†’ `cancelled`, `closed_at`, `audit_log order_cancel`, Ã­tems conservados, mesa `libre`; motivo vacÃ­o â†’ 400; no `open` â†’ 409. Tests: `TestOrder_T4_CancelOrder`, `T5_CancelRejections`.
- AC-12/AC-13: `GET /orders/{id}` con total sin `void` e `items` incluyendo `void`; `GET /orders?status=open` con `items_pending` y `opened_minutes`, `[]` si vacÃ­o, `status` invÃ¡lido â†’ 400. Tests: `TestOrder_T1_AddItemsTotalAndVoid`, `T12_ListOpen`.
- AC-14: cajero y admin operan pedidos; sin token 401. Test: `TestOrder_T13_CashierAllowedAnonymousRejected`.
- AC-15 (R-59): `Deactivate` atÃ³mico, 409 con pedido abierto. Test: `internal/repository/dining_table_test.go` `TestDiningTable_T14_DeactivateWithOpenOrder`; regresiÃ³n de US-055 en `internal/handler/tables_test.go`.
- AC-16/AC-24: CORS con `PATCH` y `api.patch`. Tests: `internal/middleware/middleware_test.go` `TestCORS_T16_AllowsPatch`; `frontend/lib/api.test.ts` (`T-18: sends PATCH with JSON body`).
- AC-22: `/mesas/12` se sirve con `mesas/_.html`. Test: `web/static_test.go` `TestFileServer_T9_DynamicRoutes`.
- AC-17..AC-21, AC-23 (UI): verificados solo por `pnpm build` (genera `/mesas` y `/mesas/_`) y lectura de cÃ³digo; `showParked = !restaurant_mode` en `frontend/app/pos/page.tsx:622,1030,1141,1155`. **T-19 (manual a 320px y 1280px) PENDIENTE del PO**: grid por estado, tocar ocupada no duplica, backend apagado sin lÃ­nea fantasma, anulaciÃ³n tachada, cancelaciÃ³n con motivo, y con `restaurant_mode`=1 sin chips de espera en el POS.
- Fuera de alcance y registrado: refresco automÃ¡tico (R-63) y 4 MENOR de QA (R-64, R-65, R-66; ver `roadmap.md`). `TestDashboardToday_T20_Golden` falla en `dev` por fecha (preexistente, R-67).

---

## US-057 â€” Cobro del pedido reusando el flujo de venta

**Como** cajero,
**quiero** cobrar la mesa con los mismos medios de pago de siempre,
**para** que la venta quede en caja, en los reportes y con su ticket, sin registrar nada dos veces.

**Esfuerzo estimado:** 1.5 dÃ­as. **Impacto:** Alto. Es lo que conecta el mÃ³dulo nuevo con el negocio real.

**Estado verificado:**
- `createSale` (`internal/handler/sale.go:15-48`) calcula subtotales de Ã­tems, arma `model.Sale` y llama `h.sales.Create`; despuÃ©s dispara la facturaciÃ³n electrÃ³nica en goroutine (`internal/handler/sale.go:44-46`).
- `SaleService.Create` (`internal/service/sale.go:39`) es el punto Ãºnico de negocio; el repositorio (`internal/repository/sale.go:37`) hace en una sola transacciÃ³n: venta + `sale_payments` + `sale_items` + stock + movimiento de caja.
- El panel de pago vive dentro de `POSPage` con su propio estado: `payMethod`, `extraPayments`, `firstAmountInput`, `cashReceived` (`frontend/app/pos/page.tsx:652-658`), y la lÃ³gica de pagos mixtos en `paymentLines` / `addPaymentLine` (lÃ­neas 899-916). Los cÃ¡lculos ya estÃ¡n fuera, en `frontend/lib/saleCalc.ts`, con tests.

**Contexto tÃ©cnico:**
El handler de checkout construye un `model.Sale` desde los Ã­tems no anulados del pedido y llama el mismo `SaleService.Create`; no se duplica una lÃ­nea de lÃ³gica de cobro. Cerrar el pedido (`status='closed'`, `sale_id`, `closed_at`) ocurre en la misma transacciÃ³n que crea la venta, o el pedido queda abierto. Frontend: se extrae el panel de pago a `frontend/components/SalePaymentPanel.tsx` y lo consumen el POS y la pantalla de pedido. Los cÃ¡lculos siguen viniendo de `lib/saleCalc.ts` â€” la extracciÃ³n mueve UI y estado, no aritmÃ©tica.

**Regla de negocio:**
```
Al cobrar:
  items considerados = order_items con status != 'void'
  sale.subtotal      = suma de (quantity * unit_price)
  sale.total         = subtotal - descuento + impuesto   (igual que hoy en createSale)
  cada order_item    â†’ un sale_item con el MISMO unit_price (snapshot del pedido)

Atomicidad: venta + cierre del pedido en una sola transacciÃ³n.
  Si la venta falla â†’ orders.status sigue 'open', no hay venta parcial.

Pedido sin Ã­tems vÃ¡lidos (vacÃ­o o todo anulado) â†’ 400, no se crea venta.
El stock se descuenta aquÃ­, no al tomar el pedido.
```

**Scope â€” quÃ© NO entra:**
- Propina y divisiÃ³n de cuenta (R-12 en `roadmap.md`).
- Descuento global por pedido: se usa el mismo campo de descuento de la venta, sin UI nueva por lÃ­nea.

**Acceptance Criteria:**

**Cobro desde el pedido:**
- [ ] BotÃ³n "Cobrar" abre el mismo panel de pago del POS (efectivo, transferencia, tarjeta, crÃ©dito y mixto)
- [ ] Efectivo muestra cambio con la misma lÃ³gica de hoy
- [ ] CrÃ©dito exige cliente y respeta el lÃ­mite de crÃ©dito igual que el POS
- [ ] Al confirmar se muestra el ticket con el mismo componente de siempre
- [ ] Terminado el cobro, la mesa queda libre en `/mesas`
- [ ] Cobrar un pedido sin Ã­tems â†’ botÃ³n deshabilitado y, si se fuerza por API, 400
- [ ] Error al cobrar â†’ mensaje visible, el pedido sigue abierto y no se creÃ³ venta

**POS existente (regresiÃ³n):**
- [ ] El POS cobra exactamente igual que antes tras la extracciÃ³n del panel: efectivo, mixto con 2 y 3 lÃ­neas, crÃ©dito
- [ ] Responsive: el panel extraÃ­do sigue funcionando en 320px en ambas pantallas

**Cierre de caja:**
- [ ] Cerrar caja con pedidos abiertos muestra el aviso "Hay N pedidos sin cobrar" y **permite** cerrar igual

**DoD:**
- [ ] Sin migraciÃ³n nueva
- [ ] `POST /api/orders/{id}/checkout` (autenticado) â†’ mismo body que `POST /api/sales` (`{ payments: [{method, amount, reference}], customer_id?, discount?, tax?, buyer_email? }`) â†’ `{ order_id, sale_id, total }`
- [ ] El checkout llama `SaleService.Create`; no se reimplementa stock, caja ni pagos
- [ ] `frontend/components/SalePaymentPanel.tsx` existe y es consumido por `app/pos/page.tsx` y `app/mesas/[orderId]/page.tsx`
- [ ] Test: pedido con cafÃ© $4.500 x2 + torta $8.000 â†’ checkout con efectivo $20.000 â†’ venta total 17.000, cambio 3.000, `orders.status='closed'` con `sale_id` poblado, 2 filas en `sale_items`
- [ ] Test: pedido con 3 Ã­tems, uno anulado â†’ la venta tiene 2 `sale_items` y el total excluye el anulado
- [ ] Test: checkout de pedido sin Ã­tems vÃ¡lidos â†’ 400 y `COUNT(*) FROM sales` no cambia
- [ ] Test: checkout con pago mixto $10.000 efectivo + $7.000 transferencia sobre total 17.000 â†’ 2 filas en `sale_payments`, `payment_method='mixed'`, saldo 0
- [ ] Build limpio (`go build ./...`, `pnpm build`, `pnpm test` para los tests de `saleCalc`)

---

## US-058 â€” Pantalla de barra (KDS) con estados, tiempos y alerta sonora

**Como** barista,
**quiero** ver en una pantalla lo que tengo que preparar y marcarlo listo,
**para** dejar de trabajar con post-its y que nadie tenga que preguntarme si ya saliÃ³.

**Esfuerzo estimado:** 2 dÃ­as. **Impacto:** Alto. Es la US que resuelve el dolor que el cliente describiÃ³.

**Estado verificado:**
- No existe ninguna pantalla ni endpoint de cocina/barra en el repo.
- No hay SSE ni WebSocket en el backend hoy: todo es request/response con Chi (`internal/handler/handler.go`).
- `frontend/hooks/useIsMobile.ts` es el Ãºnico hook propio; el polling se implementa con `useEffect` + `setInterval`, sin librerÃ­a nueva.

**Contexto tÃ©cnico:**
Nuevo `GET /api/kds/orders` que devuelve los Ã­tems en `pending|preparing|ready` de pedidos `open` **y** `closed` (el prepago se cobra antes de prepararse), con `server_time` para calcular los minutos sin depender del reloj de la tablet. Nueva pÃ¡gina `frontend/app/barra/page.tsx` con polling de 3 s. El avance de estado reusa el `PATCH /api/orders/{id}/items/{itemId}` de US-056; esta US no crea endpoints de escritura.

**Regla de negocio:**
```
Columnas: pending â†’ preparing â†’ ready. Un toque avanza al siguiente estado.
'delivered' se marca desde el pedido (US-056), no desde la barra.
Los Ã­tems 'void' nunca aparecen en la barra; si se anula uno ya mostrado,
desaparece en el siguiente poll.

minutos = (server_time - item.created_at) / 60
  < 5 min   â†’ normal
  5â€“10 min  â†’ advertencia (Ã¡mbar)
  > 10 min  â†’ alerta (rojo)

Sonido: suena una vez por poll si aparecieron ids de Ã­tem no vistos antes.
Nunca suena en el primer poll tras abrir la pantalla (si no, suena con
todo el backlog del dÃ­a).
```

**Scope â€” quÃ© NO entra:**
- SSE / WebSocket. El polling de 3 s se cambia despuÃ©s sin tocar el contrato de la API.
- ImpresiÃ³n de comanda en tÃ©rmica.
- Vistas separadas por estaciÃ³n (barra / cocina).

**Acceptance Criteria:**

**`/barra`:**
- [ ] Tres columnas: Pendiente, Preparando, Listo
- [ ] Cada tarjeta muestra mesa (o nÃºmero de turno cuando exista), los Ã­tems con cantidad, y la nota resaltada visualmente
- [ ] Cada tarjeta muestra los minutos transcurridos, con color segÃºn los umbrales de la regla
- [ ] Un toque en el Ã­tem lo avanza de estado y el cambio se ve en menos de 3 s en otra pantalla abierta
- [ ] La lista se refresca sola cada 3 s sin recargar la pÃ¡gina ni perder el scroll
- [ ] BotÃ³n "Activar sonido" al entrar; despuÃ©s de tocarlo, un Ã­tem nuevo dispara el sonido
- [ ] No suena en el primer refresco al abrir la pantalla
- [ ] Un Ã­tem anulado desde el pedido desaparece de la barra en el siguiente refresco
- [ ] Estado vacÃ­o: sin pedidos pendientes, mensaje grande "Todo al dÃ­a"
- [ ] Si el servidor no responde, se muestra "Sin conexiÃ³n" y al volver la red se reanuda sola sin recargar
- [ ] Legibilidad: tipografÃ­a y botones usables en una tablet a un metro de distancia
- [ ] Responsive: en 320px las tres columnas se apilan

**DoD:**
- [ ] Sin migraciÃ³n nueva
- [ ] `GET /api/kds/orders` (autenticado) â†’ `{ server_time, orders: [{ order_id, type, table_name, turn_number, created_at, items: [{ id, product_name, quantity, note, status, created_at }] }] }`
- [ ] Devuelve Ã­tems de pedidos `open` y `closed`; excluye `void`, `delivered` y pedidos `cancelled`
- [ ] El avance de estado usa el `PATCH` de US-056, sin endpoint nuevo
- [ ] Test: pedido **cobrado** (`status='closed'`) con un Ã­tem `pending` â†’ aparece en la respuesta del KDS (regresiÃ³n del caso prepago)
- [ ] Test: pedido con 3 Ã­tems (1 pending, 1 ready, 1 void) â†’ la respuesta trae 2 Ã­tems y ninguno es el anulado
- [ ] Test: `PATCH` de Ã­tem a `ready` â†’ `ready_at` queda poblado y `orders.updated_at` se actualiza
- [ ] Build limpio (`go build ./...`, `pnpm build`)

---

## Criterio de Ã©xito del sprint

- El dueÃ±o activa el modo restaurante, crea 8 mesas y permite el acceso por WiFi en menos de 5 minutos.
- El cajero abre la mesa 3, agrega dos capuchinos con la nota "leche deslactosada" y una torta, y el barista los ve en la tablet en menos de 3 segundos sin que nadie escriba un papel.
- El barista marca listo un capuchino y el cajero lo ve en la mesa; la torta sigue pendiente en la barra.
- El cajero cobra la mesa con pago mixto, sale el ticket de siempre, la venta aparece en el cierre de caja del dÃ­a y la mesa queda libre.

---

## Notas tÃ©cnicas para el dev

- **US-056 y US-057 comparten el catÃ¡logo de productos** con `app/pos/page.tsx`: `ProductCard` (lÃ­nea 152) y el buscador. Extraer `ProductCard` a `components/` antes de copiarlo; ya va a ser su tercer uso.
- **La extracciÃ³n de `SalePaymentPanel` (US-057) es la Ãºnica refactorizaciÃ³n autorizada del sprint.** Hacerla en un commit aparte del checkout, y probar el POS antes de seguir: si se mezcla con la lÃ³gica nueva y algo se rompe, no se sabe cuÃ¡l de las dos fue.
- `repository.InsertAuditLog` acepta un `execer` (`internal/repository/audit.go:21`), asÃ­ que la anulaciÃ³n de Ã­tem y la cancelaciÃ³n de pedido escriben la auditorÃ­a **dentro** de la misma transacciÃ³n. Fuera de ella, un rollback deja una auditorÃ­a de algo que nunca pasÃ³.
- `turn_number` no se usa en este sprint pero la columna se crea en la migraciÃ³n 18: SQLite no permite agregar columnas con `UNIQUE` despuÃ©s, y agregarla ahora evita otra migraciÃ³n solo para eso.
- Fechas en SQLite: usar `date('now','localtime')` para cualquier lÃ³gica de "hoy". `CURRENT_TIMESTAMP` guarda UTC y en Colombia (UTC-5) parte el dÃ­a a las 7 p.m.
- El `UPSERT` de `internal/repository/setup.go:83-92` lista las columnas una por una. Cada columna nueva de `business_config` hay que agregarla ahÃ­ o se guarda en silencio sin efecto â€” es un error fÃ¡cil de no notar hasta la demo.
- `orders.updated_at` no se actualiza solo: tocarlo explÃ­citamente en toda escritura de Ã­tem, o el KDS mostrarÃ¡ datos viejos.

---

## Roadmap

Lo que queda por construir vive en [roadmap.md](roadmap.md), que es la Ãºnica fuente. Este documento no reserva nÃºmeros de sprint futuros.

> Antes habÃ­a aquÃ­ una tabla de roadmap propia. Se consolidÃ³ el 2026-09-21: las copias por sprint divergieron entre sÃ­ y llegaron a reclamar el mismo nÃºmero de sprint para features distintas.

