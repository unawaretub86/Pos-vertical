# Roadmap — Pos-vertical

> **Tipo:** vigente · **Manda sobre:** qué se construye después · **Actualizado:** 2026-09-23

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
| R-05 | Confirmar con un test el desfase UTC vs hora local en los rangos de fecha (D-5), **antes de US-061** | US-059 | Inferido del código, no reproducido: las filas se guardan con `CURRENT_TIMESTAMP` (UTC, texto) y los rangos se pasan como `time.Time` que el driver serializa con `t.String()` (sin `_time_format` en `internal/database/database.go:14`), así que la comparación es de texto. En Colombia (UTC−5), las ventas después de las 19:00 caerían en la ventana del día siguiente. Primero un test que lo reproduzca, después la corrección. Incluye las **filas de aging excluidas por `days_overdue` NULL** (`metrics.go:335`, desde US-063): al corregir el formato de fecha deben volver a aparecer. Base de "hoy/ayer" del asistente. |
| R-06 | Volver semiabierto el `to` de `buildDateWhere` / `SalesSummary` (D-6) | US-059 | `internal/repository/report.go:192` usa `<= ?`, igual que `ExpenseRepository.List` (`expense.go:73`). Viola la convención `[from, to)` y duplica las ventas de medianoche entre rangos contiguos. Corregir antes de que US-061 use `SalesSummary` para `MARGIN` y `PAYMENT_MIX`. |

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
| R-45 | Higiene de la fórmula de saldo detectada en el QA de US-063 | US-063 | (1) `CustomerRepository.List` (`internal/repository/customer.go:28-39`) conserva una copia textual del FROM de saldo en vez de usar `customerBalanceFrom` (`internal/repository/metrics.go:270`); hoy son idénticas y T-35/T-36 fijan el invariante, pero la "una sola fórmula" de `spec/domain/fiado.md` sigue en dos archivos. (2) `AgingReceivables` ordena solo por `days_overdue DESC` (`metrics.go:336`) sin desempate: con empates en días y `LIMIT 5`, qué cliente entra no está definido (ya pasaba antes de US-063). Agregar desempate estable (p. ej. saldo DESC, nombre ASC) con test de empate. |

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

---

## Deuda abierta

La deuda y las limitaciones verificadas del producto están en [gaps.md](gaps.md).
Este documento cubre lo que se va a construir, no lo que falta por cerrar.

---

## Bitácora de decisiones

Cambios de rumbo que no se deducen de ningún otro documento.

| Fecha | Decisión | Por qué |
|---|---|---|
| 2026-09-23 | R-44 cerrado por US-064: `TotalPortfolioBalance` suma saldos por cliente `> 0` en vez de restar un neto global; se corrige el ejemplo numérico del ítem al archivarlo | El contrato de US-064 (§2, §8) ejecutó el seed de `TestMetricsRepo_T35_...` citado en el texto original de R-44 y obtuvo $60.000 (bug) / $75.000 (correcto), no los $40.000/$55.000 que decía el roadmap — el texto original omitía a Beto ($20.000) del seed. Andrey aprobó la corrección en GATE 1 de US-064 para que el historial de "Cerrados" no quede con una cifra falsa. |
| 2026-09-23 | R-04 cerrado por US-063: la cartera envejecida usa la misma fórmula de saldo que la cartera y `/customers` | El PO aceptó el cambio visible (saldos mayores y clientes antes ocultos). El efecto en el modal de abono del dashboard (DISC-4: mostraba la deuda equivocada y, tras abonarla, la fila desaparecía aunque quedara saldo, `frontend/app/page.tsx:285,393-404`) queda resuelto por el backend, sin cambio de frontend. |
| 2026-09-23 | El total neto de `TotalPortfolioBalance` es un bug, no una "cartera neta" intencional (R-44) | Decisión del PO en el GATE 1 de US-063. El glosario ("suma de saldos con `balance > 0`") es la definición vigente; el código se corrige, no el spec. |
| 2026-09-23 | US-059 preserva el saldo con defecto de la cartera envejecida (D-4) | El refactor debía dar exactamente los mismos números que el dashboard. La corrección va aparte (R-04), con un test que ya marca qué aserción cambiar. |
| 2026-09-21 | El asistente local (E2) sube de "Sin fecha" a SPRINT-14 | Gancho de venta inmediato y, con motor de reglas en vez de LLM, cero riesgo en hardware de 4GB. |
| 2026-09-21 | El asistente **no** lleva LLM en V1, ni siquiera vía Ollama | Requisito del PO: nada que instalar ni descargar en el PC del cliente. Un modelo local queda en R-33 condicionado a datos de uso. |
| 2026-09-21 | SPRINT-14 tomó el número que SPRINT-13 había reservado para la continuación del restaurante | El desplazamiento no se registró en su momento; R-01 recoge lo diferido. Este roadmap existe para que no vuelva a pasar. |
| 2026-09-08 | Capital embebido se descarta como línea propia | Requiere partner fintech regulado. Solo sobrevive el scoring interno (R-17), que sí es ejecutable en solitario. |
