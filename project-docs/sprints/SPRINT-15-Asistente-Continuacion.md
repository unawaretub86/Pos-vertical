# SPRINT-15 — Continuación del Asistente y Deuda UTC

**Objetivo:** Completar el caso de uso del asistente añadiendo búsqueda de inventario y periodos por "quincenas", y corregir el bug de fechas UTC/Local que causa colapsos silenciosos en los reportes (detectado en QA y en el roadmap).

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/roadmap.md`, QA de US-061.

**Deuda técnica entrante:**
- US-034 (Factus), SPRINT-05 (Mobile), SPRINT-09, SPRINT-12 y SPRINT-13 (`dining_tables`) **siguen abiertas** y aparcadas.
- R-06 (Volver semiabierto el `to` de `ExpenseRepository.List`) **sigue abierta**.

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-066 | Bugfix UTC/Local: corregir colapso en SalesSummary (R-48) y desfase de fechas (R-05) | [ ] Pendiente | — |
| US-067 | Asistente: Intent PRODUCT_LOOKUP ("¿cuánto queda de arroz?") (R-02) | [ ] Pendiente | — |
| US-068 | Asistente: Referencia temporal "la quincena" (R-03) | [ ] Pendiente | — |
| US-069 | Correcciones UI: 404 Expenses, Setup permissions, static params y CSS scroll | [ ] Pendiente | — |

**Esfuerzo total estimado: 4 días** (Sprint corto, ideal para limpiar la deuda post-QA antes de saltar a mesas/facturación).

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-066 | El bug de fechas afecta los reportes, la base entera de cálculos y los rangos de fechas. Resolverlo primero quita ambigüedades. |
| **2** | US-067 | Amplía la utilidad del asistente para un caso de uso hiper-frecuente. Requiere interactuar con la DB. |
| **3** | US-068 | Incorpora un nuevo periodo ("quincena") que requiere ajustes aislados en `period.go`. |
| **4** | US-069 | Paquete de arreglos de Interfaz y Permisos (bloqueantes para UX/Despliegue). |

---

## US-066 — Bugfix UTC/Local y colapso de SalesSummary

**Como** desarrollador,
**quiero** que SQLite guarde y procese las fechas asumiendo un mismo formato y zona horaria (Local vs UTC),
**para** que no haya discrepancias entre `dashboard.go` y los reportes (bug de 19:00 hrs en Colombia) y no colapsen en un solo bucket `period=""`.

**Esfuerzo estimado:** 1 día. **Impacto:** Alto. Corrige dos defectos críticos de reportes y cálculos base de rangos (R-05, R-48).

**Estado verificado:**
- `internal/database/database.go:14`: La conexión SQLite se abre sin especificar formato de fecha (modernc.org/sqlite soporta pragma y querystring args).
- `internal/repository/report.go:113-127`: `SalesSummary` usa `COALESCE(strftime(...), '')` causando colapso silencioso de ventas de múltiples días en un solo bucket si el fixture lo ingresa como `time.Time` literal.
- `internal/repository/metrics.go:335`: Aging excluye fechas si son nulas o en un parseo incorrecto.

**Regla de negocio:**
```text
Todos los timestamps de SQLite (incluido CURRENT_TIMESTAMP) deben interpretarse en la zona horaria local.
Eliminar el COALESCE de report.go tras corregir el parseo de SQLite desde la conexión.
```

**Acceptance Criteria:**
- [ ] Reproducir el bug (R-05) con un test usando un timestamp pasadas las 19:00 hrs (UTC-5) comprobando que no se mueva erróneamente de día.
- [ ] Agregar la configuración necesaria al DSN de `internal/database/database.go:14` (e.g., parseTime, locales o lo que pida el driver `modernc.org/sqlite`).
- [ ] Validar que 3 ventas en 3 días distintos generan exactamente 3 grupos en `SalesSummary`, no 1 vacío (R-48).
- [ ] Los test `dashboard_test.go` y `metrics_test.go` pasan con el cambio.

**DoD:**
- [ ] Sin migración.
- [ ] `internal/database/database.go` y `internal/repository/report.go` ajustados.
- [ ] Test con assert del D-5 escrito.
- [ ] Build limpio (`go build ./...`).

---

## US-067 — Asistente: Intent PRODUCT_LOOKUP

**Como** cajero o dueño,
**quiero** preguntarle al asistente "¿cuánto queda de X?" o "¿precio de X?",
**para** consultar el inventario rápido sin buscar en el catálogo visualmente.

**Esfuerzo estimado:** 1.5 días. **Impacto:** Medio-Alto. Permite al usuario consultar su data primaria (inventario). Requiere búsqueda por similitud en el catálogo.

**Estado verificado:**
- `internal/assistant/engine.go:13-25`: `PRODUCT_LOOKUP` no existe entre los Intents actuales.
- Existe `ProductRepository.Search(ctx, query)` para buscar el producto que el usuario mencione.

**Contexto técnico:**
Nuevo Intent `IntentProductLookup = "PRODUCT_LOOKUP"`. Modifica `engine.go` (banco de regex/fuzzy). Extracción de entidad: aislar el objeto directo ("arroz" en "cuanto arroz queda"), luego pasarlo a `ProductRepository.Search`.

**Acceptance Criteria:**
- [ ] Intent resuelve correctamente ante "cuánto [x] queda", "precio de la [x]", "tengo [x]", etc.
- [ ] Si `Search` encuentra múltiples matches (e.g. 5 tipos de arroz), la respuesta lista el precio/stock de los primeros 3 y dice "...y 2 opciones más".
- [ ] Si `Search` no encuentra nada, responde explícitamente que no se encontró el producto en inventario.
- [ ] Cualquier usuario (`admin` y `cashier`) puede usar este Intent (modificando la capa de permisos de US-061 si es necesario).

**DoD:**
- [ ] Toca `assistant.go`, `engine.go`.
- [ ] Nuevos test cases unitarios en `engine_test.go` para los fixtures del banco de palabras.
- [ ] Build limpio.

---

## US-068 — Asistente: Referencia temporal "la quincena"

**Como** dueño,
**quiero** preguntar "cuánto fiaron esta quincena",
**para** ver mi periodo real de cobro en Colombia (del 1 al 15, o del 16 a fin de mes).

**Esfuerzo estimado:** 0.5 días. **Impacto:** Medio. Referencia clave muy usada en el contexto del negocio local.

**Estado verificado:**
- `internal/assistant/period.go:21`: `temporalWords` no incluye "quincena".
- `internal/assistant/period.go`: no hay cálculo de rangos para "esta quincena" ni "la quincena pasada".

**Regla de negocio:**
```text
Quincena 1 (Q1): día 1 al día 15 del mes. Rango [1, 16)
Quincena 2 (Q2): día 16 al último día del mes. Rango [16, 1 del sig. mes)

"Esta quincena" evalúa en qué quincena estamos hoy, y devuelve ese rango.
"Quincena pasada" devuelve la quincena inmediatamente anterior.
```

**Acceptance Criteria:**
- [ ] "quincena" agregado a `temporalWords`.
- [ ] Resolver "esta quincena" para hoy = día 10 → rango del 1 al 15 del mes actual.
- [ ] Resolver "esta quincena" para hoy = día 20 → rango del 16 a fin del mes actual.
- [ ] Resolver "quincena pasada" correctamente cruzando límite de mes (e.g. hoy 4 de Abril -> quincena pasada es 16-31 de Marzo).
- [ ] Tests en `period_test.go` validados.

**DoD:**
- [ ] Modificado `period.go`.
- [ ] Fixtures en `period_test.go` para los 4 límites de mes.
- [ ] Build limpio.

---

## US-069 — Correcciones UI: 404 Expenses, Setup permissions, static params y CSS scroll

**Como** usuario y administrador,
**quiero** que la interfaz de la aplicación no tenga errores de ruteo y restrinja los accesos,
**para** poder operar fluidamente y sin fallos técnicos.

**Esfuerzo estimado:** 1 día. **Impacto:** Alto (Correcciones de QA).

**Estado verificado:**
1. Al crear un gasto (POST `/expenses`) en `/reports`, retorna 404 en el entorno Next.js.
2. Build fallido: `Page "/customers/[id]/page" is missing param "/customers/[id]" in "generateStaticParams()"`.
3. Label `Ventas del período` en reportes no es descriptivo.
4. El cajero (`cashier`) puede entrar a `GET /setup` porque el backend no exige `RequireAdmin`.
5. En `/settings`, la configuración de la tienda no tiene scroll (le falta `overflow-y-auto` en el contenedor principal).

**Acceptance Criteria:**
- [ ] La creación de gastos funciona (resolver el 404 verificando el cliente API y CORS o trailing slashes).
- [ ] La compilación estática (`output: export`) es exitosa agregando `generateStaticParams()` con retorno `[]` en `app/customers/[id]/page.tsx`.
- [ ] El título de ventas en el reporte es claro y descriptivo (ej: "Total vendido en el periodo").
- [ ] Un usuario con rol `cashier` no puede acceder a `/setup`. `handler.go` protege `GET /setup` con `RequireAdmin` y el frontend lo expulsa o no le muestra la pestaña.
- [ ] El contenedor en `/settings` tiene scroll vertical funcional (`overflow-y-auto`) para que se vean todas las opciones.

**DoD:**
- [ ] `next build` finaliza correctamente sin errores de "missing param".
- [ ] Se verifica manualmente cada corrección.
