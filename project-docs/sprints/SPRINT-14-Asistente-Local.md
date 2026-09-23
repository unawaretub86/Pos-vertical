# SPRINT-14 — Asistente Local de Consultas

**Objetivo:** Que el dueño le escriba al POS en español de tienda ("¿cuánto gané hoy?", "¿quién me debe?", "¿qué es lo que más vendo?") y reciba la respuesta al instante, sin internet, sin instalar nada, sin modelo de IA — leyendo las mismas métricas que ya calcula el dashboard.

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/better-finds-pos.md` (E2), `project-docs/SPRINT-13-Mesas-Pedidos-Barra.md`

**Deuda técnica entrante:**
- US-034 — Factus: validación en sandbox (abierta desde SPRINT-07, requiere acceso externo) — **sigue abierta**, sin acción en este sprint.
- SPRINT-05 — Mobile Responsive (⬜ en la tabla de progreso) — **sigue abierta**. El widget de este sprint nace responsive; el resto de la app no se toca.
- US-042c, US-043, US-044, US-045 (SPRINT-09) — **siguen abiertas**. US-044 (sugerencia de pedido) y US-042c (velocidad de venta) habilitarían dos intents más del asistente; ver Roadmap.
- US-053 — Resumen de cierre de caja por correo (SPRINT-12) — **sigue abierta**. Hay trabajo sin commitear en `internal/mailer/`; no entra aquí.
- US-054 a US-058 (SPRINT-13, módulo restaurante) — **siguen abiertas**. Este sprint no depende de ellas y no las toca. **Ver el aviso de numeración de migración más abajo.**

> [!IMPORTANT]
> **Numeración de migración — depende del orden de ejecución.**
> SPRINT-13 (US-055) reserva la **migración 15**, pero SPRINT-13 está ⬜ Pendiente. La última migración realmente aplicada es la **14** (`internal/database/migrate.go:38`, `const migration14` en la línea 482).
> - Si SPRINT-13 se ejecuta antes que este sprint → la migración de US-061 es la **16**.
> - Si este sprint se ejecuta primero → es la **15**, y US-055 pasa a la 16.
>
> El dev verifica `internal/database/migrate.go` antes de escribir la migración y usa el siguiente número libre. En este documento se la nombra **migración N**.

---

## Decisión de alcance — por qué no hay LLM

`better-finds-pos.md:69` ya lo dictamina: un modelo cuantizado necesita 8GB+ y el parque típico del cliente colombiano es de 4GB. Adicionalmente, **el requisito del dueño del producto es que no haya nada que instalar ni descargar en el PC del cliente** — eso descarta también la vía Ollama.

Este sprint entrega **T1 (regex) + T2 (fuzzy matching)**, Go puro, cero dependencias nuevas, cero peso extra en el binario, respuesta en menos de 200 ms en cualquier máquina.

**Regla de arquitectura que se mantiene aunque después entre un LLM:** el motor de lenguaje **traduce, nunca calcula**. Pregunta → `{intent, periodo}` → consulta SQL determinista → plantilla en Go. Ninguna cifra que vea el usuario sale de heurística de texto. Un número inventado en un POS destruye la confianza en todo el sistema.

**Posicionamiento:** en V1 esto NO se vende como "IA". Se vende como *"Asistente que entiende tus preguntas"*. El claim "IA local 100% offline" queda reservado para cuando exista un tier de modelo real (ver Roadmap).

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-059 | Extraer las métricas del dashboard a un repositorio parametrizable por rango | [x] Completo | 2026-09-23 |
| US-060 | Motor de intenciones T1 (regex) + T2 (fuzzy) en Go puro | [ ] Pendiente | — |
| US-061 | Servicio, endpoint `/ai/ask` y bitácora de consultas | [ ] Pendiente | — |
| US-062 | Widget de chat en la app autenticada | [ ] Pendiente | — |
| US-063 | Corregir el saldo de la cartera envejecida del dashboard (D-4) | [x] Completo | 2026-09-23 |
| US-064 | Corregir `TotalPortfolioBalance`: no restar los saldos a favor (R-44) | [x] Completo | 2026-09-23 |

**Esfuerzo total estimado: 6.5 días** (sin contar US-064, que viene del roadmap R-44 y no estaba presupuestada en el sprint original). Dentro del objetivo de ~7. Línea de corte si aprieta: US-062 se reduce a una caja de búsqueda en el dashboard en vez de un widget flotante con historial (−0.5 día). No recortar US-059: sin ella los intents solo responden "hoy" y el asistente queda cojo.

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-059 | Refactor puro sin feature visible. Desbloquea los 11 intents y de paso quita 170 líneas de SQL inline del handler del dashboard. Se hace primero porque todo lo demás lo consume. |
| **2** | US-060 | Paquete aislado, sin DB, sin HTTP, sin frontend. Es donde vive el riesgo real (¿entiende cómo habla un tendero?) y es 100% testeable en aislamiento. Se ataca temprano. |
| **3** | US-061 | Pega US-059 con US-060 y expone el endpoint. Trae la migración. |
| **4** | US-062 | Consume un endpoint ya funcional y probado con curl. La última porque es la única recortable sin dejar el backend a medias. |

---

## US-059 — Extraer las métricas del dashboard a un repositorio parametrizable por rango

**Como** desarrollador,
**quiero** que las métricas del negocio se calculen en un repositorio que reciba un rango de fechas,
**para** que el dashboard y el asistente lean exactamente el mismo número y no haya dos versiones de "cuánto vendí".

**Esfuerzo estimado:** 1.5 días. **Impacto:** Alto. Sin esto el asistente solo sabe responder "hoy", y cualquier discrepancia entre el chat y el dashboard mata la credibilidad de la feature completa.

**Estado verificado:**
- `getDashboardToday` (`internal/handler/dashboard.go:48-220`) tiene **8 queries SQL escritas inline en el handler**, todas con la ventana de hoy calculada a mano en las líneas 52-55 (`todayStart` / `todayEnd`). No hay repositorio intermedio.
- Métricas ya calculadas ahí: vendido, fiado nuevo, fiado cobrado, efectivo estimado, producto más vendido, ganancia bruta, conteo de productos sin costo, gastos del día, ganancia neta, top 5 por margen, top 5 bajo stock, top 5 cartera envejecida (`todayResponse`, línea 33).
- `ReportRepository` (`internal/repository/report.go:51`) sí recibe rango, pero solo expone `SalesSummary` y `TopProducts`. `SalesSummary` (línea 30) ya trae `TotalRevenue`, `TotalCost`, `GrossMargin`, `TotalCount` y `ByPaymentMethod`.
- Cartera: no está en ningún repositorio. `getPortfolio` (`internal/handler/report.go:14`) trae **todos** los clientes con `h.customers.List(ctx, "")`, filtra en Go los que tienen `Balance > 0` y los ordena con un insertion sort escrito a mano (líneas 44-49).
- Gastos: `ExpenseRepository.SumRange(ctx, from, to)` ya existe y ya recibe rango (`internal/repository/expense.go:26`). No se toca.
- Patrón de repositorio del proyecto: interfaz + struct privada + constructor `NewXRepository(db *sql.DB)`, cableado en `handler.New` (`internal/handler/handler.go:38`).
- Fixture de tests: `setupTestDB(t)` con `database.New(":memory:")` (`internal/handler/dashboard_test.go:14`), seed por `exec()` de INSERTs literales.

**Contexto técnico:**
Nuevo `internal/repository/metrics.go` + `metrics_test.go`. Las 8 queries de `dashboard.go` se mueven tal cual, cambiando `todayStart`/`todayEnd` por los parámetros `from`/`to`. `getDashboardToday` queda como orquestador: calcula la ventana de hoy y llama al repo. **Los números que devuelve el dashboard no cambian** — es refactor, no rediseño; los tests existentes de `dashboard_test.go` deben pasar sin tocarlos, y eso es la red de seguridad de esta US.

Se agrega también `Debtors(ctx, limit)` al nuevo repo, con la query de cartera hecha en SQL (`ORDER BY balance DESC LIMIT ?`) en vez del filtrado + insertion sort en Go. `getPortfolio` se reescribe encima de ese método. `MetricsRepository` se cablea en `Handler` como `metrics repository.MetricsRepository`.

**Regla de negocio:**
```
Convención de rango: SIEMPRE semiabierto [from, to)
  hoy  = [00:00 de hoy, 00:00 de mañana)
Igual que la ventana actual del dashboard (dashboard.go:52-55) y que
ExpenseRepository.SumRange. Ningún BETWEEN, que es inclusivo y duplica
las ventas de medianoche.

Zona horaria: hora local del servidor (time.Now().Location()), como ya
hace el dashboard. El POS corre en la máquina de la tienda.

Definiciones que NO cambian respecto al dashboard actual:
  vendido        = SUM(sales.total) WHERE status='completed'
  ganancia bruta = SUM((si.unit_price - p.cost) * si.quantity) solo p.cost > 0
  ganancia neta  = ganancia bruta - gastos del rango
  efectivo       = pagos cash del rango - movimientos cash_out del rango
  margen %       = (GrossMargin / TotalRevenue) * 100, y 0 si TotalRevenue = 0
  cartera        = SUM(pagos credit) - SUM(abonos), solo balance > 0
```

**Scope — qué NO entra:**
- Agrupación por hora del día y por vendedor (G8 de `better-finds-pos.md:163`). No existe hoy y no la necesita ningún intent de este sprint.
- Cachear métricas. Las queries son sobre SQLite local; medir antes de optimizar.
- Cambiar el contrato JSON de `/api/dashboard/today` o `/api/reports/portfolio`. El frontend no se toca en esta US.

**Acceptance Criteria:**

**Refactor sin regresión:**
- [ ] `GET /api/dashboard/today` devuelve exactamente el mismo JSON que antes del refactor (mismos campos, mismos valores)
- [ ] Los tests existentes de `internal/handler/dashboard_test.go` pasan sin modificarse
- [ ] `GET /api/reports/portfolio` devuelve la misma lista, ordenada por saldo descendente
- [ ] No queda SQL inline en `internal/handler/dashboard.go`

**Repositorio nuevo:**
- [ ] Todos los métodos aceptan `from, to time.Time` salvo los que son un estado puntual (bajo stock, cartera)
- [ ] Con un rango sin ventas, cada método devuelve cero o slice vacío, nunca `nil` ni error

**DoD:**
- [ ] Sin migración (refactor puro)
- [ ] `internal/repository/metrics.go` con la interfaz `MetricsRepository`:
```go
type MetricsRepository interface {
    Sold(ctx, from, to) (total float64, count int, err error)
    Profit(ctx, from, to) (gross, expenses, net float64, unconfiguredCost int, err error)
    CashFlow(ctx, from, to) (cashSales, cashOut float64, err error)
    CreditFlow(ctx, from, to) (newCredit, collected float64, err error)
    TopByMargin(ctx, from, to, limit) ([]MarginItem, error)
    LowStock(ctx, limit) ([]LowStockItem, error)
    Debtors(ctx, limit) ([]Debtor, error)
}
```
- [ ] `getDashboardToday` y `getPortfolio` reescritos sobre esos métodos; sin cambios de contrato HTTP
- [ ] `MetricsRepository` cableado en `handler.New` (`internal/handler/handler.go:38`)
- [ ] Test: 2 ventas hoy ($28.000 y $12.000) + 1 venta ayer ($50.000) → `Sold(hoy)` = 40.000 / 2 tickets; `Sold(ayer)` = 50.000 / 1 ticket
- [ ] Test: venta a las 23:59:59 de hoy entra en el rango de hoy; venta a las 00:00:00 de mañana NO entra (verifica el semiabierto)
- [ ] Test: producto con `cost = 0` no suma a la ganancia bruta pero sí incrementa `unconfiguredCost`
- [ ] Test: `Debtors(5)` con 3 clientes (saldos 80.000, 150.000 y 0) → devuelve 2 clientes, el de 150.000 primero
- [ ] Test: rango vacío (mañana) → `Sold` devuelve 0 / 0 sin error
- [ ] Build limpio (`go build ./...`, `pnpm build` en `frontend/`)

### Criterios verificados

> Cerrada el 2026-09-23 (QA vuelta 1: OK, 0 bloqueantes). Los checkboxes de arriba son el enunciado original. Lo que se verificó es esto, con las decisiones del PO en GATE 1: se amplió la interfaz con `TopSeller` y `AgingReceivables`; "sin SQL inline" aplica solo a `getDashboardToday`; el saldo de la cartera envejecida se preserva con su defecto actual (D-4, roadmap R-04); portfolio vacío devuelve `[]`; un error de DB responde 500.
> Tests: `internal/repository/metrics_test.go` (repo) e `internal/handler/metrics_equivalence_test.go` (handler).

| AC | Criterio (EARS) | Tests |
|---|---|---|
| AC-1 | CUANDO un admin llama `GET /api/dashboard/today` y la DB responde sin error, el sistema DEBE devolver el mismo JSON que antes del refactor (13 campos, mismos valores y orden, `top_by_margin` vacío con menos de 2 productos con costo, listas `[]` y nunca `null`). | `TestDashboardToday_T20_Golden`, `TestDashboardToday_T21_TopByMarginNeedsTwo`, `TestDashboardToday_T22_AgingReceivables` (pasan también contra `dev`) |
| AC-2 | CUANDO se corre `go test`, los tests de `internal/handler/dashboard_test.go` DEBEN pasar sin modificar el archivo. | `TestGetDashboardToday_GrossProfit`, `TestGetDashboardToday_NoSales`; diff vacío (QA, T-25) |
| AC-3 | CUANDO un admin llama `GET /api/reports/portfolio`, el sistema DEBE devolver `{id, name, phone, balance, oldest_debt_at}` de los clientes con `balance > 0`, ordenados por saldo DESC y nombre ASC. | `TestPortfolio_T23_OrderAndFields` |
| AC-4 | MIENTRAS no haya deudores, `/reports/portfolio` DEBE devolver `data: []`. | `TestPortfolio_T24_EmptyIsArray` |
| AC-5 | `getDashboardToday` NO DEBE contener SQL; solo calcula la ventana `[hoy 00:00, mañana 00:00)` local y llama a `MetricsRepository`. | Revisión de QA (T-26): `grep` sin coincidencias en `getDashboardToday` |
| AC-6 | El sistema DEBE exponer `MetricsRepository` en `internal/repository/metrics.go` con `NewMetricsRepository(db)`, cableado en `handler.New` como `metrics`. | Revisión de QA (T-27): `handler.go:29,49,81` |
| AC-7 | CUANDO se invoca un método de flujo con `from, to`, el sistema DEBE filtrar `created_at >= from AND created_at < to` con las mismas definiciones SQL del dashboard. | `TestMetricsRepo_T1_SoldTodayAndYesterday`, `_T2_SoldHalfOpenMidnight`, `_T3_SoldIgnoresVoided`, `_T4_Profit`, `_T6_CashFlow`, `_T7_CreditFlow`, `_T8_TopByMargin`, `_T9_TopByMarginSingleItem`, `_T11_LowStock` |
| AC-8 | CUANDO se invoca `Debtors(limit)`, el sistema DEBE usar la fórmula de saldo de `customer.go`, filtrar `balance > 0`, ordenar por saldo DESC y nombre ASC en SQL, y aplicar el límite (`<= 0` = todos). | `TestMetricsRepo_T12_Debtors`, `_T13_DebtorsLimit`, `_T14_DebtorsTieByName`, `_T15_DebtorsIgnoresVoidedCredit` |
| AC-9 | CUANDO se invoca `TopSeller(from, to)`, el sistema DEBE devolver el producto con mayor cantidad vendida, o `""`/0 sin error si no hay ventas. | `TestMetricsRepo_T10_TopSeller`, `_T18_EmptyRange` |
| AC-10 | CUANDO se invoca `AgingReceivables(limit)`, el sistema DEBE devolver la misma lista que el dashboard anterior, **incluido el defecto D-4** (decisión del PO). | `TestMetricsRepo_T16_AgingReceivables`, `_T17_AgingReceivablesPreservesD4`, `TestDashboardToday_T22_AgingReceivables` |
| AC-10b | MIENTRAS una fila de aging tenga `days_overdue` NULL, el SQL DEBE excluirla explícitamente; los errores de scan se propagan. | `TestMetricsRepo_T30_AgingExcludesNullDaysOverdue` |
| AC-11 | CUANDO se invoca `Profit`, `expenses` DEBE ser igual a `ExpenseRepository.SumRange` del mismo rango y `net = gross − expenses`. | `TestMetricsRepo_T4_Profit`, `_T5_ProfitExpensesMatchSumRange` |
| AC-12 | MIENTRAS el rango no tenga datos, cada método DEBE devolver 0 / `""` y slices vacíos no-nil, sin error. | `TestMetricsRepo_T18_EmptyRange` |
| AC-13 | SI una query falla, ENTONCES el método DEBE devolver el error envuelto con `%w`. | `TestMetricsRepo_T19_ClosedDBWrapsErrors` |
| AC-14 | El cambio NO DEBE agregar migraciones (última: 14). | Revisión de QA (T-27): `migrate.go:38` |
| AC-15 | `go build`, `go test`, `pnpm build` y `pnpm test` DEBEN pasar. | Ejecución de QA: todos exit 0. `pnpm lint` falla por 41 problemas anteriores en `frontend/`, que no toca esta US (roadmap R-35) |
| AC-16 | SI `getDashboardToday` recibe error del repositorio, ENTONCES DEBE responder 500 `"error obteniendo métricas"` sin cifras. | `TestDashboardToday_T28_DBErrorReturns500` |
| AC-17 | SI `Debtors` falla, ENTONCES `getPortfolio` DEBE responder 500 `"error obteniendo cartera"` (no-regresión). | `TestPortfolio_T29_DBErrorReturns500` |

---

## US-060 — Motor de intenciones T1 (regex) + T2 (fuzzy) en Go puro

**Como** Don Carlos (dueño),
**quiero** escribir la pregunta como me salga, con tildes o sin ellas y con errores de dedo,
**para** no tener que aprenderme comandos ni recordar en qué pantalla está cada dato.

**Esfuerzo estimado:** 2 días. **Impacto:** Alto. Es el corazón de la feature y el único punto donde el resultado puede sentirse tonto. Todo lo demás es plomería.

**Estado verificado:**
- No existe nada de esto: `grep -rn "intent\|Intent" internal/` no devuelve resultados relacionados.
- No hay dependencias de NLP en `go.mod`. Esta US no agrega ninguna: `regexp`, `strings` y `unicode` son librería estándar.
- El proyecto ya tiene tests de lógica pura con tablas de casos, p. ej. `frontend/lib/saleCalc.test.ts` y `internal/service/*_test.go`. Se sigue ese patrón.

**Contexto técnico:**
Paquete nuevo `internal/assistant/` con `engine.go`, `normalize.go`, `period.go` y sus tests. **Cero imports de `database/sql`, `net/http` o de los repositorios.** Entra un string, sale un `Query{Intent, Period, Confidence, Tier}`. Esa pureza es lo que permite probar 60 frases reales en milisegundos y lo que deja la puerta abierta a enchufar un tier de modelo después sin tocar nada más.

Pipeline: `normalize → T1 regex → T2 fuzzy → unknown`. El extractor de periodo corre aparte del de intención, sobre la misma cadena normalizada.

**Regla de negocio:**
```
NORMALIZACIÓN (obligatoria antes de cualquier match)
  1. minúsculas
  2. quitar tildes y diéresis (á→a, é→e, í→i, ó→o, ú→u, ü→u)  — la ñ SE CONSERVA
  3. quitar signos ¿ ? ¡ ! , . ;
  4. colapsar espacios múltiples y recortar extremos
Efecto: los patrones se escriben sin alternativas [aá]; "¿CUÁNTO VENDÍ?" y
"cuanto vendi" son la misma cadena.

T1 — REGEX (confianza 1.0)
  Patrones explícitos por intent sobre la cadena normalizada.
  Primer patrón que coincide, gana. Orden de evaluación fijo y declarado
  en el registro de intents (ver más abajo): sin orden fijo, "cuanto gane
  vendiendo" cae en SALES o PROFIT según el humor del mapa.

T2 — FUZZY (confianza 0.0–0.99)
  Solo corre si T1 no coincidió.
  Cada intent trae un banco de frases semilla. Para cada frase:

    tokens        = campos de la cadena, descartando palabras vacías
                    (de, la, el, los, las, mi, me, un, una, que, en, por, y)
    coincide(t,s) = t == s  OR  levenshtein(t,s) <= 1 y len(t) >= 4
    score         = (tokens que coinciden) / (tokens de la frase semilla)

  score del intent = el mayor score entre sus frases semilla
  gana el intent de mayor score
  UMBRAL = 0.60. Por debajo → UNKNOWN.
  Empate a menos de 0.10 de diferencia entre dos intents → AMBIGUO,
  se devuelven los dos candidatos para que el usuario elija.

PERIODO (independiente del intent)
  "hoy" | sin mención     → [00:00 hoy, 00:00 mañana)
  "ayer"                  → [00:00 ayer, 00:00 hoy)
  "esta semana"           → [lunes 00:00, ahora)        semana inicia LUNES
  "semana pasada"         → [lunes previo, lunes actual)
  "este mes"              → [día 1 00:00, ahora)
  "mes pasado"            → [día 1 mes previo, día 1 mes actual)
  "este ano" / "este año" → [1 de enero 00:00, ahora)
  Rango siempre semiabierto [from, to), igual que US-059.

  PROHIBIDO EL FALLBACK SILENCIOSO A "HOY".
  "Sin mención de tiempo" significa que NINGUNA palabra de la frase se parece
  a una referencia temporal. Es distinto de "había una palabra de tiempo y no
  la entendí", y confundirlas es el peor error posible del motor: el usuario
  pregunta por ayer, recibe la cifra de hoy, y nada en la respuesta le avisa.

  El extractor de periodo aplica el mismo fuzzy de T2 contra el vocabulario
  temporal (hoy, ayer, semana, mes, ano, pasada, pasado, esta, este):
    coincide dentro de levenshtein <= 1  → resuelve ese periodo
      "aller" → ayer   ·   "oy" → hoy   ·   "mez pasado" → mes pasado
    hay una palabra candidata a tiempo que no resuelve
                                        → Period.Unresolved = true
    ninguna palabra candidata a tiempo  → periodo hoy, normal

  Con Period.Unresolved = true el servicio NO responde con cifras. Devuelve:
    "¿De qué fecha? Puedo darte hoy, ayer, esta semana o este mes."
  más esos cuatro periodos como chips.

  Toda respuesta con cifras nombra su periodo en el texto ("Hoy…", "Ayer…",
  "Este mes…"), de modo que un periodo mal interpretado sea visible al leer.

COMPARACIÓN (transversal, no es un intent)
  Si la frase contiene "mas que|menos que|comparado|vs|que ayer|que el mes pasado"
  → Query.Compare = true. El servicio resuelve el mismo intent dos veces:
  el periodo pedido y el inmediatamente anterior del mismo tamaño.
```

**Los 11 intents:**

| # | Intent | Responde a | Datos que consumirá (US-061) |
|---|--------|-----------|------------------------------|
| 1 | `SALES` | "cuánto vendí", "cuánto facturé", "cómo van las ventas" | `Sold` |
| 2 | `PROFIT` | "cuánto gané", "cuánta ganancia", "cuánto me quedó limpio" | `Profit` |
| 3 | `MARGIN` | "cuál es mi margen", "qué porcentaje gano" | `SalesSummary.GrossMargin` |
| 4 | `TOP_PRODUCTS` | "qué es lo que más vendo", "cuál es mi producto estrella" | `TopProducts` + `TopByMargin` |
| 5 | `TOP_MARGIN` | "qué me deja más plata", "en qué gano más" | `TopByMargin` |
| 6 | `LOW_STOCK` | "qué se me está acabando", "qué tengo que pedir" | `LowStock` |
| 7 | `DEBTORS` | "quién me debe", "cómo va la cartera", "a quién le fié" | `Debtors` |
| 8 | `EXPENSES` | "cuánto he gastado", "cuánto llevo en gastos" | `ExpenseRepository.SumRange` |
| 9 | `PAYMENT_MIX` | "cuánto me entró por Nequi", "cuánto en efectivo" | `SalesSummary.ByPaymentMethod` |
| 10 | `CASH` | "cuánta plata hay en caja", "cuánto efectivo tengo" | `CashFlow` |
| 11 | `CREDIT_FLOW` | "cuánto fié hoy", "cuánto me abonaron" | `CreditFlow` |

Más `UNKNOWN` y `AMBIGUOUS`, que no son intents de negocio sino estados del motor.

**Scope — qué NO entra:**
- Búsqueda de un producto puntual ("¿cuánto queda de arroz?"). Requiere extraer la entidad del texto y resolverla contra el catálogo; es una US propia. El tipo `Query` nace con un campo `Entity string` sin usar para no rediseñarlo después.
- Preguntas encadenadas o memoria de la conversación ("¿y ayer?"). Cada pregunta es independiente.
- Cualquier modelo de lenguaje.

**Acceptance Criteria:**

**Normalización:**
- [ ] "¿CUÁNTO VENDÍ HOY?" y "cuanto vendi hoy" producen el mismo resultado
- [ ] La ñ sobrevive: "este año" no se convierte en algo que rompa el patrón

**T1 — frases directas:**
- [ ] Las 11 intenciones se resuelven con su frase canónica y con al menos 2 variantes cada una
- [ ] La confianza reportada es 1.0 y el tier es `regex`

**T2 — frases torcidas:**
- [ ] "kien me debe plata" → `DEBTORS`
- [ ] "cuanto bendi oy" → `SALES` con periodo hoy
- [ ] "ke se esta acabando" → `LOW_STOCK`
- [ ] "en ke gano mas" → `TOP_MARGIN`
- [ ] El tier reportado es `fuzzy` y la confianza es menor a 1.0

**Periodo:**
- [ ] "cuánto vendí" sin mención de fecha → periodo hoy
- [ ] "cuánto vendí ayer" → periodo ayer
- [ ] "cuánto vendí este mes" → desde el día 1 hasta ahora
- [ ] "cuánto vendí la semana pasada" → lunes a lunes previos
- [ ] "cuanto vendi aller" → periodo **ayer**, no hoy (fuzzy sobre el vocabulario temporal)
- [ ] "cuanto vendi el juebes" → `Period.Unresolved = true`, **nunca** periodo hoy
- [ ] "cuánto vendí" (sin palabra de tiempo) → periodo hoy con `Unresolved = false`

**Bordes:**
- [ ] "¿cómo hago un sándwich?" → `UNKNOWN`
- [ ] Cadena vacía o solo espacios → `UNKNOWN`, sin pánico
- [ ] Cadena de 5.000 caracteres → responde `UNKNOWN` sin colgarse
- [ ] Dos intents empatados dentro de 0.10 → `AMBIGUOUS` con los dos candidatos

**DoD:**
- [ ] Sin migración, sin endpoint, sin frontend
- [ ] Paquete `internal/assistant/` que no importa `database/sql`, `net/http` ni `internal/repository`
- [ ] `Parse(q string) Query` como única función exportada del motor, donde `Query{Intent, Period, Compare, Entity, Confidence, Tier}`
- [ ] Registro de intents con orden de evaluación explícito, para que agregar un intent sea una entrada más en la tabla
- [ ] Test de tabla con **mínimo 60 frases reales** cubriendo los 11 intents, con y sin tildes, con errores de dedo
- [ ] Test: `levenshtein("vendi","bendi") == 1` y `levenshtein("hoy","doy") == 1` pero "hoy" no hace fuzzy por tener menos de 4 caracteres
- [ ] Test: "ventas de aller" → detecta "ayer" por fuzzy o devuelve AMBIGUOUS, **nunca** devuelve "hoy"
- [ ] Test: "cuanto vendi hoy comparado con ayer" → `SALES`, periodo hoy, `Compare = true`
- [ ] Test: "cuanto vendi aller" → `SALES` con periodo **ayer**; "cuanto vendi el juebes" → `Unresolved = true`; "cuanto vendi" → periodo hoy. Los tres en el mismo caso de tabla, porque el bug es confundirlos entre sí
- [ ] Test: ninguna frase del banco semilla de un intent resuelve a un intent distinto (prueba de no-solapamiento entre bancos)
- [ ] `go test ./internal/assistant/...` corre en menos de 1 segundo
- [ ] Build limpio (`go build ./...`, `pnpm build`)

---

## US-061 — Servicio, endpoint `/ai/ask` y bitácora de consultas

**Como** Don Carlos (dueño),
**quiero** que la respuesta me llegue redactada en español con mis cifras reales,
**para** entenderla de un vistazo sin interpretar una tabla.

**Esfuerzo estimado:** 1.5 días. **Impacto:** Alto. Es lo que convierte el motor en una respuesta.

**Estado verificado:**
- No existe ninguna ruta `/ai`: no hay coincidencias de `ai` en las rutas de `internal/handler/handler.go`. Las rutas autenticadas se registran en el grupo que abre en `internal/handler/handler.go:100`.
- `/dashboard` y `/reports/*` están bajo `appmw.RequireAdmin` (`internal/handler/handler.go:157` y `164`). **El asistente devuelve ventas, ganancias, margen y cartera: los mismos datos.** Debe ir en un grupo `RequireAdmin`, no solo autenticado, o un usuario `cashier` obtiene por chat lo que la UI le niega.
- Formato de moneda en el frontend: `formatCOP` en `frontend/lib/format.ts`. En backend no hay equivalente; el servicio lo necesita para armar el texto.
- Última migración aplicada: **14** (`internal/database/migrate.go:38`). Ver el aviso de numeración al inicio del documento.

**Contexto técnico:**
`internal/service/assistant.go` recibe `MetricsRepository`, `ReportRepository`, `ExpenseRepository` y el motor de US-060. Resuelve el intent, consulta, y arma la respuesta con una plantilla por intent. `internal/handler/assistant.go` expone `POST /ai/ask` dentro de un grupo `RequireAdmin` nuevo, siguiendo el patrón del grupo de reportes.

La migración N crea `assistant_queries`: **no es historial de chat**, es la bitácora de lo que el motor NO entendió. Es el insumo que decide si vale la pena un tier de modelo más adelante y qué frases agregar al banco semilla. Sin ella, ampliar el asistente es adivinar.

**Regla de negocio:**
```
FORMATO DE MONEDA
  Pesos colombianos, sin decimales, separador de miles con punto: $1.234.567
  Se redondea al peso con math.Round. Nunca se muestran centavos.

LA PLANTILLA ES LA ÚNICA FUENTE DEL TEXTO
  El motor jamás genera ni redacta cifras. Selecciona intent y periodo;
  las cifras salen de SQL y se insertan en una plantilla fija de Go.

ENRIQUECIMIENTO DE TOP_PRODUCTS (volumen vs. margen)
  Si el producto #1 por unidades != el producto #1 por margen, la respuesta
  agrega una segunda línea de contraste:
    "Lo que más vendés es <A> (<n> unidades), pero lo que más plata te deja
     es <B> (<m>% de margen)."
  Si coinciden, o si no hay datos de margen (TopByMargin vacío porque
  ningún producto tiene costo configurado), se omite la línea.
  Es la respuesta que convierte el asistente en un consejo y no en un reporte.

COSTOS SIN CONFIGURAR
  Si unconfiguredCost > 0, toda respuesta de PROFIT, MARGIN o TOP_MARGIN
  agrega: "Ojo: <n> productos no tienen costo configurado, así que la
  ganancia real puede ser mayor."

LISTAS LARGAS EN EL CHAT
  Para intents que devuelven listas (DEBTORS, LOW_STOCK, TOP_PRODUCTS), la consulta SQL usa un LIMIT ajustado (ej. 6).
  Si la lista devuelta tiene más de 5 elementos, la plantilla muestra solo los 5 primeros y resume el resto con la frase "y <N> más".
  Esto evita romper la UI del widget de chat.

SIN DATOS
  Cada intent tiene su texto de rango vacío, nunca "$0":
    SALES   → "Todavía no hay ventas registradas hoy."
    DEBTORS → "Nadie te debe plata ahora mismo. Cartera en cero."

UNKNOWN
  Texto fijo + 3 sugerencias en el campo suggestions. Nunca una disculpa sola.

AMBIGUOUS
  "¿Te referís a las ventas o a la ganancia?" + los dos candidatos en suggestions.

BITÁCORA
  Se registra SIEMPRE: intent resuelto (incluido unknown), tier y confianza.
  Se guarda el texto de la consulta TAL CUAL lo escribió el usuario.
  No se guarda la respuesta ni ninguna cifra: la tabla es para aprender
  vocabulario, no para duplicar datos del negocio.
```

**Scope — qué NO entra:**
- Endpoint para leer la bitácora. En V1 se consulta con `sqlite3` a mano. Una pantalla de "qué me preguntan y no entiendo" es una US del próximo sprint.
- Respuestas en streaming. La respuesta es una consulta SQL, llega completa.
- Limitar la frecuencia de consultas. Es un binario local con un solo usuario.

**Acceptance Criteria:**

**Respuestas:**
- [ ] Los 11 intents devuelven texto redactado con cifras reales de la base
- [ ] Las cifras coinciden exactamente con las que muestra el dashboard para el mismo periodo
- [ ] `TOP_PRODUCTS` agrega la línea de contraste cuando el más vendido no es el más rentable
- [ ] Con productos sin costo configurado, `PROFIT` agrega la advertencia
- [ ] Un periodo sin datos devuelve el texto propio del intent, no "$0"
- [ ] `UNKNOWN` devuelve el texto fijo y 3 sugerencias
- [ ] Una consulta con `Compare = true` devuelve ambos periodos y la variación porcentual
- [ ] Toda respuesta con cifras nombra su periodo en el texto ("Hoy…", "Ayer…", "Este mes…")
- [ ] Una consulta con `Period.Unresolved = true` **no** devuelve ninguna cifra: pregunta la fecha y ofrece los cuatro periodos como chips

**Seguridad:**
- [ ] Sin token → 401
- [ ] Con token de un usuario `cashier` → 403 en todos los intents
- [ ] Con token de `admin` → 200

**Bitácora:**
- [ ] Toda consulta inserta una fila, incluidas las no entendidas
- [ ] Un fallo al insertar en la bitácora NO rompe la respuesta al usuario

**DoD:**
- [ ] Migración N (verificar el siguiente número libre en `internal/database/migrate.go`):
```sql
CREATE TABLE IF NOT EXISTS assistant_queries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id),
	query TEXT NOT NULL,
	intent TEXT NOT NULL,
	tier TEXT NOT NULL CHECK (tier IN ('regex','fuzzy','none')),
	confidence REAL NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_assistant_queries_intent ON assistant_queries(intent);
```
- [ ] `POST /api/ai/ask` (**admin**) → `{ intent, period: {from, to, label}, text, data, suggestions, tier, confidence }`
- [ ] `data` lleva el payload tipado del intent (lista de deudores, de productos, etc.), no solo el texto — el widget lo renderiza y un tier de modelo futuro lo redacta
- [ ] Registrado en un grupo `r.Use(appmw.RequireAdmin)` en `internal/handler/handler.go`, mismo patrón que el grupo de reportes de la línea 164
- [ ] Servicio y repos cableados en `handler.New` (`internal/handler/handler.go:38`)
- [ ] Test: seed de 2 ventas hoy ($28.000 y $12.000) → "cuanto vendi hoy" responde texto que contiene `$40.000` y `2`
- [ ] Test: seed de 6 clientes con deuda → "quien me debe" lista los 5 de mayor deuda y añade el texto "y 1 más"
- [ ] Test: seed donde el más vendido (90 unidades, 11% margen) difiere del más rentable (24 unidades, 38% margen) → la respuesta de `TOP_PRODUCTS` contiene ambos nombres
- [ ] Test: "¿cómo hago un sándwich?" → `intent = "unknown"`, 3 sugerencias, y se insertó 1 fila en `assistant_queries` con `tier = 'none'`
- [ ] Test: token de `cashier` → 403 y **no** se filtra ninguna cifra en el cuerpo de la respuesta
- [ ] Test: formato de moneda → 1234567.4 se renderiza `$1.234.567`
- [ ] Test: con ventas hoy y ayer distintas, "cuanto vendi el juebes" **no** devuelve ninguna de las dos cifras; responde preguntando la fecha
- [ ] Build limpio (`go build ./...`, `pnpm build`)

---

## US-062 — Widget de chat en la app autenticada

**Como** Don Carlos (dueño),
**quiero** preguntarle al POS desde cualquier pantalla sin dejar lo que estoy haciendo,
**para** resolver la duda en el momento en que me da, no cuando llego al dashboard.

**Esfuerzo estimado:** 1.5 días. **Impacto:** Medio-Alto. El backend ya sirve por API, pero sin esto nadie lo usa.

**Estado verificado:**
- `frontend/app/layout.tsx` es el layout raíz y envuelve **todas** las rutas, incluidas `/login`, `/setup` y `/onboarding`. Montar el widget ahí lo mostraría antes de iniciar sesión.
- El verdadero contenedor autenticado es `frontend/components/AppLayout.tsx`: verifica `isAuthenticated()`, redirige a `/login`, y solo renderiza sus hijos cuando `ready` es true. Ahí va el widget.
- `AppLayout` ya detecta móvil con `matchMedia('(max-width: 767px)')` y usa `usePathname()` — las dos cosas que el widget necesita para ocultarse.
- `frontend/app/pos/page.tsx` tiene 1.624 líneas y un modo móvil por pestañas (`mobileTab`). Un botón flotante en la esquina inferior derecha compite con la zona de cobro.
- `formatCOP` ya existe en `frontend/lib/format.ts`; el cliente HTTP es `api` en `frontend/lib/api.ts`.
- Estilos por variables CSS (`var(--canvas)`, `var(--surface-2)`, `var(--border)`, `var(--text-primary)`), con tema claro/oscuro vía `ThemeProvider`.

**Contexto técnico:**
Nuevo `frontend/components/AssistantWidget.tsx`, montado dentro de `AppLayout` al final del árbol, de modo que hereda la verificación de sesión y nunca aparece en pantallas públicas. Se oculta en `/pos` por `pathname`. Historial en estado de React: **efímero por diseño**, se pierde al recargar y no se persiste en ningún lado.

**Regla de negocio:**
```
VISIBILIDAD
  Se renderiza solo si el usuario es admin (getUser().role === 'admin'),
  porque el endpoint responde 403 a cashier: mostrar el botón a quien
  siempre recibirá un error es peor que no mostrarlo.
  Se oculta en /pos mediante CSS (`display: none` o `hidden` en Tailwind): esa pantalla es para vender, no para consultar. El componente NO se debe desmontar del DOM, para preservar el historial si el usuario regresa al dashboard.

ESTADO DEL HISTORIAL
  Vive en useState. No se persiste ni en localStorage ni en servidor.
  Recargar la página lo borra, y eso es intencional.

SUGERENCIAS
  Las suggestions de la respuesta se renderizan como chips pulsables que
  envían esa consulta al tocarlas. En el mensaje de bienvenida y en toda
  respuesta unknown siempre hay chips: son lo que le enseña al usuario
  qué puede preguntar. Un párrafo de disculpa se lee una vez; los chips
  se usan.
```

**Scope — qué NO entra:**
- Dictado por voz.
- Exportar o compartir la conversación.
- Mostrar el widget a usuarios `cashier`.

**Acceptance Criteria:**

**Widget:**
- [ ] Botón flotante en la esquina inferior derecha, visible en todas las pantallas autenticadas
- [ ] No aparece en `/login`, `/setup`, `/onboarding` ni `/change-password`
- [ ] Se oculta visualmente (pero sigue montado) al entrar en `/pos`, para no perder la conversación
- [ ] No aparece para un usuario con rol `cashier`
- [ ] Al abrir muestra el saludo y 3 chips de ejemplo
- [ ] Respeta tema claro y oscuro (variables CSS, ningún color fijo)

**Conversación:**
- [ ] Escribir y enviar muestra la pregunta y luego la respuesta
- [ ] Enter envía; el foco vuelve al input tras responder
- [ ] Mientras espera, se ve un indicador y el input queda deshabilitado
- [ ] Las respuestas con lista (deudores, bajo stock, top productos) se ven como lista, no como párrafo corrido
- [ ] Los montos usan `formatCOP`
- [ ] Los chips de sugerencia envían su consulta al tocarlos
- [ ] Error de red → "No pude consultar los datos. Intentá de nuevo." y el input se rehabilita
- [ ] Navegar entre pantallas autenticadas (ej. Dashboard -> Pos -> Dashboard) mantiene el historial
- [ ] Recargar la página explícitamente (F5) deja el historial vacío

**Responsive:**
- [ ] En 320px el panel ocupa la pantalla completa con botón de cerrar visible
- [ ] El teclado del celular no tapa el input

**DoD:**
- [ ] Sin migración, sin endpoint nuevo
- [ ] `frontend/components/AssistantWidget.tsx` montado en `frontend/components/AppLayout.tsx`, **no** en `frontend/app/layout.tsx`
- [ ] Consume `POST /api/ai/ask` con el cliente `api` de `frontend/lib/api.ts`
- [ ] Sin dependencias npm nuevas
- [ ] Verificación manual: preguntar "cuánto vendí hoy" y contrastar el número con la tarjeta del dashboard — deben coincidir
- [ ] Verificación manual: entrar como `cashier` y confirmar que el botón no existe
- [ ] Build limpio (`go build ./...`, `pnpm build`)

---

## US-063 — Corregir el saldo de la cartera envejecida del dashboard (D-4)

**Origen:** roadmap R-04 (defecto D-4, detectado y preservado a propósito en US-059).

**Como** Don Carlos (dueño),
**quiero** que la lista de "cartera envejecida" del dashboard muestre lo que cada cliente me debe de verdad,
**para** cobrarle la cifra correcta y no perder de vista a quien lleva más días debiendo.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Alto. Toca el diferenciador del producto: el mismo cliente aparece con un saldo en el dashboard y con otro en `/customers` y en `/reports/portfolio`, y un cliente con varios fiados y abonos puede desaparecer de la lista aunque deba.

**Estado verificado:**
- `MetricsRepository.AgingReceivables` (`internal/repository/metrics.go:314-331`) une `customer_payments` agregado por cliente a **cada fila** de `sale_payments` con `method='credit'` (`:322-326`) y hace `SUM(cp.paid)` (`:319`): los abonos se multiplican por la cantidad de líneas de fiado del cliente. El comentario de `:309-313` lo documenta como defecto preservado.
- `Debtors` (`metrics.go:263-285`) y `CustomerRepository.List` (`internal/repository/customer.go:24-39`) usan tablas derivadas agregadas por cliente y dan el saldo correcto.
- Ejecutado el 2026-09-23 (test temporal, borrado): Dora (fiados $25.000 + $25.000, abono $10.000) → aging **$30.000**, `Debtors` y `List` **$40.000**. Eli (3 fiados de $10.000, abonos $5.000 + $10.000) → debe $15.000 pero **no aparece** en aging (el defecto le calcula −$15.000).
- Los tests `TestMetricsRepo_T17_AgingReceivablesPreservesD4` (`internal/repository/metrics_test.go:536`, aserción `:558`) y `TestDashboardToday_T22_AgingReceivables` (`internal/handler/metrics_equivalence_test.go:241`, fila `:260`) fijan hoy el valor con defecto (30.000) con el comentario "preserva D-4, se corrige en US posterior".
- El dashboard permite abonar desde la cartera envejecida y muestra la deuda de esa fila (`frontend/app/page.tsx:285`); el backend valida el abono contra el saldo correcto (`internal/service/customer.go:92-97`).

**Regla de negocio:**
```
saldo en cartera envejecida = saldo(cliente) de spec/domain/fiado.md
  = SUM(sale_payments.amount credit, venta 'completed') − SUM(customer_payments.amount)
Solo clientes con saldo > 0. days_overdue = días desde la venta a crédito
'completed' más vieja (sin cambio). Orden: days_overdue DESC. Límite: 5.
```

**Scope — qué NO entra:**
- `TotalPortfolioBalance` y el total de `/customers` (otra definición de "cartera"; ver roadmap al cierre).
- El desfase UTC/local y las filas con `days_overdue` NULL (R-05).
- Cambios de frontend o del contrato JSON de `/api/dashboard/today`.

**Acceptance Criteria:**
- [ ] El saldo de cada fila de `aging_receivables` es igual al saldo del mismo cliente en `/reports/portfolio` y `/customers`
- [ ] Dora (fiados $25.000 + $25.000, abono $10.000) aparece con $40.000 y 7 días
- [ ] Un cliente con saldo exacto $0 no aparece; uno con abonos mayores que lo fiado tampoco
- [ ] `days_overdue` y el orden de la lista no cambian

**DoD:**
- [ ] Sin migración, sin cambio de contrato HTTP
- [ ] T-17 y T-22 pasan de 30.000 a 40.000 y pierden el comentario "preserva D-4"
- [ ] Build limpio (`go build ./...`, `go test ./...`, `pnpm build`)

### Criterios verificados

> Cerrada el 2026-09-23 (QA vuelta 1: OK, 0 bloqueantes, 2 menores → roadmap R-45). Los checkboxes de arriba son el enunciado original. Decisiones del PO en GATE 1: el total neto de `TotalPortfolioBalance` es un bug fuera de alcance (roadmap R-44); se acepta el cambio visible (saldos mayores en aging y clientes antes ocultos). `AgingReceivables` y `Debtors` comparten ahora `customerBalanceExpr` / `customerBalanceFrom` (`internal/repository/metrics.go:265,270`). Sin cambio de regla en `spec/domain/fiado.md` (solo la fuente nueva).
> Tests: `internal/repository/metrics_test.go` (repo) e `internal/handler/metrics_equivalence_test.go` (handler). Humo manual de QA: 2 fiados de $25.000 + abono de $10.000 → aging, portfolio, `/customers` y `/customers/1/balance` = 40.000.

| AC | Criterio (EARS) | Tests |
|---|---|---|
| AC-1 | CUANDO se invoca `AgingReceivables(limit)`, el sistema DEBE calcular el saldo como `SUM(sale_payments.amount credit de ventas 'completed') − SUM(customer_payments.amount)`, contando cada abono una sola vez sin importar cuántos fiados tenga el cliente. | `TestMetricsRepo_T17_AgingReceivablesPaymentsCountedOnce` (Dora 40.000 / 7 días), `_T31_AgingSeveralCreditsAndPayments` |
| AC-2 | CUANDO un admin llama `GET /api/dashboard/today`, el saldo de cada fila de `aging_receivables` DEBE ser igual al del mismo cliente en `/reports/portfolio` y `/customers`. | `TestMetricsRepo_T35_AgingBalanceMatchesDebtorsListAndBalance`, `TestDashboardToday_T36_AgingBalanceMatchesPortfolioAndCustomers`, `TestDashboardToday_T22_AgingReceivables` |
| AC-3 | MIENTRAS un cliente tenga saldo > 0 y `days_overdue` no NULL, el sistema DEBE incluirlo aunque tenga varios fiados y varios abonos. | `TestMetricsRepo_T31_AgingSeveralCreditsAndPayments` (Eli 3 × $10.000 − $15.000 = 15.000 / 9 días, antes oculta) |
| AC-4 | SI el saldo de un cliente es exactamente 0, ENTONCES el sistema NO DEBE incluirlo. | `TestMetricsRepo_T32_AgingExcludesExactZeroBalance`, `_T16_AgingReceivables` |
| AC-5 | SI los abonos superan lo fiado, ENTONCES el sistema NO DEBE incluir al cliente. | `TestMetricsRepo_T33_AgingExcludesNegativeBalance` |
| AC-6 | `days_overdue` DEBE seguir siendo `hoy − MIN(created_at)` de las ventas 'completed' con línea credit, sin distinguir abonos; orden `days_overdue DESC`; máximo `limit` filas. | `TestMetricsRepo_T17_AgingReceivablesPaymentsCountedOnce`, `_T31_AgingSeveralCreditsAndPayments`, `_T16_AgingReceivables` |
| AC-7 | CUANDO un cliente tenga ventas mixtas o anuladas, el saldo DEBE contar solo las líneas `credit` de ventas `completed`. | `TestMetricsRepo_T34_AgingMixedAndVoidedSales` (Iván: credit $15.000 de venta mixta + $20.000 − $5.000 = 30.000 / 8 días; la anulada no cuenta) |
| AC-8 | (no-regresión) MIENTRAS una fila tenga `days_overdue` NULL, el SQL DEBE excluirla; SI la query falla, el error DEBE volver envuelto con `%w`; sin deudores, slice vacío no-nil. | `TestMetricsRepo_T30_AgingExcludesNullDaysOverdue`, `_T19_ClosedDBWrapsErrors`, `_T18_EmptyRange` |
| AC-9 | El cambio NO DEBE agregar migraciones, NI cambiar el JSON de `aging_receivables` (`customer_id`, `name`, `balance`, `days_overdue`), NI tocar `frontend/`. | `TestDashboardToday_T36_AgingBalanceMatchesPortfolioAndCustomers` (claves JSON); revisión de QA: `git diff dev...HEAD --stat` sin `frontend/` ni `migrate.go` |
| AC-10 | `go build`, `go test`, `pnpm build` y `pnpm test` DEBEN pasar. | Ejecución de QA (T-37): todos exit 0. `pnpm lint` sigue con los 41 problemas previos (R-35) |

---

## US-064 — Corregir `TotalPortfolioBalance`: no restar los saldos a favor

**Origen:** roadmap R-44 (prioridad alta), abierto en el GATE 1 de US-063.

**Como** Don Carlos (dueño),
**quiero** que el total de cartera de `/customers` sea lo que de verdad me deben,
**para** no ver un total más bajo (o absurdo) solo porque un cliente abonó de más.

**Esfuerzo estimado:** 0.3 días. **Impacto:** Alto. Toca el diferenciador del producto: el total que ve el dueño en `/customers` puede ser menor que la suma real de lo que le deben, o incluso caer si un cliente paga de más.

**Estado verificado:**
- `CustomerRepository.TotalPortfolioBalance` (`internal/repository/customer.go:145-160`) resta `SUM(customer_payments.amount)` de **todos** los clientes de `SUM(sale_payments.amount credit, completed)` de **todos** los clientes, en una sola resta global: no agrega por cliente ni filtra `balance > 0`. `List` y `Balance`, en el mismo archivo, sí lo hacen por cliente (`:24-39`, `:127-142`).
- `MetricsRepository` ya expone la fórmula de saldo por cliente como constantes reutilizables en el mismo paquete `repository`: `customerBalanceExpr` (`internal/repository/metrics.go:265`) y `customerBalanceFrom` (`:270-281`), usadas por `Debtors` y `AgingReceivables`.
- Se expone como `total_portfolio` en `GET /api/customers` (`internal/handler/customer.go:31-39`).
- Ejecutado el 2026-09-23 (test temporal, borrado): con Dora ($40.000 fiado / $10.000 abono → $30.000), Eli ($20.000 / $5.000 → $15.000), Fer ($20.000 / $20.000 → $0), Gus ($10.000 / $30.000 → −$20.000): `TotalPortfolioBalance` da **$25.000**; la suma de saldos `> 0` es **$45.000**.
- Contradice `spec/domain/glosario.md:11` ("Cartera... suma de saldos con `balance > 0`") y `spec/domain/fiado.md:37` ("El cliente con saldo `0` o negativo no está en cartera"). El glosario no cambia (decisión del PO en el GATE 1 de US-063).

**Regla de negocio:**
```
total_portfolio = SUM(saldo(cliente)) para cada cliente con saldo(cliente) > 0
saldo(cliente)  = spec/domain/fiado.md (misma fórmula que List, Balance, Debtors,
                  AgingReceivables): créditos de ventas 'completed' menos abonos,
                  agregados por cliente antes de sumar/restar.
Clientes con saldo 0 o negativo (saldo a favor) no aportan ni restan al total.
```

**Scope — qué NO entra:**
- R-45 (higiene de fórmula: `List` duplica el FROM de saldo en vez de reusar `customerBalanceFrom`; desempate de `AgingReceivables`). Ya está en el roadmap.
- R-05 (desfase UTC/hora local) y R-06 (`to` inclusivo en `SalesSummary`). No relacionados.
- Agregar rango de fechas a `TotalPortfolioBalance` (sigue siendo histórico total).
- Cambiar el filtro `has_balance=true` de `listCustomers` o el contrato JSON de `/api/customers`.

**Acceptance Criteria:**
- [x] `TotalPortfolioBalance` devuelve la suma de los saldos de clientes con `balance > 0`, con la misma fórmula que `List`/`Balance`
- [x] Un cliente con saldo exactamente 0 no aporta al total
- [x] Un cliente con saldo negativo (abonó de más) no resta del total de los demás
- [x] Sin clientes, o con todos en saldo `≤ 0`, el total es 0 sin error
- [x] `GET /api/customers`: `total_portfolio` es igual a la suma de `customers[].balance` positivos del mismo payload

**DoD:**
- [x] Sin migración, sin cambio de contrato HTTP
- [x] `internal/repository/customer_test.go` (nuevo) con la fórmula corregida verificada contra el valor con el bug (25.000 → 45.000 con el fixture Dora/Eli/Fer/Gus)
- [x] `internal/handler/customer_test.go` (nuevo) verificando `total_portfolio` end-to-end
- [x] Build limpio (`go build ./...`, `go test ./...`, `pnpm build`)

**Criterios verificados** (cierre 2026-09-23, QA vuelta 1 OK, 0 bloqueantes/menores):

| AC (EARS) | Test |
|---|---|
| AC-1: CUANDO se invoca `CustomerRepository.TotalPortfolioBalance`, el sistema DEBE devolver la suma de los saldos de los clientes con `balance > 0`, calculados con la misma fórmula que `List`/`Balance` (créditos de ventas `completed` menos abonos, agregados por cliente antes de sumar). | `internal/repository/customer_test.go`: `TestCustomerRepo_T1_TotalPortfolioBalance_ExcludesZeroAndNegative`, `TestCustomerRepo_T3_TotalPortfolioBalance_MatchesSumOfBalance` |
| AC-2: SI el saldo de un cliente es exactamente `0`, ENTONCES ese cliente NO DEBE aportar al total. | `internal/repository/customer_test.go`: `TestCustomerRepo_T1_TotalPortfolioBalance_ExcludesZeroAndNegative` (Fer, saldo 0), `TestCustomerRepo_T2_TotalPortfolioBalance_NoDebtorsIsZero` |
| AC-3: SI un cliente abonó más de lo que fiado (saldo negativo), ENTONCES ese cliente NO DEBE restar del total de los demás. | `internal/repository/customer_test.go`: `TestCustomerRepo_T1_TotalPortfolioBalance_ExcludesZeroAndNegative` (Gus, saldo −20.000), `TestCustomerRepo_T2_TotalPortfolioBalance_NoDebtorsIsZero` |
| AC-4: MIENTRAS no haya clientes, o ninguno tenga saldo `> 0`, el sistema DEBE devolver `0`, nunca `nil` ni error. | `internal/repository/customer_test.go`: `TestCustomerRepo_T2_TotalPortfolioBalance_NoDebtorsIsZero` (subcasos "sin clientes" y "solo clientes con saldo <= 0") |
| AC-5: CUANDO un admin llama `GET /api/customers`, el campo `total_portfolio` DEBE ser igual a la suma de los `balance` positivos de los clientes listados en el mismo payload. | `internal/handler/customer_test.go`: `TestListCustomers_T5_TotalPortfolioMatchesPositiveBalances` |
| AC-6: SI la consulta a la base de datos falla, ENTONCES `TotalPortfolioBalance` DEBE devolver el error envuelto con `%w` (no-regresión) y el handler DEBE seguir respondiendo 500 "error calculando cartera". | `internal/repository/customer_test.go`: `TestCustomerRepo_T4_TotalPortfolioBalance_ClosedDBWrapsError` |
| AC-7: El cambio NO DEBE alterar las claves ni los tipos del JSON de `GET /api/customers` (`customers`, `total_portfolio`), NI agregar migración, NI tocar `frontend/`. | `internal/handler/customer_test.go`: `TestListCustomers_T5_TotalPortfolioMatchesPositiveBalances` (verifica exactamente 2 claves top-level); confirmado por diff de QA sin cambios en `frontend/` ni `migrate.go` |

Fixture usado en todos los tests de repositorio (Dora/Eli/Fer/Gus, ver contrato `changes/US-064/contract.md` §4): `Dora` fiado $40.000 / abono $10.000 → saldo $30.000; `Eli` fiado $20.000 / abono $5.000 → saldo $15.000; `Fer` fiado $20.000 / abono $20.000 → saldo $0; `Gus` fiado $10.000 / abono $30.000 → saldo −$20.000. `TotalPortfolioBalance` corregido = $45.000 (antes del fix daba $25.000).

---

## Validaciones Previas con Usuario

Antes o durante el desarrollo de este sprint, validar lo siguiente **con Don Carlos (dueño)**:

- **Chips de sugerencia:** mostrarle un mockup o el widget funcional en el celular. ¿Entiende que los chips se pueden tocar para no tener que escribir? Si no lo entiende, los chips fallan en su propósito de enseñar y acelerar el uso, y hay que rediseñarlos antes de US-062.
- **Referencias de tiempo:** preguntarle *"cuando querés ver el cuadre, ¿preguntás por 'ayer', o por días específicos como 'el martes', 'el fin de semana' o 'la quincena'?"*. Prioriza las referencias temporales del Roadmap.
- **La quincena, aparte:** si menciona la quincena por su cuenta, no es una fecha más. Es el ciclo de cobro del fiado (15 y 30). Anotar textualmente cómo la nombra y para qué intent la usa — muy probablemente `DEBTORS`, no `SALES`.

**Por qué solo Don Carlos y no Laura:** US-062 oculta el widget a los usuarios con rol `cashier`, y US-061 les responde 403. Laura nunca ve el asistente, así que no tiene con qué validarlo.

> [!IMPORTANT]
> **Decisión abierta del PO — asistente para el cajero.**
> Este sprint asume que el asistente es solo para el dueño, porque los 11 intents devuelven ventas, ganancias, margen y cartera, y esos datos ya están restringidos a `admin` en la UI (`internal/handler/handler.go:157,164`).
> Si se quiere que Laura también pregunte, **no basta con mostrarle el widget**: hay que partir los intents en dos grupos de permiso — `LOW_STOCK` y `PRODUCT_LOOKUP` para cualquier usuario autenticado; `SALES`, `PROFIT`, `MARGIN`, `TOP_MARGIN`, `DEBTORS`, `EXPENSES`, `PAYMENT_MIX`, `CASH` y `CREDIT_FLOW` solo para admin. Eso cambia US-061 (el grupo de rutas pasa a ser autenticado, con el filtro de permiso dentro del servicio) y US-062 (el widget se renderiza para todos, con distinto set de chips según el rol).
> Andrey decide. Mientras la decisión esté abierta, el sprint se implementa como está escrito: solo admin.

---

## Criterio de éxito del sprint

- El dueño abre el asistente desde cualquier pantalla, escribe "cuanto vendi oy" sin tildes y con error de dedo, y recibe la cifra correcta en menos de un segundo.
- Pregunta "qué es lo que más vendo" y el asistente no solo lista el top: le dice que su producto estrella no es el que más plata le deja. Ese es el momento que vende el producto en una demo.
- Se desconecta el cable de red de la máquina y las 11 intenciones siguen respondiendo igual, porque nunca hubo una llamada a internet que perder.
- El cajero no ve el botón, y si llama al endpoint a mano recibe 403 sin una sola cifra del negocio.

---

## Notas técnicas para el dev

- **US-059 es la única con riesgo de regresión.** Es refactor de código que el dashboard ya usa en producción. La red de seguridad son los tests de `internal/handler/dashboard_test.go`: si hay que modificarlos para que pasen, el refactor cambió comportamiento y está mal.
- **`internal/assistant/` no importa repositorios.** Si el motor termina necesitando la base de datos, el diseño se torció: el motor decide *qué* preguntar, el servicio *pregunta*.
- **Fechas en SQLite:** el proyecto compara `created_at` contra `time.Time` pasado como parámetro y eso ya funciona en el dashboard actual (`dashboard.go:63`). Seguir ese patrón; no usar `strftime` para filtrar rangos.
- **Semiabierto siempre.** `>= from AND < to`. Un `BETWEEN` es inclusivo en ambos extremos y contaría dos veces las ventas de medianoche entre dos rangos contiguos — exactamente el bug que aparece al comparar "hoy vs ayer".
- **US-060 y US-061 comparten el tipo `Query`.** Se define en `internal/assistant/` y el servicio lo consume; no duplicar la definición de intents en el paquete de servicio.
- **El orden de evaluación de los patrones T1 importa.** Iterar un `map` en Go es aleatorio por diseño: el registro de intents tiene que ser un slice ordenado, no un mapa, o los tests pasarán de forma intermitente.
- **`assistant_queries` guarda la consulta cruda pero nunca cifras.** Verificar en la revisión que ningún monto ni nombre de cliente llegue a esa tabla.
- **Verificar el número de migración antes de escribirla** (ver el aviso al inicio). Depende de si SPRINT-13 se ejecutó antes.

---

## Roadmap

Lo que queda por construir vive en [roadmap.md](roadmap.md), que es la única fuente. Este documento no reserva números de sprint futuros.

> Antes había aquí una tabla de roadmap propia. Se consolidó el 2026-09-21: las copias por sprint divergieron entre sí y llegaron a reclamar el mismo número de sprint para features distintas.
