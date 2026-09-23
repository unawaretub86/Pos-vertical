# Benchmark del pipeline agéntico

> **Tipo:** vigente · **Manda sobre:** cómo rinde el pipeline `/us` corrida a corrida · **Actualizado:** 2026-09-23

Cada corrida de `/us` agrega una fila a la tabla de corridas y una sección de detalle. El orquestador lo hace en el paso 6 de `.claude/commands/us.md`, en el mismo commit del merge. Sirve para decidir con datos, no con impresiones, si el pipeline vale lo que cuesta y qué ajustar. El diseño está en [tosdd-plan.md](tosdd-plan.md).

## Estado actual — dónde retomar

> Se actualiza al final de cada sesión de trabajo con el pipeline.

**Última corrida:** 4 (US-065, nivel M, 2026-09-23). Corrida en curso: ninguna, no hay worktrees `us/*` abiertos.

**Pendiente de decisión del PO al retomar:** el ajuste propuesto en la corrida 3 (chequeo explícito de `agentId` vivo antes de `Agent` vs `SendMessage`) sigue sin aplicarse a `us.md` — la corrida 4 repitió el mismo error que iba a prevenir. Próxima US: a elegir del backlog de SPRINT-14 o de items nuevos del roadmap (no quedó ninguno abierto tras cerrar R-45).

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
