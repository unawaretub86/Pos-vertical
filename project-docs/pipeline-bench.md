# Benchmark del pipeline agéntico

> **Tipo:** vigente · **Manda sobre:** cómo rinde el pipeline `/us` corrida a corrida · **Actualizado:** 2026-09-23

Cada corrida de `/us` agrega una fila a la tabla de corridas y una sección de detalle. El orquestador lo hace en el paso 6 de `.claude/commands/us.md`, en el mismo commit del merge. Sirve para decidir con datos, no con impresiones, si el pipeline vale lo que cuesta y qué ajustar. El diseño está en [tosdd-plan.md](tosdd-plan.md).

## Estado actual — dónde retomar

> Se actualiza al final de cada sesión de trabajo con el pipeline.

**Última corrida:** 8 (US-054, nivel L, 2026-09-30). Antes: 7 (US-062, nivel L, 2026-09-29). Corrida en curso: ninguna, no hay worktrees `us/*` abiertos. **SPRINT-14 completo en código y tests** (US-059 a US-065); queda pendiente la validación manual del teclado en celular real (M-7 / AC-20) de US-062.

**Aplicado antes de esta corrida:** el ajuste de la corrida 6 (checklist + bloques de tiempo declarados en `implementer.md`, commit `37e5c10`). En la corrida 7 el implementer terminó en 3,9 min sin colgarse, pero el prompt del orquestador también repitió el aviso de trozar archivos y de reportar cada ~20-25 min, así que no se puede separar qué parte del resultado es del ajuste. Una sola muestra: no concluir aún.

**Abierto, sin ajuste aplicado:** (1) el pipeline no tiene forma de verificar UI en navegador: el agente `qa` no tiene herramienta de navegador y el orquestador tampoco intentó una hasta que el PO lo preguntó; la verificación con Edge headless (Playwright) encontró R-55 (ver corrida 7). (2) El clasificador de auto mode falló en Bash y PowerShell durante el cierre y frenó el pipeline en el checkpoint de `close`.

**Próxima US:** siguiente de SPRINT-13 (US-056 en adelante). Pendiente del PO: prueba manual con dos dispositivos de US-054 (AC-16 / T-M).

**(obsoleto, corrida 7)** ninguna en SPRINT-14. Lo que sigue lo define el roadmap (`project-docs/roadmap.md`), que ahora incluye R-49 a R-55 originados en US-062.

**Cómo retomar:** en una sesión nueva, elegir la próxima US y correr `/us US-XXX [M|L]`.

## Cómo se mide

| Métrica | Fuente | Nota |
|---|---|---|
| Tokens por agente | `subagent_tokens` de la notificación de cada agente | Se registra el **último valor** de cada agente. Al retomarlo con `SendMessage`, el valor crece con el contexto acumulado, así que no se suman las retomas |
| Tool uses / duración | `tool_uses` y `duration_ms` de cada notificación | Estos **sí se suman** entre retomas. La duración es tiempo de agente; no incluye la espera del PO en los gates |
| Paradas no planeadas | El orquestador | Veces que se detuvo fuera de los GATE 1 y 2 por algo que el contrato no resolvía |
| Vueltas de QA | `qa.md` | 1 = aprobó a la primera |
| Defectos encontrados | Contrato (DISCREPANCIAS) + hallazgos de test-author y QA | Solo defectos reales del código o de los docs, no preguntas de alcance |
| Ítems al roadmap | Modo `close` | IDs `R-NN` creados |
| Violaciones | Chequeos `git` del orquestador | Un agente tocó archivos fuera de lo suyo |

## Corridas

| # | Fecha | US | Nivel | Tokens (último valor por agente) | Tools | Min agente | Paradas no planeadas | Vueltas QA | Defectos | Roadmap | Violaciones | Diff |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2026-09-23 | US-059 | L | ~440k (A 169k · T 102k · I 71k · Q 101k) | 125 | ~27.5 | 2 | 1 | 5 | 5 | 0 | +1.547 / −224, 13 archivos |
| 2 | 2026-09-23 | US-063 | L | ~310k (A 138k · T 71k · I 35k · Q 67k) | 111 | ~18.5 | 0 | 1 | 4 | 2 (+1 cerrado) | 0 | +466 / −43, 7 archivos |
| 3 | 2026-09-23 | US-064 | M | ~308k (A 173k = 117k+57k · I 71k · Q 64k) | 145 | ~18.0 | 0 | 1 | 1 | 0 (+1 cerrado) | 0 | +353 / −12, 7 archivos |
| 4 | 2026-09-23 | US-065 | M | ~282k (A 176k = 117k+59k · I 59k · Q 47k) | 113 | ~13.3 | 0 | 1 | 0 | 1 cerrado (R-45) | 1 (retomar) | +217 / −20, 7 archivos |
| 5 | 2026-09-23 | US-060 | L | ~522k (A 154k · T 109k · I 175k · Q 85k) | 160 | ~83.8 | 1 (implementer cayó por límite de salida) | 1 | 5 | 1 (R-46) | 0 | +1.333 / −2, 9 archivos |
| 6 | 2026-09-24 | US-061 | M | ~708k (A 372k=149k+109k+115k · I 186k, 2 intentos previos sin telemetría · Q 150k) | 281 (incompleto, 2 intentos de implementer sin `tool_uses` reportado) | ~46.7 agente (+~310 min de pared perdidos en 2 cuelgues, ver abajo) | 2 (implementer colgado dos veces) | 1 | 6 | 2 nuevos (R-47, R-48) + 1 cerrado parcial (R-06) | 0 | +1.867 / −20, 16 archivos |
| 7 | 2026-09-29 | US-062 | L | ~538k (A 340k=135k+102k+75k+28k · T 65k · I 64k · Q 69k) | 168 | ~27.4 | 0 (más 1 parada por el clasificador de auto mode, ajena al contrato) | 1 | 7 discrepancias de contrato (D-1..D-7) + 1 hallazgo preexistente fuera de alcance | 7 nuevos (R-49 a R-55) | 0 | +1.198 / −8, 9 archivos |
| 8 | 2026-09-30 | US-054 | L | ~191k (A 79k=51k+55k+79k · T 32k · I 45k · Q 36k) | 51 | ~9.9 | 0 (un PASA⚠️ mostrado al PO por regla, no por parada) | 1 | 6 discrepancias de contrato (sprint desfasado, rutas dinámicas, `go build` sin `web/out`…) | 1 nuevo (R-62) + 1 cerrado (R-55) | 0 | +452 / −55, 22 archivos |
| 9 | 2026-09-30 | US-056 | L | ~514k (A 213k=82k+49k+83k · T 95k=82k+13k · I 114k · Q 92k=63k+29k) | 205 | ~22 | 1 (checkout principal con cambio en `.claude/`; más un PASA⚠️ mostrado al PO por regla) | 1 (+ smoke 1b) | 8 discrepancias de contrato + 1 defecto de tests (seed choca con migración) | 5 nuevos (R-63..R-67) + 1 cerrado (R-59) | 0 | +2.155 / −26, 26 archivos |

### Tendencia

| | Corrida 1 | Corrida 2 | Lectura |
|---|---|---|---|
| Tokens | ~440k | ~310k (−30%) | Baja por el tamaño de la US y porque se retomaron agentes en vez de relanzarlos |
| Min agente | ~27.5 | ~18.5 | — |
| Paradas no planeadas | 2 | **0** | El ajuste "el analista verifica ejecutando" funcionó |
| Vueltas QA | 1 | 1 | — |
| Violaciones | 0 | 0 | La separación entre tests y código se sostiene |

---

## Corrida 1 — US-059 (piloto)

**US:** extraer las métricas del dashboard a un repositorio parametrizable por rango (SPRINT-14). Refactor puro. **Commit:** `cd2e590`.

### Por agente

| Agente | Invocaciones | Tokens (último) | Tools | Duración |
|---|---|---|---|---|
| analyst | 4 (contrato, ajuste por GATE 1, ajuste T-29 / NULL, cierre) | 169k | 24 + 7 + 5 + 10 = 46 | 287 + 109 + 59 + 119 s = 574 s |
| test-author | 2 (29 tests, luego T-30) | 102k | 16 + 3 = 19 | 328 + 31 s = 359 s |
| implementer | 1 | 71k | 22 | 264 s |
| qa | 1 | 101k | 38 | 454 s |
| **Total** | 8 | ~440k | 125 | 1.651 s ≈ 27.5 min |

### Qué encontró

| Tipo | Hallazgo | Quién |
|---|---|---|
| Defecto 🔴 | D-4: la cartera envejecida multiplica los abonos por la cantidad de fiados del cliente ($30.000 en vez de $40.000) → R-04 | analyst |
| Defecto 🔴 (por confirmar) | D-5: fechas UTC comparadas como texto en hora local → R-05 | analyst |
| Defecto | D-6: `to` inclusivo en `buildDateWhere` y en `ExpenseRepository.List` → R-06 | analyst |
| Defecto | Error de `Scan` ignorado en silencio en aging, top por margen y stock bajo | test-author |
| Defecto de doc | El sprint decía 8 queries; eran 10. La interfaz del sprint no cubría top seller ni aging | analyst |
| Test inútil evitado | T-15, tal como estaba en el contrato, pasaba contra un stub vacío | test-author |
| Error del contrato | T-29 se suponía rojo, pero el código ya respondía 500 | test-author |
| Doc falso | CLAUDE.md sugería que el CI corre `pnpm lint`; no lo corre, y hay 41 problemas previos | qa |

### Paradas no planeadas (2)

1. **T-29 ya pasaba:** el contrato suponía un comportamiento que el código no tenía. Se reclasificó como test de no-regresión.
2. **`days_overdue` NULL:** el contrato no decía cómo tratar esas filas, y propagar el error rompía la equivalencia. El PO eligió la opción A: excluirlas de forma explícita en el SQL.

Las dos vinieron de que el Analista **dedujo** comportamiento en vez de **ejecutarlo**.

### Fricción operativa

- `git worktree remove` falló en Windows por archivos ignorados (`node_modules`, `.next`, 565 MB). Hubo que borrar la carpeta a mano.
- gopls marca errores falsos dentro del worktree, porque no está en el workspace de Go. Es ruido.
- El contrato cambió después del rojo (T-30) y `/us` no tenía un paso para eso.

### Ajustes aplicados después de esta corrida

| Ajuste | Dónde |
|---|---|
| Borrado forzado del worktree + `rm -rf` de lo que quede | `.claude/commands/us.md` paso 5 |
| Retomar agentes con `SendMessage` si la sesión sigue viva (el analyst pasó de 136k a 169k tokens en 3 retomas, contra ~100k por arranque en frío) | `us.md` convenciones, `tosdd-plan.md` §2 |
| Paso 2b: nueva vuelta del test-author si el contrato cambia después del rojo | `us.md` |
| El analyst verifica **ejecutando** (curl o test rápido) las afirmaciones sobre respuestas HTTP y errores, no solo leyendo | `.claude/agents/analyst.md` |
| Nota sobre los errores falsos de gopls | `us.md` |
| Registrar el benchmark en cada corrida | `us.md` paso 6 |
| CLAUDE.md: el CI no corre lint | `CLAUDE.md:52` |

### Qué medir en la corrida 2

- ¿Bajan las paradas no planeadas con la verificación ejecutada del Analista?
- Costo de una US **M** frente a una **L**.
- R-04 es un fix pequeño: sirve para ver si el pipeline sobra en cambios chicos.

---

## Corrida 2 — US-063 (fix del bug D-4, desde el roadmap R-04)

**US:** corregir el saldo de la cartera envejecida del dashboard (SPRINT-14). Bug fix que cambia un número visible para el dueño. **Commits de la rama:** `180e201` → `fbccc6b` (RED) → `d999fad` → `0c5b276`, fusionados en un solo commit en `dev`.

### Por agente

| Agente | Invocaciones | Tokens (último) | Tools | Duración |
|---|---|---|---|---|
| analyst | 3 (contrato + registro de la US, decisiones de GATE 1, cierre); retomado con `SendMessage` | 138k | 33 + 8 + 18 = 59 | 333 + 33 + 122 s = 488 s |
| test-author | 1 | 71k | 15 | 222 s |
| implementer | 1 | 35k | 7 | 113 s |
| qa | 1 | 67k | 30 | 290 s |
| **Total** | 6 | ~310k | 111 | 1.112 s ≈ 18.5 min |

### Qué encontró

| Tipo | Hallazgo | Quién |
|---|---|---|
| Defecto 🔴 (alcance real mayor) | DISC-1: el bug no solo daba montos equivocados, también **ocultaba deudores**. Eli debía $15.000 y no aparecía porque el cálculo le daba −$15.000 | analyst (ejecutado) |
| Defecto 🔴 | DISC-3: `TotalPortfolioBalance` resta los saldos a favor ($40.000 vs $55.000) → R-44 | analyst |
| Defecto (efecto en UI) | DISC-4: el modal de abono mostraba la deuda equivocada y la fila desaparecía tras abonar. Lo resuelve el fix del backend | analyst (leído) |
| Defecto menor | `ORDER BY days_overdue DESC` sin desempate con `LIMIT` → R-45 | qa |
| Deuda | `CustomerRepository.List` duplica la fórmula de saldo → R-45 | implementer + qa |

Con la búsqueda por grep se confirmó que no hay otros JOIN que multipliquen montos.

### Paradas no planeadas: 0

Solo se pasó por los dos gates. El Analista ejecutó el comportamiento actual con un test temporal antes de escribir el contrato, y ningún test cayó en `PASA⚠️`.

### Fricción operativa

- La US venía del roadmap y no existía en el sprint. `/us` supone que existe (paso 0.2), así que el Analista la registró en modo `contract`. Eso dejó el sprint modificado en el worktree, y el orquestador lo commiteó aparte (`docs(US-063): register US`) para que el commit `RED` tuviera solo tests.
- El GATE 1 dejó decisiones que solo tocaban §7 y §8 del contrato. El Analista las registró **en paralelo** con el test-author, sin conflicto.
- El roadmap no tenía formato para ítems cerrados; el Analista creó la sección "Cerrados".

### Ajustes propuestos (pendientes del "sí" del PO)

| Ajuste | Dónde |
|---|---|
| Paso 0.2: si la US viene del roadmap (R-NN), el Analista la registra en el sprint durante el modo `contract` y el orquestador la commitea aparte antes de `RED` | `.claude/commands/us.md` |
| Si las decisiones del GATE 1 solo tocan §7 y §8, el test-author puede correr en paralelo | `.claude/commands/us.md` |
| Formato "Cerrados" del roadmap documentado en su encabezado | `project-docs/roadmap.md` |

### Qué medir en la corrida 3

- Una US **M** (sin test-author), para comparar el costo contra L.
- Si R-44 (otro fix chico, de saldo) confirma que ~300k tokens es el piso de una US L, o si baja más.

---

## Corrida 3 — US-064 (R-44, nivel M)

**US:** corregir `TotalPortfolioBalance` para que no reste los saldos a favor (bug declarado en el GATE 1 de US-063). Bug fix chico, sin test-author separado (nivel M). **Rama:** `us/US-064` → `78fbdac` (registro) → `71a30a2` (implementación) → `a78a959` (cierre), fusionados en `dev`.

### Por agente

| Agente | Invocaciones | Tokens (último) | Tools | Duración |
|---|---|---|---|---|
| analyst | 2 (contrato + registro de la US; cierre) — **lanzados en frío, no retomados con SendMessage** | 117k + 57k = 173k | 39 + 22 = 61 | 375 + 161 s = 536 s |
| test-author | 0 (nivel M lo salta) | — | — | — |
| implementer | 1 | 71k | 35 | 234 s |
| qa | 1 | 64k | 49 | 310 s |
| **Total** | 4 | ~308k | 145 | 1.080 s ≈ 18.0 min |

### Qué encontró

| Tipo | Hallazgo | Quién |
|---|---|---|
| Defecto de doc | El texto de R-44 en el roadmap citaba cifras del seed de `TestMetricsRepo_T35` ($40.000/$55.000) que no coinciden con lo que ese seed produce al ejecutarlo ($60.000/$75.000) — el texto olvidó a un cliente del seed (Beto, $20.000) | analyst (ejecutado) |

Sin discrepancias bloqueantes de alcance. El bug en sí ya estaba caracterizado desde el GATE 1 de US-063; esta corrida solo lo confirmó ejecutando con un fixture propio (25.000 con el bug, 45.000 correcto) y lo corrigió.

### Paradas no planeadas: 0

Solo los dos gates. La nota de §8 (corregir la cifra de R-44) era informativa, no bloqueante, y Andrey la resolvió en el mismo GATE 1.

### Fricción operativa

- **Violación de la convención "retomar en vez de relanzar":** el orquestador lanzó un `analyst` nuevo para el modo `close` (`a7354c5de9d3b5136`) en vez de retomar con `SendMessage` al `analyst` del contrato (`ac2bf2bf927d46ac4`), que seguía vivo en la sesión. Costó un arranque en frío extra (~57k tokens) en vez de una retoma más barata. Es un error del orquestador, no de un agente — corregir en la próxima corrida.
- Antes del paso 0 se encontraron gitlinks sucios preexistentes en `.claude/worktrees/*` (residuos de worktrees de Claude Code, no del POS) y un `rewrite_links.py` staged sin relación. La regla del paso 0.1 exige parar; se consultó a Andrey, que autorizó seguir. No se tocó ninguno de los dos.
- Nivel **M** funcionó sin fricción: el implementer escribió tests e implementación en un solo commit, siguiendo la tabla de tests del contrato igual de detallada que en nivel L; QA no encontró tests debilitados por faltar el test-author.

### Ajustes propuestos (pendientes del "sí" del PO)

| Ajuste | Dónde |
|---|---|
| Antes de lanzar cualquier agente, el orquestador revisa explícitamente si ya hay un `agentId` vivo en la sesión para ese rol y esa US antes de decidir `Agent` vs `SendMessage` (no solo "recordarlo", sino un chequeo explícito en el paso correspondiente) | `.claude/commands/us.md` pasos 1 y 5 |

### Qué medir en la corrida 4

- Si se corrige la fricción de retomar, ¿baja el costo de analyst en un cierre similar (2 invocaciones → 1 retomada)?
- Nivel M sigue dando ~300k tokens totales pese a saltar test-author: el ahorro de saltar un agente (~65-100k) se compensa con más costo de analyst por el error de retoma. Repetir nivel M sin ese error para aislar el verdadero piso de costo de una US chica.

---

## Corrida 4 — US-065 (R-45, nivel M)

**US:** higiene de la fórmula de saldo detectada en el QA de US-063 — `CustomerRepository.List` deja de duplicar textualmente el `FROM` de saldo (reusa `customerBalanceExpr`/`customerBalanceFrom`) y `AgingReceivables` gana desempate estable (`balance DESC, name ASC`) cuando empata `days_overdue`. Bug fix chico + refactor de higiene, nivel M. **Rama:** `us/R-45` → `ee4f334` (implementación) → `0c3cf4a` (cierre), fusionados en `dev`. El analyst registró la US en el sprint como **US-065** durante el modo `contract` (R-45 venía del roadmap, sin entrada propia en el sprint).

### Por agente

| Agente | Invocaciones | Tokens (último) | Tools | Duración |
|---|---|---|---|---|
| analyst | 2 (contrato + registro de la US; cierre) — **de nuevo lanzados en frío, no retomados con SendMessage** | 117k + 59k = 176k | 41 + 19 = 60 | 360 + 123 s = 483 s |
| test-author | 0 (nivel M lo salta) | — | — | — |
| implementer | 1 | 59k | 27 | 165 s |
| qa | 1 | 47k | 26 | 148 s |
| **Total** | 4 | ~282k | 113 | 796 s ≈ 13.3 min |

### Qué encontró

| Tipo | Hallazgo | Quién |
|---|---|---|
| Defecto menor | Redacción redundante en `spec/domain/fiado.md:38` — la frase nueva del delta repetía "ordena por `days_overdue` descendente" dos veces en la misma oración | qa |

Sin discrepancias de alcance: el texto de `roadmap.md:48` describía con precisión ambos puntos (confirmado con grep y, en el caso del desempate, ejecutando un fixture temporal que mostró el bug real — con `LIMIT` menor que los empatados, el cliente con mayor deuda podía quedar fuera del corte por orden de inserción).

### Paradas no planeadas: 0

Solo los dos gates. La única fricción no fue una parada del contrato sino un error del orquestador (ver abajo).

### Fricción operativa

- **Se repitió la violación de "retomar en vez de relanzar" de la corrida 3, pese a estar documentada como ajuste pendiente:** el orquestador lanzó un `analyst` nuevo (`ab24d286fefcf0abb`) para el modo `close` en vez de retomar con `SendMessage` al `analyst` del contrato (`ae7f21565663133bd`), que seguía vivo en la sesión. El ajuste propuesto en la corrida 3 ("chequeo explícito de `agentId` vivo antes de `Agent` vs `SendMessage`") nunca se aplicó a `us.md` — quedó como propuesta pendiente del "sí" del PO y nadie volvió a plantearlo antes de la corrida 4. Mismo costo extra de arranque en frío (~59k tokens) que en la corrida 3.
- Mismo estado sucio preexistente que en la corrida 3 (`rewrite_links.py` staged en el checkout principal, gitlinks modificados en `.claude/worktrees/*`), ya autorizado por Andrey entonces; se dejó intacto de nuevo, sin volver a preguntar porque no cambió desde la corrida anterior.
- Nivel **M** volvió a funcionar sin fricción de contrato/tests: el implementer escribió los 3 tests nuevos (T-39, T-40, T-41) más los cambios de código en un solo commit, sin que QA encontrara tests débiles por faltar el test-author.

### Ajustes propuestos (pendientes del "sí" del PO)

| Ajuste | Dónde |
|---|---|
| El de la corrida 3 sigue sin aplicarse y sigue siendo necesario: antes de lanzar cualquier agente, el orquestador debe listar explícitamente (no solo "recordar") los `agentId` vivos de la sesión para esa US antes de decidir `Agent` vs `SendMessage`. Dos corridas seguidas con el mismo error sugiere que hace falta un paso mecánico en `us.md`, no una nota de convención | `.claude/commands/us.md` pasos 1 y 5 |

### Qué medir en la corrida 5

- Si se aplica el ajuste de arriba antes de correr: ¿el costo total baja de ~280k a algo más cercano a ~220k (analyst en 1 sola invocación retomada)?
- El piso de una US M sin errores de retoma sigue sin medirse limpio — van 2 corridas seguidas con el mismo defecto de proceso inflando el costo de analyst en +~50-60k tokens.

---

## Corrida 5 — US-060 (Motor de intenciones T1/T2, nivel L)

**US:** paquete `internal/assistant/` desde cero — normalización, 11 intenciones por regex (T1) y fuzzy matching con Levenshtein (T2), extracción de periodo relativo (hoy/ayer/semana/mes/año + días de la semana con error de dedo). Go puro, sin DB ni HTTP; primera pieza de una cadena de 3 (US-060 → US-061 → US-062). Nivel **L**, primera corrida con el ajuste de retoma mecánico ya aplicado. **Rama:** `us/US-060` → `4f0ba6b` (RED, 31 tests) → `b36f108` (implementación) → `4a20d52` (cierre), fusionados en `dev` por squash.

### Por agente

| Agente | Invocaciones | Tokens (último) | Tools | Duración |
|---|---|---|---|---|
| analyst | 4 (contrato inicial; ajuste por 2 decisiones GATE 1; agregar T-30/T-31 al contrato; cierre) — **las 4 retomadas con `SendMessage`, cero arranques en frío tras la primera** | 73k → 110k → 115k → 154k (último: 154k) | 23+34+5+20 = 82 | 303+130+29+155 = 617 s |
| test-author | 2 (29 tests iniciales; agregar T-30/T-31) — retomado con `SendMessage` | 98k → 109k (último: 109k) | 18+6 = 24 | 634+77 = 711 s |
| implementer | 2 (arranque; **retomado tras un crash**) | falló sin uso reportado → 175k (último: 175k) | 30 (solo la invocación que terminó) | 3.417 s (incluye el turno cortado, retomado en la misma conversación) |
| qa | 1 | 85k | 24 | 285 s |
| **Total** | 9 | ~522k (154+109+175+85) | 160 | 5.029 s ≈ 83.8 min |

### Qué encontró

| Tipo | Hallazgo | Quién |
|---|---|---|
| Defecto del sprint 🔴 (verificado ejecutando Levenshtein fuera del repo) | El umbral `levenshtein<=1` que el propio sprint usa como ejemplo no alcanza sus propios casos: `"aller"→"ayer"` y `"kien"→"quien"` dan distancia real 2, no 1 | analyst |
| Ambigüedad de contrato | "Palabra candidata a tiempo que no resuelve" (→ `Unresolved`) no tenía criterio de candidatura definido; "jueves" no estaba en ningún vocabulario del sistema | analyst |
| Gap de cobertura | AC-13 ("mes pasado") y AC-14 ("este año") estaban en §3 del contrato pero sin fila propia en la tabla de tests de §4 — nadie los iba a probar | test-author |
| Menor | `ambiguousSeed` compartida a propósito entre bancos T1 de SALES y PROFIT tensiona la letra de AC-25 (no-solapamiento); no bloquea, es tensión entre dos ACs del propio contrato | qa |
| Menor | Vocabulario de días de la semana no tiene guard de longitud mínima que sí aplica al fuzzy de intents — sin falso positivo hoy, riesgo si el vocabulario crece → **R-46** | qa |

### Paradas no planeadas (1)

**El implementer se cortó a mitad de turno:** `API Error: Claude's response exceeded the 64000 output token maximum`, probablemente generando un archivo completo (con banco de frases y mapa de sinónimos) en un solo bloque de texto. Cero código de producción había quedado escrito (`git status` limpio, tests intactos) — no fue necesario revertir nada. Se retomó con `SendMessage` pidiéndole explícitamente dividir el trabajo en archivos/ediciones más chicas, y terminó sin volver a cortarse. Duración total del agente (3.417 s) incluye el turno perdido, así que no es comparable 1:1 con implementers de corridas anteriores sin este incidente.

### Fricción operativa

- El ajuste de retoma mecánico (aplicado antes de esta corrida, ver `us.md`) funcionó: los 3 agentes con más de una invocación (analyst ×4, test-author ×2, implementer ×2) se retomaron siempre con `SendMessage`, nunca con `Agent` en frío innecesario — primera corrida sin esa violación desde la 2.
- Al aplicar el fix a `us.md`, el orquestador cometió sin querer `rewrite_links.py` (archivo ajeno pre-staged desde antes de la sesión) en el mismo commit por usar `git commit -m` sin excluirlo del índice. Se corrigió con un commit separado (`git rm --cached`) antes de tocar el worktree de la US. Lección: revisar `git status` después de `git add <archivo>` puntual, no asumir que el índice solo tiene lo que se acaba de agregar.
- Nivel **L** con test-author funcionó según lo esperado: encontró un gap real de cobertura (AC-13/AC-14 sin test) que en nivel M probablemente habría pasado desapercibido hasta QA o después.
- El crash del implementer por límite de salida es nuevo en el historial de corridas — no había pasado en las 4 anteriores, posiblemente por el tamaño del banco de frases/sinónimos de esta US frente a fixes chicos anteriores.

### Ajustes propuestos (pendientes del "sí" del PO)

| Ajuste | Dónde |
|---|---|
| Cuando el implementer vaya a escribir un archivo grande con tablas/bancos de datos extensos (regex, sinónimos, frases semilla), indicarle de entrada que divida la escritura en varios `Write`/`Edit` en vez de un archivo completo en un solo bloque, para evitar el límite de 64k tokens de salida por turno | `.claude/agents/implementer.md` o `us.md` paso 3 |
| Antes de cualquier `git commit` en el checkout principal fuera del worktree de la US, revisar `git status` explícitamente para no arrastrar archivos ajenos pre-staged | `us.md`, nota general para el orquestador |

### Qué medir en la corrida 6

- ¿El aviso al implementer sobre dividir archivos grandes evita el crash en la próxima US con volumen de datos similar (US-061 o US-062)?
- US-061 depende de US-059 (métricas) y US-060 (motor) ya cerradas — buen caso para medir si una US "de integración" (servicio + endpoint + bitácora, con DB y HTTP) cuesta más o menos que el motor puro de esta corrida.

## Corrida 6 — US-061 (servicio, endpoint `/ai/ask` y bitácora, nivel M)

**US:** pega US-059 (métricas) con US-060 (motor de intenciones) — servicio nuevo (`internal/service/assistant.go`), repo de bitácora (`internal/repository/assistant_queries.go`), handler+endpoint (`internal/handler/assistant.go`), migración 15. Nivel **M**, sin test-author. **Rama:** `us/US-061` → `1454fd6` (implementación+tests, sin RED separado) → `f793074` (cierre), fusionados en `dev` por squash.

### Por agente

| Agente | Invocaciones | Tokens (último) | Tools | Duración |
|---|---|---|---|---|
| analyst | 3 (contrato inicial; corrección con las 6 decisiones de GATE 1 — el orquestador investigó D-2 por su cuenta antes de relanzar; cierre) — **las 3 en frío, `ListAgents` nunca mostró una viva** | 149k → 109k → 115k (último: 115k) | 41+51+33 = 125 | 374+522+275 = 1.171 s |
| implementer | 3 (arranque; retomado en frío tras crash por límite de salida a los 600s; retomado en frío otra vez tras un segundo cuelgue de 5h matado a mano) | falló sin uso reportado → falló sin uso reportado → 186k (último: 186k, único con telemetría) | 79 (solo la invocación que terminó) | 846 s reportados (agente) + ~310 min de pared sin telemetría en los 2 intentos fallidos |
| qa | 1 | 150k | 77 | 785 s |
| **Total** | 7 | ~708k (372+186+150, subestimado — 2 intentos de implementer sin `subagent_tokens`) | 281 (subestimado, mismo motivo) | 2.802 s de telemetría ≈ 46.7 min + ~310 min de pared no medidos |

### Qué encontró

| Tipo | Hallazgo | Quién |
|---|---|---|
| Defecto real 🔴 (bloqueante, roadmap R-06) | `buildDateWhere` (report.go:192) inclusivo en `to` — el PO exigió corregirlo global en vez de aceptar el riesgo; el fix también destapó el mismo bug latente en `dashboard.go:142-159` | analyst, decisión GATE 1 |
| Gap real | `ReportRepository.TopProducts` ordena por ingresos, pero el spec de TOP_PRODUCTS exige el contraste "#1 por unidades" — investigado por el orquestador (no por el analyst) tras la instrucción del PO "averigualo, buscá"; resuelto con un método nuevo (`TopSellers`) para no romper el endpoint de reportes existente que sí depende del orden por ingresos | orquestador + analyst |
| Riesgo real, atrapado antes de producción | El motor nunca devuelve `Tier="none"`; la migración exige `CHECK tier IN (...,'none')`. Sin mapear `""→"none"` antes del INSERT, el 100% de las consultas `UNKNOWN` —justo las que la bitácora existe para capturar— habrían roto el INSERT | analyst, en el contrato inicial (AC-15) |
| Defecto preexistente en `dev`, no de esta US | `SalesSummary` agrupado colapsa en un solo bucket cuando `created_at` es un `time.Time` literal insertado a mano (patrón de fixture del proyecto); no se dispara hoy porque `sale.go` siempre usa `DEFAULT CURRENT_TIMESTAMP`. QA lo reprodujo contra `dev` sin el diff de esta US → **R-48** | qa |
| Menor, aceptado | 4 frases de entrenamiento nuevas en `engine.go` no listadas en el contrato original — QA verificó que revertirlas rompe 4/7 casos de `Period.Label` y que no rompen nada de US-060 | qa |
| Deferido a roadmap | R-06 solo cubre `report.go`; `ExpenseRepository.List` (expense.go:73) sigue con `to` inclusivo, sin US que lo dispare todavía → cierre parcial de R-06 | analyst, modo close |
| Deferido a roadmap | `frontend/app/reports/page.tsx:98` arma `to` como 23:59:59 en vez de inicio del día siguiente; tras el fix de `buildDateWhere` queda una ventana de 1 segundo sin cubrir → **R-47** | analyst, contrato |

### Paradas no planeadas (2)

**Intento 1 del implementer:** cortado por el mismo límite de salida que ya había pasado en la corrida 5 (archivo grande en un solo bloque), esta vez detectado por el watchdog automático a los 600s sin progreso. El orquestador verificó `git status`/`go build` antes de relanzar: nada roto, se retomó en frío con instrucción explícita de trabajar en bloques chicos.

**Intento 2 del implementer, el hallazgo nuevo de esta corrida:** el mismo aviso de "dividí el trabajo en bloques chicos" hizo que el agente avanzara MUY lento en vez de cortarse — corrió **5 horas reales** (el PO preguntó "¿cómo carajo lleva 5 horas?") sin que el watchdog de 600s lo detectara como colgado, porque técnicamente seguía haciendo tool calls, solo que a un ritmo altísimo de lentitud (diff final de esa vuelta: +15 líneas sobre el intento anterior). El orquestador comparó el progreso real (`git diff --stat` entre el estado post-crash-1 y el estado a las 5h) para confirmar que no era percepción, mató el proceso con `TaskStop`, verificó que el archivo a medio editar (`engine.go`) no había quedado roto (`go build` pasó igual), y relanzó una tercera vez con alcance deliberadamente acotado (solo los 2 archivos de test que faltaban, con la instrucción de correr `go test` de esos paquetes después de cada bloque en vez de al final). Esa tercera vuelta sí terminó, en 846s.

### Fricción operativa

- El ajuste de la corrida 5 ("avisar que divida archivos grandes") **no es suficiente por sí solo** contra los cuelgues del implementer — puede evitar el crash de salida (lo hizo) y al mismo tiempo producir un colgado silencioso mucho peor (lentitud extrema no detectada por el watchdog de tiempo-sin-progreso).
- Los 2 intentos fallidos del implementer no generaron notificación con `subagent_tokens`/`tool_uses` — el benchmark de esta corrida subestima el costo real en tokens y herramientas de esos 2 intentos, solo tiene el de la vuelta que terminó.
- El orquestador no tiene un mecanismo propio de vigilancia activa fuera de las notificaciones — probó `ScheduleWakeup` para chequear en 20 min y lo canceló de inmediato al notar que esa herramienta es específica de `/loop` y no aplica a esta sesión. El chequeo del intento 2 (a las 5h) fue reactivo, disparado por la queja del PO, no por vigilancia propia del orquestador.
- Investigar D-2 (orquestador en vez de analyst) funcionó bien: la instrucción del PO fue "buscá vos", y verificar código real (`grep` de `TopProducts`/`TopByMargin`, quién más llama a `TopProducts`) antes de relanzar al analyst evitó una vuelta extra de preguntas.

### Ajustes propuestos (pendientes del "sí" del PO)

| Ajuste | Dónde |
|---|---|
| Darle al implementer un límite de tiempo de pared explícito (ej. 20-25 min) para completar un bloque de trabajo declarado, no solo watchdog de progreso; si lo excede, el orquestador chequea `git diff --stat` contra el último checkpoint conocido y decide matar/continuar con datos, no a ciegas | `.claude/agents/implementer.md`, `us.md` paso 3 |
| Cuando el implementer arranca una vuelta de "completá lo que falta" (no un contrato completo desde cero), pedirle una lista explícita de los tests/archivos pendientes ANTES de escribir código, para que el orquestador pueda medir avance real contra un checklist en vez de solo `git diff --stat` | `.claude/agents/implementer.md` |
| No usar `ScheduleWakeup` fuera de una sesión `/loop` — confirmado en esta corrida que es específico de esa skill y hay que cancelarlo si se llama por error | nota operativa para el orquestador, no requiere cambio de archivo |

### Qué medir en la corrida 7

- ¿El límite de tiempo de pared (si el PO lo aprueba) reduce los cuelgues silenciosos del implementer, o el problema es específico de "vueltas de completar trabajo ajeno" como la de esta corrida?
- US-062 (widget de chat, frontend) es la primera US de este bloque que toca React/Next.js — buen caso para medir si el patrón de cuelgues es un problema del agente `implementer` en general o algo específico de Go/backend.

## Corrida 7 — US-062 (widget de chat del asistente, nivel L)

**US:** widget flotante de chat en la app autenticada que consume `POST /ai/ask` (US-061): `frontend/lib/assistant.ts` (lógica pura + store de módulo), `AssistantWidget.tsx` (solo render), montaje en `AppLayout.tsx`, más un cambio de backend por decisión del PO en GATE 1 (chips de sugerencia como frases completas, `internal/service/assistant.go`). Primera US frontend del pipeline. **Rama:** `us/US-062` → `4ef5a34` (RED) → `378c6d0` (implementación) → `1b32e38` (cierre) → `740d067` (R-55), fusionados en `dev` por squash.

### Por agente

| Agente | Invocaciones | Tokens (último) | Tools | Duración |
|---|---|---|---|---|
| analyst | 4 (contrato; ajuste con las decisiones del GATE 1; cierre; registro de R-55) — **las 4 en frío, `ListAgents` nunca mostró una viva** | 135k → 102k → 75k → 28k | 48+26+26+12 = 112 | 652+193+175+50 = 1.070 s |
| test-author | 1 | 65k | 16 | 135 s |
| implementer | 1 | 64k | 17 | 236 s |
| qa | 1 | 69k | 23 | 206 s |
| **Total** | 7 | ~538k (340+65+64+69) | 168 | 1.646 s ≈ 27,4 min |

### Qué encontró

| Tipo | Hallazgo | Quién |
|---|---|---|
| Defecto real 🔴 | Los chips que emitía el backend no funcionaban al reenviarlos (`hoy`, `ayer`, `cuánto vendiste`… devolvían `unknown`). El analyst lo demostró ejecutando el motor; el PO eligió la opción A (cambiar las sugerencias en `assistant.go`, código de US-061 ya cerrada) → T-12 y T-13 de ida y vuelta | analyst (D-2) |
| Afirmación falsa del sprint | "`AppLayout` sigue montado al navegar": cada una de las 13 páginas monta su propio `AppLayout`; un `useState` perdería el historial → store de módulo (D-1) | analyst |
| Otras discrepancias de contrato | D-3..D-7: `formatCOP` casi no aplica, las listas hay que parsearlas del `text`, `isMobile` no es reutilizable, `interactiveWidget` ausente (teclado no testeable), `#000` sobre `--accent` | analyst |
| Hallazgo preexistente, fuera de alcance 🔴 | `web/static.go:23` abre la ruta con barra inicial, `io/fs` la rechaza, y el binario sirve `index.html` para toda URL (incluidos los `.js`): la UI embebida queda en blanco. Además `frontend/lib/api.ts:3` fija `http://localhost:8080/api` y el CORS solo acepta `:3000`. **No lo vio el QA automatizado; lo encontró la verificación manual con navegador** → **R-55** | orquestador, con Edge headless (Playwright) |
| Menores de QA, aceptados | `boxShadow` con `rgba` fijo; `catch {}` sin log (→ R-54); `useIsMobile` parte en `false` | qa |
| Deferido a roadmap | R-49 (401 → `/login`), R-50 (layout compartido), R-51 (tests de componentes), R-52 (periodos incompletos por intent), R-53 (chips ambiguos pierden el periodo), R-54 (`catch {}` sin log), R-55 (frontend embebido) | analyst |

### Paradas no planeadas: 0 del contrato

Ninguna parada por algo que el contrato no resolviera, ningún cuelgue del implementer. Sí hubo una interrupción ajena al pipeline: el clasificador de auto mode devolvió errores en Bash y PowerShell durante el cierre (~10 intentos) y el pipeline quedó frenado en el checkpoint de `close` hasta que volvió solo. Con herramientas de solo lectura (Grep/Read) se pudo verificar el contenido de los docs, pero no el `git status`.

### Verificación manual con navegador

El agente `qa` no tiene herramienta de navegador y marcó M-1..M-9 como pendientes (correcto). El orquestador no lo intentó hasta que el PO preguntó por qué no se había hecho. Se hizo entonces con `playwright-core` instalado fuera del repo (en el directorio temporal del job) y el Edge instalado: 23 comprobaciones, 21 OK y 2 falsos fallos por errores del propio script (regex sensible a mayúsculas, semilla de datos incompleta). Cubrió M-1..M-6, M-8, M-9, P-1 opción A, P-3, saludo, carga y foco, historial por navegación SPA y F5, sin storage, tema claro/oscuro, 320 px y rol cajero. Quedó sin verificar el teclado en celular real (AC-20) y el solape con el Toast.

Costo real de armar el entorno: la primera pasada fue vacía sin saberlo (las páginas salían en blanco por R-55 y "0 botones" pasaba trivialmente); hubo que agregar una aserción de "la página renderizó texto" para que la comprobación M-3 valiera algo. Compilar con `-overlay` para cambiar el puerto (8080 estaba ocupado por otro proceso del PO), reconstruir el frontend con `NEXT_PUBLIC_API_URL=/api` y un servidor Node con proxy propio, por R-55 y por el CORS a `:3000`.

### Fricción operativa

- Un solo commit de `implementer.md` (`37e5c10`) más un prompt con aviso de trozar archivos y reportar por bloques coincidieron con la primera corrida sin cuelgues del implementer en tres. No se puede atribuir la mejora a uno de los dos.
- El analyst se lanzó en frío las 4 veces: `ListAgents` no mostró ninguna instancia viva cada vez que se consultó (los agentes terminados desaparecen de la lista poco después de entregar). El chequeo mecánico se cumplió, pero en la práctica ahora nunca hay retoma posible salvo que el orquestador le escriba al agente antes de que termine.
- Git Bash reescribe valores como `NEXT_PUBLIC_API_URL=/api` a `C:/Program Files/Git/api` y mangla los acentos de `curl -d` inline; hay que usar `MSYS_NO_PATHCONV=1` y `--data-binary @archivo`.
- En el checkout principal apareció trabajo ajeno sin commitear durante la sesión (arreglo en curso de `web/static.go`, un borrador `SPRINT-15-Asistente-Continuacion.md`, `web/out/.gitkeep` borrado por una compilación). El merge por squash no los tocó porque no se solapaban con el diff y el commit se hizo sin `-a`.

### Ajustes propuestos (pendientes del "sí" del PO)

| Ajuste | Dónde |
|---|---|
| Para US con UI, agregar al paso 4 una verificación con navegador (Playwright + Edge, o la herramienta de navegador integrada si está disponible) hecha por el orquestador o un agente con esa herramienta, y exigir que las comprobaciones "no aparece X" incluyan una aserción de que la página realmente renderizó | `.claude/commands/us.md`, contrato §4 (M-n) |
| Documentar en `project-docs/` (o como skill de proyecto vía `/run-skill-generator`) cómo levantar el binario con un puerto/DB de prueba y el frontend con API relativa, para no reconstruirlo cada vez; deja de hacer falta cuando R-55 se cierre | `project-docs/` o `.claude/skills/` |
| Aclarar en `us.md` que, tras entregar, un agente desaparece de `ListAgents`, y que la retoma solo es posible si el orquestador manda `SendMessage` antes; o cambiar el chequeo a "usa el agentId de la invocación anterior" | `.claude/commands/us.md` |

### Qué medir en la corrida 8

- ¿El implementer vuelve a terminar sin cuelgues en una US de tamaño similar (frontend + backend)? Con una muestra más se puede intentar separar el efecto del ajuste de `implementer.md` del efecto del aviso en el prompt.
- Si el PO aprueba la verificación con navegador, cuánto tiempo y tokens añade por US con UI.
- Una US que arranque con `ListAgents` mostrando algo vivo, para probar que la ruta de retoma sigue funcionando.

## Corrida 8 — US-054 (acceso LAN + servir el frontend desde el binario, nivel L)

| Agente | Invocaciones | Tokens (último valor) | Tools | Duración |
|---|---|---|---|---|
| analyst | 3 (contract, ajuste GATE 1, close; las dos últimas por `SendMessage`) | 79k | 23 | ~4,2 min |
| test-author | 1 | 32k | 6 | ~1,1 min |
| implementer | 1 | 45k | 12 | ~2,8 min |
| qa | 1 | 36k | 10 | ~1,8 min |

**Qué encontró.** El contrato reprodujo con un `fstest.MapFS` el comportamiento real de `FileServer` y detectó lo que el sprint no decía: rutas dinámicas (`customers/[id]`, `catalogo/[token]`) exportadas como un solo `_.html`, `go build` roto en checkout limpio por `//go:embed all:out` sin `.gitkeep` trackeado, y `lan_access_enabled` ya existente. Todas las decisiones (P-1..P-8) se resolvieron en un solo GATE 1. QA: 0 BLOQUEANTE, 3 MENOR; ninguno cambió código.

**Verificación del orquestador.** Antes del merge se armó el binario con `web/out` poblado y se comprobó por `curl`: `/login`, `/customers/5` y `/catalogo/abc` → HTML; chunk JS → `text/javascript`; `/_next/nope.js` → 404; `/api/setup/status` → 200. Esto cierra lo que QA dejó sin probar (su smoke corrió con `web/out` vacío). Sigue sin verificar en navegador real: T-12 y AC-16 / T-M.

**Paradas no planeadas:** 0. Un caso `PASA⚠️` en T-11 (guarda de regresión) se mostró al PO por regla.

**Fricción operativa.**
- `ListAgents` sí mostró al analyst vivo tras entregar (a diferencia de la corrida 7), y la retoma por `SendMessage` funcionó dos veces. El agente se ve en la lista mientras dura la sesión; desaparecía en la corrida 7 por otra causa no identificada.
- El implementer escribió `servidor.md` (spec) por su cuenta; el analyst lo verificó en `close`. Es borde del reparto de roles, sin daño.
- Corrida notablemente más barata que las anteriores (~191k vs 282k–708k): contrato acotado, sin vueltas de QA y sin cuelgues. US más pequeña que US-062/US-061; no concluir todavía.

### Qué medir en la corrida 9
- Si la verificación con navegador (ajuste pendiente de corridas 7) se aprueba, cuánto añade por US con UI.
- Si el implementer vuelve a terminar sin cuelgues.

## Corrida 9 — US-056 (pedido por mesa, nivel L)

| Agente | Invocaciones | Tokens (último valor) | Tools | Duración |
|---|---|---|---|---|
| analyst | 3 (contract, ajuste GATE 1, close; las tres en frío) | 83k (82k + 49k + 83k entre las tres) | 112 | ~7,0 min |
| test-author | 2 (rojo; corrección de seed) | 13k (82k + 13k) | 16 | ~4,0 min |
| implementer | 1 | 114k | 44 | ~6,6 min |
| qa | 2 (revisión; smoke 1b) | 29k (63k + 29k) | 33 | ~4,4 min |

**Qué encontró.** El contrato descubrió ejecutando: el sprint daba por existente el ocultamiento de carritos con `restaurant_mode` (no existe), PATCH no pasaba CORS (`middleware.go:97`), `/mesas/[orderId]` necesitaba `generateStaticParams` y `dynamicRoutes`, y las FK no se aplican, por lo que hizo falta un INSERT atómico (R-59 cerrado sin migración). Siete preguntas del PO se resolvieron en un solo GATE 1 más una repregunta (Q-6, sobre qué era polling) y Q-8. QA: 0 BLOQUEANTE, 4 MENOR en la revisión y 1 MENOR en el smoke; ninguno cambió código.

**Defecto de tests.** El seed `INSERT INTO users (id=1…)` del test-author chocaba con la migración (`migrate.go:252`, `INSERT OR IGNORE`), así que T-3, T-14 y T-15 no podían pasar. Lo detectó el implementer, no el test-author. Costó una invocación extra del test-author y un nuevo commit `RED` (`1bcdd65`).

**Paradas no planeadas:** 1. El checkout principal tenía un gitlink de `.claude/worktrees/` modificado; el PO mandó seguir. Además, un `PASA⚠️` (T-14) se mostró al PO por regla.

**Fricción operativa.**
- `ListAgents` no mostró ningún agente terminado del mismo rol en ningún paso; todas las invocaciones fueron en frío.
- El smoke quedó omitido en la primera pasada porque el puerto 8080 es fijo (`cmd/pos/main.go:74-81`) y estaba ocupado por el PO; se repitió como vuelta 1b cuando lo liberó.
- `go run scripts/copy-frontend.go` borra `web/out/.gitkeep` en el worktree; QA lo restauró con `git checkout`.
- `TestDashboardToday_T20_Golden` falla por fecha también en `dev` (R-67).

### Qué medir en la corrida 10
- Si el test-author corre sus tests contra las migraciones reales, cuántas vueltas de tests se evitan.
- Costo del smoke separado cuando el puerto está ocupado.
