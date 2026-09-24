# Roadmap — Pos-vertical

> **Tipo:** vigente · **Manda sobre:** qué se construye después · **Actualizado:** 2026-09-24

Única fuente de verdad sobre lo que viene. Antes existía una tabla de Roadmap al final de cada documento de sprint; se copiaban entre sí, divergieron, y terminaron reclamando el mismo número de sprint para cosas distintas. **Los documentos de sprint ya no llevan roadmap propio ni reservan números futuros.**

Cada ítem tiene un ID estable (`R-NN`) que no cambia aunque se mueva de bucket.

---

## Cómo se lee

| Bucket | Significa |
|---|---|
| **Siguiente** | Candidato al próximo sprint. Andrey elige entre estos. |
| **Listo para planear** | Sin bloqueo técnico. Entra cuando haya presupuesto de sprint. |
| **Bloqueado** | Depende de algo que todavía no existe. La dependencia está nombrada. |
| **Sin fecha** | Válido pero sin señal de que bloquee la venta al primer cliente. |
| **Requiere tercero** | No ejecutable en solitario. |
| **Cerrados** | Resuelto. Queda como historial con la US que lo cerró. |

---

## Siguiente

| ID | Ítem | Origen | Nota |
|---|---|---|---|
| R-01 | Continuación del módulo restaurante: para llevar con turno diario, modificadores con precio | SPRINT-13 | El turno es media US sobre tablas ya creadas. Los modificadores tocan catálogo y precios. **Solo tiene sentido después de cerrar SPRINT-13**, que está pendiente. |
| R-02 | Continuación del asistente: intent `PRODUCT_LOOKUP` ("¿cuánto queda de arroz?") | SPRINT-14 | Requiere extracción de entidad y desambiguación contra el catálogo. US completa, no un intent más. Depende de cerrar SPRINT-14. |
| R-03 | Referencia temporal "la quincena" en el asistente | SPRINT-14 | **Condicionado a la validación con Don Carlos.** Si la nombra por su cuenta, no es una fecha más: es el ciclo de cobro del fiado (15 y 30) y probablemente se pide junto a `DEBTORS`. Sería la referencia temporal de mayor valor. |
| R-05 | Confirmar con un test el desfase UTC vs hora local en los rangos de fecha (D-5) | US-059 | Inferido del código, no reproducido: las filas se guardan con `CURRENT_TIMESTAMP` (UTC, texto) y los rangos se pasan como `time.Time` que el driver serializa con `t.String()` (sin `_time_format` en `internal/database/database.go:14`), así que la comparación es de texto. En Colombia (UTC−5), las ventas después de las 19:00 caerían en la ventana del día siguiente. Primero un test que lo reproduzca, después la corrección. Incluye las **filas de aging excluidas por `days_overdue` NULL** (`metrics.go:335`, desde US-063): al corregir el formato de fecha deben volver a aparecer. Base de "hoy/ayer" del asistente. **Sigue abierto tras US-061 (2026-09-24):** la nota decía "antes de US-061"; el contrato de US-061 confirmó que `internal/database/database.go:14` no cambió (sin DSN de zona horaria) y que ninguna de las decisiones de GATE 1 lo tocó — deuda heredada de US-059, sin plan de cierre todavía. |
| R-48 | `SalesSummary` agrupada colapsa en un solo bucket cuando `created_at` se inserta como `time.Time` literal (mismo origen que R-05) | QA de US-061 | Hallazgo de QA (vuelta 1, MENOR, `qa.md`): al corregir `buildDateWhere` (R-06), el implementer envolvió el `strftime(...)` de `SalesSummary` agrupada en `COALESCE(..., '')` (`internal/repository/report.go:113-127`) porque sin eso 3 tests de US-061 fallaban con `sql: Scan error ... converting NULL to string`. QA reprodujo el mismo crash contra `report.go` de `dev` (bug preexistente, no introducido por US-061) y confirmó que no se dispara en producción (`internal/repository/sale.go:46` nunca inserta `created_at` explícito, siempre usa el `DEFAULT CURRENT_TIMESTAMP` de SQLite). Con el `COALESCE`, ventas de días distintos con `created_at` como `time.Time` literal (el patrón de fixture de `CLAUDE.md`/`dashboard_test.go:14`) colapsan silenciosamente en un solo grupo `period=""` en vez de fallar fuerte — QA lo verificó con una prueba ad-hoc (3 ventas en 3 días → 1 solo grupo `total=300 count=3`). No afecta a ningún AC de US-061 ni al frontend hoy (no hay test de regresión sobre `Grouped`). Candidato a resolverse junto con R-05: mismo origen (formato de `strftime` sobre el texto que produce el driver de SQLite). |

## Listo para planear

| ID | Ítem | Origen | Nota |
|---|---|---|---|
| R-10 | G8 — reportes por hora pico y por vendedor | `research/2026-09-08-investigacion-mercado.md:167` | `report.go` no agrupa por hora ni por `user_id`. ~0.5 día: dos queries `GROUP BY`. Desbloquea R-20 y es base para comisiones. |
| R-11 | F7 + I4 — PIN rápido por cajero y permisos granulares | `research/2026-09-08-investigacion-mercado.md` (bloque F) | Cambio de turno sin logout completo. Relevante cuando el negocio pasa de 1 empleado. |
| R-12 | Propina, dividir cuenta, transferir ítems entre mesas | SPRINT-13 | Complican el cobro, que en SPRINT-13 recién se estabiliza. Solo si el cliente los pide. |
| R-13 | Pantalla "qué me preguntan y no entiendo" | SPRINT-14 | Lee `assistant_queries`. Necesita semanas de uso real acumuladas para mostrar algo. |
| R-14 | Fechas exactas en el asistente ("15 de abril", "marzo", "el martes") | SPRINT-14 | Entra cuando la bitácora demuestre que se piden cuadres históricos. |
| R-15 | H2 — apartados / layaway | `research/2026-09-08-investigacion-mercado.md` (bloque H) | Extiende el diferenciador de fiado. El doc fuente lo marca como pedido del mes 2, no del día 1. |
| R-16 | G2 — kardex y ajustes de inventario | `research/2026-09-08-investigacion-mercado.md` (bloque G) | "Requisito 2026" según el doc fuente. |
| R-17 | Scoring interno del cliente fiado | `research/2026-09-08-investigacion-mercado.md` (bloque D) | Score de riesgo, límite de crédito sugerido y alerta temprana de mora, con el historial de abonos que ya se guarda. **Ningún competidor local lo hace.** Complejidad media, cero dependencia externa. Es el diferenciador más fuerte sin construir. |
| R-18 | E1 — sync multi-caja por LAN offline | `research/2026-09-08-investigacion-mercado.md` (bloque E1) | 2–5 cajas, mismo inventario, sin nube. La parte dura es resolución de conflictos. Relevante cuando el cliente crece a multi-caja. |
| R-19 | Sacar el SQL inline que queda en `getDashboard` (D-3) | US-059 | Ventas recientes (`internal/handler/dashboard.go:183`) y la cartera calculada en Go sobre `customers.List`. US-059 limpió solo `getDashboardToday` por decisión del PO. Candidato: `RecentSales(limit)` en `MetricsRepository` y total de cartera sobre `Debtors`. |
| R-06 | Volver semiabierto el `to` de `ExpenseRepository.List` (`expense.go:73`) | US-059 | **Cerrado parcialmente por US-061 (2026-09-24):** la mitad de este ítem, `buildDateWhere` en `internal/repository/report.go:192` (compartida por `SalesSummary` y `TopProducts`), ya es semiabierta `[from, to)` — `<=` → `<`, con test `TestReportRepo_T22_BuildDateWhereIsHalfOpen`. Queda abierta la otra mitad: `ExpenseRepository.List` (`expense.go:73`) sigue con `<=` inclusivo. US-061 no la tocó porque el asistente usa `ExpenseRepository.SumRange` (ya semiabierta desde antes), no `List`. Sin urgencia atada a una US concreta. |

## Bloqueado

| ID | Ítem | Bloqueado por |
|---|---|---|
| R-20 | Intents `PEAK_HOURS` y `BY_SELLER` en el asistente | R-10 (G8). Cuando exista el agrupamiento por hora y vendedor, cada intent es una fila más en el registro. |
| R-21 | Intents `DEAD_STOCK` y `REORDER` en el asistente | US-048 (capital muerto) y US-044 (sugerencia de pedido), ambas pendientes de SPRINT-09. |
| R-22 | US-044 — sugerencia de pedido al proveedor | Desbloqueada desde el cierre de SPRINT-11 (US-052 dio proveedores). Pendiente de programar; pertenece a SPRINT-09, que sigue abierto. |

## Sin fecha

| ID | Ítem | Por qué |
|---|---|---|
| R-30 | G3–G7 — vencimientos, combos, etiquetas, historial de precio, foto de producto | Features de nicho (farmacia, ferretería). Sin señal de que bloqueen la primera venta. |
| R-31 | H3–H5 — domicilios, impuesto de bolsas, cotizaciones | Igual: nicho o volumen que el cliente actual no tiene. |
| R-32 | Plano visual de mesas arrastrable, rol "mesero", comanda impresa, KDS por SSE | Cosmético o dependiente de que el cliente crezca. El plano vende en demo pero no resuelve lo que la lista no resuelva. |
| R-33 | Tier de modelo de lenguaje local en el asistente | Condicionado a dos cosas: que `assistant_queries` muestre volumen real de preguntas fuera de dominio, **y** que exista una vía sin instalar nada en el PC del cliente. Si entra, traduce texto a `{intent, periodo}` y nunca genera cifras. |
| R-35 | Higiene de calidad detectada en el QA de US-059 | Todo es anterior a US-059 y nada rompe CI: (1) `pnpm lint` falla con 41 problemas (32 errores, 9 warnings) en `frontend/`, y `.github/workflows/test.yml` no corre lint aunque CLAUDE.md dice que CI corre los cuatro comandos; (2) `gofmt` marca `internal/handler/dashboard.go` (tags de `getDashboard`) y `internal/handler/handler.go:130`; (3) los 500 de `/dashboard/today` y `/reports/portfolio` no registran el error envuelto en el log, así que se pierde la causa al diagnosticar. |
| R-34 | App móvil nativa del dueño | `K1` (acceso LAN, US-054) cubre el caso dentro del local. Fuera del local requiere nube, que contradice el posicionamiento. |
| R-46 | Guard de longitud mínima en el fuzzy del vocabulario temporal/días del asistente | El fuzzy de vocabulario temporal y de días de la semana (`internal/assistant/period.go:61-72`) no tiene el guard `len(t)>=4` que sí aplica al fuzzy de intents (decisión GATE 1 de US-060: se omitió a propósito porque `"oy"→"hoy"`, largo 2, debe resolver). No es un bug hoy — los 66 casos del banco de US-060 no producen falso positivo, verificado en QA (T-24). Riesgo a vigilar si el vocabulario temporal crece (p. ej. al construir R-03 "quincena" o R-14 "fechas exactas"): una palabra corta legítima podría matchear un día de la semana por accidente vía Levenshtein≤1. |
| R-47 | Alinear `frontend/app/reports/page.tsx:98` (`endOfDay`) al patrón `[from, start-of-next-day)` | Hallazgo D-5 del contrato de US-061: `/reports` arma su `to` como "fin de día" (`23:59:59`), no "inicio del día siguiente" como usan `dashboard.go` y `period.go`. Tras el fix de `buildDateWhere` (`<=`→`<`, R-06 parcial) queda una ventana de 1 segundo (`23:59:59` exacto) sin cubrir: una venta creada en ese segundo exacto no aparece en `/reports/sales` ni `/reports/top-products`, aunque sí en `/dashboard/today`. Riesgo acotado (1s sobre 86.400, resolución de `created_at` es de segundo), sin test ni reporte de usuario que lo haya detectado. No bloqueante; cambio de una línea en el frontend cuando se priorice. |

## Requiere tercero

| ID | Ítem | Qué falta |
|---|---|---|
| R-40 | H1 — recargas y pago de servicios | Alianza con agregador (Movilred / Puntored / SuRed). |
| R-41 | Capital embebido (préstamo sobre data de ventas) | Partner fintech regulado por la SFC. Los tres líderes mundiales ya lo venden; en Colombia nadie lo hace a nivel tienda de barrio. Visión de largo plazo, no se persigue en solitario. |
| R-42 | K3 — envío automático por WhatsApp API o SMTP | Cuenta business y costo recurrente. Contradice "offline sin mensualidad". Posible add-on pago si un cliente lo pide. |
| R-43 | K4 — acceso remoto en vivo (tunnel o nube) | 5+ días y rompe el posicionamiento offline. Solo cuando un cliente pregunte "¿y desde la casa?". |

---

## Cerrados

Ítems resueltos. Se conservan como historial con la US que los cerró; el ID no se reutiliza.

| ID | Ítem | Origen | Cerrado por |
|---|---|---|---|
| R-04 | ~~Corregir el saldo de la cartera envejecida del dashboard (D-4)~~ | US-059 | **Resuelto por US-063 (2026-09-23).** `AgingReceivables` usa ahora la misma fórmula que `Debtors` (`customerBalanceExpr` / `customerBalanceFrom`, `internal/repository/metrics.go:265,270`). Dora: $30.000 → $40.000; clientes con 3+ fiados y abonos parciales (Eli) vuelven a aparecer. Tests: `TestMetricsRepo_T17_AgingReceivablesPaymentsCountedOnce`, `_T31`…`_T35`, `TestDashboardToday_T22_AgingReceivables`, `_T36`. Texto original: `MetricsRepository.AgingReceivables` (`internal/repository/metrics.go:314-331`) hace `JOIN sale_payments` + `SUM(cp.paid)`, así que multiplica los abonos por la cantidad de fiados del cliente. Ejemplo: fiados de $25.000 + $25.000 con abono de $10.000 muestran $30.000 en vez de $40.000. Contradice `spec/domain/fiado.md` (un saldo, una fórmula). US-059 lo preservó a propósito por decisión del PO. La corrección unifica la fórmula con `Debtors`, cambia la aserción de `TestMetricsRepo_T17_AgingReceivablesPreservesD4` y `TestDashboardToday_T22_AgingReceivables` (Dora 30.000 → 40.000; comentario "preserva D-4") y actualiza el spec de fiado. Toca el diferenciador. |
| R-44 | ~~**Prioridad alta.** Corregir `TotalPortfolioBalance`: resta los saldos a favor (DISC-3)~~ | US-063 | **Resuelto por US-064 (2026-09-23).** `CustomerRepository.TotalPortfolioBalance` (`internal/repository/customer.go:145-163`) ahora agrega el saldo por cliente reutilizando `customerBalanceExpr`/`customerBalanceFrom` (`internal/repository/metrics.go:265,270`, misma fórmula que `List`/`Balance`/`Debtors`/`AgingReceivables`) y suma solo los saldos `> 0`, en vez de restar en una sola operación todos los abonos de todos los fiados. Tests: `TestCustomerRepo_T1_TotalPortfolioBalance_ExcludesZeroAndNegative`, `_T2_NoDebtorsIsZero`, `_T3_MatchesSumOfBalance`, `_T4_ClosedDBWrapsError` (`internal/repository/customer_test.go`), `TestListCustomers_T5_TotalPortfolioMatchesPositiveBalances` (`internal/handler/customer_test.go`). Spec actualizado: `spec/domain/fiado.md:40`. Texto original: `CustomerRepository.TotalPortfolioBalance` (`internal/repository/customer.go:145-160`) resta todos los abonos de todos los fiados, así que un cliente que abonó de más reduce el total. Con el seed de `TestMetricsRepo_T35_AgingBalanceMatchesDebtorsListAndBalance` (Dora 40.000, Eli 15.000, Fer 0, Gus −15.000) devuelve **$40.000**; la suma de saldos > 0 es **$55.000**. Se muestra como `total_portfolio` en `GET /customers` (`internal/handler/customer.go:31-39`). Contradice `spec/domain/glosario.md:11` ("suma de saldos con `balance > 0`") y `spec/domain/fiado.md` (saldo 0 o negativo no está en cartera). El PO lo declaró bug en el GATE 1 de US-063; el glosario no cambia. Toca el diferenciador. **Corrección de cifra al cerrar (2026-09-23, aprobada por Andrey en GATE 1 de US-064):** el ejemplo de arriba decía que ese mismo seed daba $40.000 (bug) / $55.000 (correcto); al ejecutarlo tal cual (el implementer de US-064 corrió el seed real, que incluye también a **Beto**, saldo $20.000, omitido del texto original), el resultado real es **$60.000** con el bug y **$75.000** correcto — no $40.000/$55.000. Se corrige aquí para que el historial no quede con una cifra falsa que nadie reprodujo antes de escribirla; la naturaleza del bug y su causa eran correctas, solo el ejemplo numérico estaba mal. |
| R-45 | ~~Higiene de la fórmula de saldo detectada en el QA de US-063~~ | US-063 | **Resuelto por US-065 (2026-09-23).** (1) `CustomerRepository.List` (`internal/repository/customer.go:24-28`) reemplazó su copia textual del `SELECT`/`FROM` de saldo por las constantes compartidas `customerBalanceExpr`/`customerBalanceFrom` (`internal/repository/metrics.go:265,270-281`), mismo patrón que ya usaban `Debtors`, `AgingReceivables` y `TotalPortfolioBalance`. (2) `AgingReceivables` (`internal/repository/metrics.go:339`) agregó desempate estable: `ORDER BY days_overdue DESC, balance DESC, c.name ASC` (mismo criterio que `Debtors`, con `days_overdue` como criterio primario sin cambio). Antes del fix, con empate de `days_overdue` el orden dependía de `rowid`/inserción: un cliente con $50.000 de deuda podía quedar fuera de un `LIMIT` menor mientras uno con $10.000 sí entraba, solo por orden de inserción — verificado ejecutando en el contrato (§2). Ningún saldo ni `days_overdue` cambió para ningún cliente; solo el orden en empates. Tests: `TestCustomerRepo_T5_ListSearchFiltersByNameOrPhone` (`internal/repository/customer_test.go`), `TestMetricsRepo_T40_AgingTieBreakByBalanceThenName`, `TestMetricsRepo_T41_AgingTieBreakDecidesLimitCutoff` (`internal/repository/metrics_test.go`); no-regresión confirmada en `TestMetricsRepo_T16_AgingReceivables`, `_T31_AgingSeveralCreditsAndPayments`, `_T14_DebtorsTieByName` y `go test ./...` completo. Spec actualizado: `spec/domain/fiado.md:38-39` (nueva viñeta de orden de `AgingReceivables`). Sin migración, sin cambio de contrato JSON, sin `frontend/`. |

---

## Deuda abierta

La deuda y las limitaciones verificadas del producto están en [gaps.md](gaps.md).
Este documento cubre lo que se va a construir, no lo que falta por cerrar.

---

## Bitácora de decisiones

Cambios de rumbo que no se deducen de ningún otro documento.

| Fecha | Decisión | Por qué |
|---|---|---|
| 2026-09-24 | R-06 cerrado parcialmente por US-061: `buildDateWhere` (`report.go:192`) ya es semiabierto; `ExpenseRepository.List` (`expense.go:73`) sigue abierto bajo el mismo ID | GATE 1 de US-061 acotó la corrección de `<=`→`<` a la función `buildDateWhere`, compartida por `SalesSummary`/`TopProducts`, porque el asistente consume `ExpenseRepository.SumRange` (ya semiabierta), no `List`. No se reabre R-06 como nuevo, se deja la mitad pendiente con la misma ID (Regla 3: no se abre ID nuevo para el resto de un ítem ya existente). |
| 2026-09-24 | QA de US-061 encontró 2 MENOR, ninguno bloqueante; el segundo pasa a R-48, el primero queda aceptado sin ítem de roadmap | (1) 4 frases nuevas de entrenamiento T1 en `internal/assistant/engine.go:105-110` (fuera de la tabla "Archivos a tocar" del contrato) — QA verificó que son necesarias para que `Period.Label` resuelva en 7/7 casos y que no rompen ningún test de US-060; aditivo, aceptado sin más acción. (2) El `COALESCE(strftime(...), '')` que el implementer agregó a `SalesSummary` agrupada (`report.go:113-127`) para no romper con fixtures de `created_at` como `time.Time` literal expone un bug preexistente (colapso silencioso en un solo bucket) que no se dispara en producción hoy pero tampoco tiene test de regresión → **R-48**, junto a R-05 por compartir causa raíz (formato de timestamp). |
| 2026-09-24 | R-05 sigue abierto: US-061 no lo tocó, la nota "antes de US-061" queda obsoleta | Confirmado en el contrato de US-061 (§2): `internal/database/database.go:14` no cambió y ninguna de las 6 decisiones de GATE 1 de US-061 tocó el desfase UTC/hora local. Se actualiza la nota de R-05 para no dejar una condición temporal que ya venció. |
| 2026-09-23 | QA de US-060 encontró 2 MENOR; el segundo pasa a R-46, el primero queda documentado sin ítem propio | (1) `ambiguousSeed` ("cuanto gane vendiendo") se agrega a propósito al banco T1 de `SALES` y `PROFIT` a la vez — es el ejemplo literal que el contrato usa para AC-4/T-4 (frase deliberadamente ambigua), y en sentido estricto tensiona la letra de AC-25 ("ninguna frase del banco de un intent resuelve a otro"). No es un bug ni queda sin test: está documentado en el código (`engine.go:87-90`) y no lo cubre `TestEngine_T24` porque esa tabla reutiliza el banco T3, que no incluye `ambiguousSeed`. Es una nota sobre cómo escribir el próximo contrato que reutilice un caso "ambiguo a propósito" como semilla de dos intents, no deuda de producto — no amerita `R-NN`. (2) La ausencia de guard de longitud mínima en el fuzzy del vocabulario de días sí es una superficie de riesgo real si el vocabulario crece → **R-46**. |
| 2026-09-23 | R-45 cerrado por US-065: `List` reutiliza `customerBalanceExpr`/`customerBalanceFrom`, `AgingReceivables` agrega desempate estable (saldo DESC, nombre ASC) en empate de `days_overdue` | Higiene de código sin cambio de cifra visible en el caso general; el desempate sí cambia, en el caso raro de empate exacto de días con `LIMIT` ajustado, cuál cliente aparece en el widget. Ningún bloqueo técnico ni discrepancia de contrato; ejecutado y verificado en el contrato de US-065. |
| 2026-09-23 | R-44 cerrado por US-064: `TotalPortfolioBalance` suma saldos por cliente `> 0` en vez de restar un neto global; se corrige el ejemplo numérico del ítem al archivarlo | El contrato de US-064 (§2, §8) ejecutó el seed de `TestMetricsRepo_T35_...` citado en el texto original de R-44 y obtuvo $60.000 (bug) / $75.000 (correcto), no los $40.000/$55.000 que decía el roadmap — el texto original omitía a Beto ($20.000) del seed. Andrey aprobó la corrección en GATE 1 de US-064 para que el historial de "Cerrados" no quede con una cifra falsa. |
| 2026-09-23 | R-04 cerrado por US-063: la cartera envejecida usa la misma fórmula de saldo que la cartera y `/customers` | El PO aceptó el cambio visible (saldos mayores y clientes antes ocultos). El efecto en el modal de abono del dashboard (DISC-4: mostraba la deuda equivocada y, tras abonarla, la fila desaparecía aunque quedara saldo, `frontend/app/page.tsx:285,393-404`) queda resuelto por el backend, sin cambio de frontend. |
| 2026-09-23 | El total neto de `TotalPortfolioBalance` es un bug, no una "cartera neta" intencional (R-44) | Decisión del PO en el GATE 1 de US-063. El glosario ("suma de saldos con `balance > 0`") es la definición vigente; el código se corrige, no el spec. |
| 2026-09-23 | US-059 preserva el saldo con defecto de la cartera envejecida (D-4) | El refactor debía dar exactamente los mismos números que el dashboard. La corrección va aparte (R-04), con un test que ya marca qué aserción cambiar. |
| 2026-09-21 | El asistente local (E2) sube de "Sin fecha" a SPRINT-14 | Gancho de venta inmediato y, con motor de reglas en vez de LLM, cero riesgo en hardware de 4GB. |
| 2026-09-21 | El asistente **no** lleva LLM en V1, ni siquiera vía Ollama | Requisito del PO: nada que instalar ni descargar en el PC del cliente. Un modelo local queda en R-33 condicionado a datos de uso. |
| 2026-09-21 | SPRINT-14 tomó el número que SPRINT-13 había reservado para la continuación del restaurante | El desplazamiento no se registró en su momento; R-01 recoge lo diferido. Este roadmap existe para que no vuelva a pasar. |
| 2026-09-08 | Capital embebido se descarta como línea propia | Requiere partner fintech regulado. Solo sobrevive el scoring interno (R-17), que sí es ejecutable en solitario. |
