# Plan — Orquestación agéntica con Spec-Driven Development (SDD)

> **Tipo:** propuesta · **Vigente:** pendiente de aprobación del PO · **Manda sobre:** cómo se desarrolla una US con agentes
> **Fecha:** 2026-09-23 · Verificado contra el repo en esa fecha.
> **Revisión 2 (2026-09-23):** de 6 a 4 agentes, `changes/` fuera de git, worktree creado al inicio de la US.

**Pedido del PO:** una orquestación de agentes para desarrollar las tareas. Un agente de negocio (planear, revisar y actualizar la documentación), uno que lea requisitos, uno que ejecute código, uno que lo revise y uno de unit tests y smoke tests, todo apoyado en SDD u otra metodología actual.

Este documento dice qué se propone, qué se encontró en el repo y **por qué** se recomienda cada decisión.

---

## 1. Qué es SDD y dónde está este proyecto

**Spec-Driven Development:** el spec es la fuente de verdad. El código se deriva del spec y se verifica contra él, nunca al revés.

| Metodología | Idea central | Artefactos |
|---|---|---|
| **GitHub Spec Kit** | specify → plan → tasks → implement | `spec.md`, `plan.md`, `tasks.md` |
| **Kiro (AWS)** | Requisitos en formato EARS ("CUANDO X, el sistema DEBE Y") | `requirements.md`, `design.md`, `tasks.md` |
| **OpenSpec** | Spec vivo + "cambios" que traen un *delta* del spec y se archivan al cerrarse | `specs/` + `changes/<id>/` |

### Hallazgo: el repo ya funciona como OpenSpec

| Concepto OpenSpec | Ya existe en el repo |
|---|---|
| Spec vivo | `project-docs/spec/domain/` (`fiado.md`, `caja.md`, `rentabilidad.md`, `facturacion-dian.md`, `glosario.md`) + `spec/product.md`, `spec/personas.md` |
| Unidad de cambio | `project-docs/sprints/SPRINT-XX-*.md`, que se congela al cerrar (`project-docs/README.md`, tabla "Los cuatro tipos") |
| Sincronizar el delta del spec | CLAUDE.md Regla 2 y el punto 8 del DoD en `project-docs/workflow.md:15` |
| Verificar antes de afirmar | CLAUDE.md Regla 1 (`archivo:línea`) |

**Decisión recomendada:** no instalar Spec Kit ni Kiro. Adoptar el modelo de OpenSpec sobre la estructura actual y agregar solo una carpeta de trabajo temporal por US (`changes/US-XXX/`, fuera de git, ver §3).

**Por qué:** instalar un framework nuevo duplicaría lo que ya existe (`spec/`, sprints, roadmap) y crearía una segunda fuente de verdad. Eso es justo lo que las Reglas 2 y 3 de CLAUDE.md buscan evitar. Lo que falta no es metodología, es **separar roles y poner puntos de control entre ellos**.

---

## 2. Los cuatro agentes

En Claude Code, un subagente es un archivo `.claude/agents/<nombre>.md` que define rol, herramientas permitidas y modelo. Hoy no existe `.claude/agents/`. A nivel de usuario existen los skills `ralph`, `ralph-workflow` y `task-planner` (en `~/.claude/skills/`), que se pueden reutilizar dentro de los prompts.

| # | Agente | Fusiona | Rol | Herramientas | Escribe en |
|---|---|---|---|---|---|
| 1 | `analyst` | spec-keeper + requirements-analyst | Lee `spec/`, el sprint y el código; detecta el delta del spec y redacta `contract.md`. Al cerrar la US, consolida el delta en `spec/` y cierra la US en el sprint | Read, Grep, Glob, Edit, Write, Bash (para `git diff`) | `changes/US-XXX/contract.md`; al cierre, `project-docs/spec/` y `project-docs/sprints/` |
| 2 | `test-author` | — | Escribe los tests **en rojo** a partir del contrato, sin tocar la implementación | Read, Edit, Write, Bash (`go test`, `pnpm test`) | Solo `*_test.go` y `*.test.ts` |
| 3 | `implementer` | — | Escribe código hasta poner los tests en verde | Todas | `internal/`, `frontend/` (no toca tests) |
| 4 | `qa` | code-reviewer + smoke-runner | Revisa el diff contra el contrato y `CLAUDE.md`, corre los comandos de CI y un smoke con el binario | Read, Grep, Glob, Bash, Write (solo `qa.md`) | `changes/US-XXX/qa.md` |

### Por qué 4 y no 6

Cada subagente arranca en frío y vuelve a leer `CLAUDE.md`, el sprint y el código. Con 6 agentes eso pasaba 6 veces por US.
- **spec-keeper y analyst leían exactamente lo mismo.** Separarlos costaba un arranque en frío sin ganar independencia real.
- **reviewer y smoke-runner miran el mismo diff.** Uno lo lee y el otro lo ejecuta, y juntarlos le da a QA ambas evidencias para su veredicto.

Son dos arranques en frío menos por US y un canal de comunicación menos que se puede desalinear.

**Aclaración sobre el Analista:** que sea "el mismo agente" al inicio y al cierre significa **la misma definición, no la misma memoria**. Al cierre arranca en frío y retoma todo desde `contract.md`. Por eso el contrato debe dejar el delta del spec escrito de forma explícita (sección "Delta de spec"), no implícito en la conversación.

### Por qué test-author e implementer siguen separados

Esta es la separación que no se negocia. Si el mismo agente escribe el código y sus tests, los tests terminan describiendo lo que el código *hace*, no lo que el contrato *pide*. Separados:
- los tests salen del contrato, con los números concretos que exige CLAUDE.md (por ejemplo: "3 abonos de $30k + $20k + $10k → total 60.000, sobra 0");
- el implementer no puede editar los tests para "hacerlos pasar", porque su permiso de escritura los excluye.

### Por qué QA tiene Bash y cómo se controla

QA necesita Bash para correr `go test`, `pnpm build` y el binario. El costo es que deja de ser de solo lectura: con `sed -i` o `echo >` podría "arreglar" en silencio lo que debería reportar, y restringir herramientas no lo impide.

**Control:** el orquestador toma `git status --porcelain` y `git diff --stat` antes y después de QA. Si QA dejó cambios en el árbol (excepto `changes/`), se descartan con `git checkout` y se registran como hallazgo. QA reporta, nunca corrige.

### Por qué el orquestador no es un agente

Los subagentes de Claude Code no pueden lanzar otros subagentes. La orquestación vive en el hilo principal (Andrey + la sesión de Claude) y se empaqueta en un comando `/us US-XXX` (`.claude/commands/us.md`).

---

## 3. Flujo de una US

```
PO elige US ──► [0] orquestador     crea worktree + rama us/US-XXX desde dev
                        │           (TODO lo que sigue ocurre dentro del worktree)
                        ▼
                [1] analyst         lee spec/, sprint y código → changes/US-XXX/contract.md:
                        │           - criterios en formato EARS
                        │           - estado verificado con archivo:línea (Regla 1)
                        │           - lista de tests con números concretos
                        │           - "Delta de spec": qué regla cambia y cómo
                        ▼
              ◆ GATE 1 — Andrey aprueba el contrato ◆
                        ▼
                [2] test-author     escribe los tests → confirma que FALLAN (rojo)
                        ▼
                [3] implementer     escribe código → tests en verde
                        ▼
                    git status (snapshot)
                [4] qa              revisa el diff + corre CI + smoke → changes/US-XXX/qa.md
                    git status (¿QA tocó algo? → se descarta y es hallazgo)
                        │
                 ¿BLOQUEANTE? ──sí──► vuelve a [3] (máximo 2 vueltas y luego escala a Andrey)
                        │ no
                        ▼
                [1] analyst         aplica el delta en spec/, pasa los criterios verificados
                                    a la US del sprint, la marca [x]; lo que no entró va a roadmap.md
                        ▼
              ◆ GATE 2 — Andrey aprueba → commit en la rama → merge a dev ◆
                        ▼
                [0] orquestador     borra el worktree (changes/US-XXX/ desaparece con él)
```

### Por qué cada paso

| Paso | Por qué |
|---|---|
| **[0] Worktree primero** | `changes/` está en `.gitignore`, y un worktree nuevo solo trae archivos versionados. Si el contrato se escribiera en `dev`, el test-author y el implementer no lo verían desde su worktree. Creando el worktree antes del Analista, los 4 agentes trabajan en el mismo directorio y ven los archivos ignorados. Además, un implementer que se equivoca no ensucia `dev`. |
| Analista antes de todo | Evita construir sobre una regla de negocio desactualizada. Al planear SPRINT-14, la Regla 1 cazó cuatro afirmaciones falsas heredadas de docs viejos (CLAUDE.md, Regla 1). |
| Contrato con `archivo:línea` | Es donde nace la verificación. Si el contrato está mal, todo lo que viene después está mal. |
| GATE 1 antes de codificar | Corregir un contrato es barato; corregir código ya escrito no. Andrey decide el alcance (CLAUDE.md: "Andrey es Product Owner y decide alcance y prioridad"). |
| Tests en rojo antes del código | Prueba que el test detecta la ausencia de la feature. Un test que pasa sin código no prueba nada. |
| Snapshot de `git status` alrededor de QA | Es la única garantía real de que QA no corrigió nada en silencio (§2). |
| Máximo 2 vueltas | Evita loops infinitos entre implementer y QA. Si en 2 vueltas no converge, el problema está en el contrato, no en el código. |
| Analista al cierre, mismo commit | Es la Regla 2 aplicada de forma mecánica, no dependiendo de que alguien se acuerde. |
| Criterios verificados al sprint | `contract.md` se pierde con el worktree. Lo que vale la pena conservar (los criterios con evidencia) queda en la US del sprint, que sí se commitea. |

### Artefactos y persistencia

```
<worktree de la US>/changes/US-059/    ← en .gitignore
  contract.md    ← analyst
  qa.md          ← qa (una sección por vuelta)
```

**Lo que llega al commit:** el código fuente, los tests, la actualización de `spec/` y el sprint con la US cerrada. Nada más.

**Por qué fuera de git:** `contract.md` y `qa.md` son el canal de comunicación entre agentes, no documentación del producto. Commitearlos llenaría el repo con conversaciones entre agentes, que envejecen igual que los docs que la Regla 1 ya obliga a desconfiar. La evidencia que sí importa sobrevive en tres lugares versionados: los tests, `spec/` y la US del sprint.

---

## 4. Ejemplo: `qa`

```markdown
---
name: qa
description: Revisa el diff de una US contra su contract.md y CLAUDE.md, corre CI y smoke. Reporta, nunca corrige.
tools: Read, Grep, Glob, Bash
model: opus
---
Eres QA. NO editas archivos fuera de changes/<US>/. Si algo falla, lo reportas.
Cualquier cambio tuyo en el árbol será descartado y contado como hallazgo.

1. Revisión (cada hallazgo con archivo:línea), leyendo contract.md y `git diff dev...HEAD`:
- ¿Cada criterio del contrato tiene un test que lo cubra?
- SQL: ¿algún BETWEEN en fechas? ¿concatenación en vez de `?`?
- ¿Errores sin `%w` o silenciados?
- ¿Iteración de map donde importa el orden?
- ¿Se editó una migración ya aplicada en migrate.go?
- Frontend: ¿colores fijos en vez de var(--...)? ¿formatCOP? ¿funciona en 320px?
- ¿Algo de ventas/ganancia/cartera fuera de un grupo RequireAdmin?
- ¿Cambió una regla de negocio sin "Delta de spec" en el contrato?

2. Ejecución: go build ./... · go test ./... · (frontend/) pnpm build · pnpm test ·
   smoke: levantar el binario y hacer curl a los endpoints que toca la US.

Salida: changes/<US>/qa.md con BLOQUEANTE / MENOR / OK y la salida real de cada comando.
```

**Por qué este checklist:** sale literal de las secciones "Arquitectura", "Convenciones" y "Trampas conocidas" de CLAUDE.md. QA no inventa criterio: hace cumplir lo que el PO ya decidió.

---

## 5. Costos y niveles

- **Tokens:** cada agente re-lee el contexto desde cero. Con 4 agentes, una US completa cuesta aproximadamente entre 2.5 y 4 veces más que hacerla directo. El costo que más pesa son las vueltas implementer ↔ QA, no la cantidad de agentes.
- **Velocidad:** baja. **Calidad:** sube. Los gates humanos se mantienen porque son donde el PO decide.

| Nivel | Cuándo | Pipeline |
|---|---|---|
| **S** | Bug fix, cambio de texto | Directo, sin agentes |
| **M** | Cambio en una sola capa, sin regla de negocio | worktree → analyst → implementer → qa |
| **L** | Feature nueva o cambio de regla de negocio | Pipeline completo de 4 |

**Por qué:** usar el pipeline para cambiar un label es desperdicio. Su valor aparece cuando hay reglas de negocio de por medio, sobre todo en el fiado, que es el diferenciador del producto. En M se salta el test-author porque el cambio suele cubrirse ajustando tests existentes; si la US agrega comportamiento nuevo, es L.

---

## 6. Prerrequisitos — resueltos el 2026-09-23

La reorganización de `project-docs/` había dejado punteros rotos en `CLAUDE.md`, `README.md`, `workflow.md`, `roadmap.md` y tres docs de `spec/domain/`, además de tres citas `archivo:línea` que ya apuntaban a líneas equivocadas. Todo se corrigió en el repo. Los skills `ralph-workflow`, `ralph`, `pos-prodcheck` y `task-planner` se actualizaron en `~/.claude/skills/`, **fuera de git**, con respaldo previo.

**Por qué importaba:** si los documentos mienten, los agentes heredan la mentira.

**Pendiente:** las tablas de bugs y el contexto competitivo dentro de `pos-prodcheck/SKILL.md` son de la era MVP. Se arreglaron sus punteros, pero no se re-verificó su contenido.

---

## 7. Plan de ejecución

| Paso | Qué | Por qué en este orden |
|---|---|---|
| 1 | ~~Corregir punteros rotos~~ ✅ hecho 2026-09-23 | Los agentes dependen de que los punteros sean ciertos |
| 2 | ~~Infraestructura del pipeline~~ ✅ hecho 2026-09-23 — ver §8 | La infraestructura mínima del pipeline |
| 3 | Pilotear con **US-059** (SPRINT-14: extraer las métricas del dashboard a un repositorio parametrizable por rango) | Refactor puro, sin feature visible y con tests claros. Calibra los agentes con bajo riesgo antes de algo grande como el módulo de mesas (SPRINT-13) |
| 4 | Retrospectiva del piloto: ajustar prompts, niveles y gates | Los prompts se calibran con uso real, no en papel |
| 5 | Si el piloto funciona: agregar el pipeline a `workflow.md` y volver vigente este documento | Regla 2: si cambia cómo se trabaja, cambia el doc que lo define |

---

## Decisiones del PO

**Tomadas (2026-09-23):**
- [x] 4 agentes: analyst, test-author, implementer, qa.
- [x] `changes/US-XXX/` va en `.gitignore`. Al commit solo llegan el código, los tests, `spec/` y el sprint cerrado.
- [x] Worktree creado al inicio de la US; los 4 agentes trabajan dentro.
- [x] Sección de hallazgos condensada en prerrequisitos, conservando el pendiente de `pos-prodcheck`.

- [x] Se mantienen los dos gates humanos (GATE 1: contrato; GATE 2: merge).
- [x] US-059 es el piloto.

---

## 8. Dónde vive cada pieza

| Pieza | Archivo | Qué hace |
|---|---|---|
| Orquestador | `.claude/commands/us.md` | Comando `/us US-XXX [M\|L]`. Crea el worktree en `../Pos-vertical-wt/US-XXX` (rama `us/US-XXX`), lanza los agentes, verifica con `git` que cada uno solo tocó lo suyo, para en los dos gates y hace el merge `--squash` a `dev` |
| Agente 1 | `.claude/agents/analyst.md` | Modo `contract` (redacta el contrato) y modo `close` (spec, sprint, roadmap) |
| Agente 2 | `.claude/agents/test-author.md` | Tests en rojo, solo archivos `*_test.go` y `*.test.ts(x)` |
| Agente 3 | `.claude/agents/implementer.md` | Código hasta verde; no edita tests |
| Agente 4 | `.claude/agents/qa.md` | Revisión del diff + CI + smoke con el binario; solo escribe `qa.md` |
| Plantilla | `.claude/templates/contract.md` | 8 secciones: objetivo, estado verificado, AC en EARS, tests, archivos, delta de spec, fuera de alcance, preguntas |
| Scratch | `changes/` en `.gitignore` | Vive dentro del worktree y se borra con él |

**Checkpoints en la rama de la US:** `test(US-XXX)` (rojo) → `feat(US-XXX)` (verde) → `docs(US-XXX)` (cierre). Sirven para que el orquestador verifique con `git diff` que el implementer no tocó los tests. A `dev` llega un único commit por el `--squash`.

**Smoke:** `cmd/pos/main.go:21` usa `data.db` del directorio actual y `cmd/pos/main.go:41` fija `localhost:8080`. Por eso QA corre el binario solo desde el worktree (con un `data.db` nuevo y aislado) y omite el smoke si el puerto está ocupado, en vez de matar el POS de Andrey.

**Requisito para correr `/us`:** `dev` sin cambios pendientes en docs, `.claude/` o código. El worktree sale del último commit, así que los agentes no ven lo que no esté commiteado.
