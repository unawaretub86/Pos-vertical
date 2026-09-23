---
description: Ejecuta una US con el pipeline SDD de 4 agentes (analyst → test-author → implementer → qa → analyst) y dos gates del PO
argument-hint: US-XXX [M|L]
---

Orquesta la US **$ARGUMENTS** según `project-docs/tosdd-plan.md`. Tú eres el orquestador: lanzas los subagentes, verificas lo que hacen y paras en los gates. **No escribes código, tests ni docs tú mismo.**

Si no se indica nivel, usa **L**. En nivel **M** se salta el paso 2 (test-author).

Convenciones de este comando:
- `WT` = ruta absoluta del worktree de la US.
- `ID` = el identificador de la US (`US-XXX`).
- Todo comando git va con `git -C "$WT"`.
- A cada subagente le pasas en el prompt: el `ID`, la ruta absoluta de `WT`, el sprint y el modo (si aplica).

## 0. Preparar el worktree

1. En el checkout principal, `git status --porcelain`. Si hay cambios sin commitear en `CLAUDE.md`, `project-docs/`, `.claude/` o en el código, **detente y avisa a Andrey**: el worktree sale del último commit de `dev` y los agentes no verían esos cambios.
2. Ubica la US con Grep en `project-docs/sprints/` y confirma que sigue `[ ]`.
3. Crea el worktree: `git worktree add "../Pos-vertical-wt/$ID" -b "us/$ID" dev`. Si la rama ya existe, pregunta a Andrey si se retoma o se descarta.
4. `mkdir -p "$WT/changes/$ID"`.

## 1. Contrato — agente `analyst`, modo `contract`

Lanza `analyst`. Luego verifica que existe `$WT/changes/$ID/contract.md` y que tiene las 8 secciones de la plantilla.

### ◆ GATE 1 — Andrey aprueba el contrato

Muestra el resumen del analyst, las DISCREPANCIAS y las preguntas para el PO, con la ruta del contrato. **Detente y espera.** Si Andrey pide cambios, vuelve a lanzar `analyst` con sus correcciones. No avances sin un "sí" explícito.

## 2. Tests en rojo — agente `test-author` (solo nivel L)

Lanza `test-author`. Luego verifica:
- `git -C "$WT" status --porcelain` solo muestra `*_test.go`, `*.test.ts` o `*.test.tsx` (y `changes/`, que está ignorado). Si hay otro archivo, revierte con `git -C "$WT" checkout -- <archivo>` (o bórralo si es nuevo) y relanza al agente indicando la violación.
- Su tabla no tiene tests `PASA⚠️`. Si los hay, muéstralos a Andrey antes de seguir.

Checkpoint: `git -C "$WT" add -A && git -C "$WT" commit -m "test($ID): red tests from contract"`. Guarda el hash como `RED`.

## 3. Implementación — agente `implementer`

Lanza `implementer`. Luego verifica:
- `git -C "$WT" diff RED -- '*_test.go' '*.test.ts' '*.test.tsx'` está vacío (en nivel M, compara contra el commit base de la rama). Si no lo está, revierte esos archivos y trátalo como BLOQUEANTE de la vuelta.

Checkpoint: `git -C "$WT" commit -am "feat($ID): implementation"` (agrega los archivos nuevos con `add -A` antes).

## 4. QA — agente `qa`

1. Snapshot: `git -C "$WT" status --porcelain` y `git -C "$WT" rev-parse HEAD`.
2. Lanza `qa` e indícale el número de vuelta y el hash `RED`.
3. Snapshot de nuevo. Si QA cambió algo fuera de `changes/`, revierte con `git -C "$WT" checkout -- .` (y `git -C "$WT" clean -fd -e changes/` para archivos nuevos) y anota en `qa.md` la violación.
4. Lee el veredicto:
   - **BLOQUEANTE** y es la vuelta 1 o 2 → vuelve al paso 3 pasándole al implementer solo los BLOQUEANTE de esa vuelta.
   - **BLOQUEANTE** en la vuelta 3 → **detente y escala a Andrey**. Dos vueltas sin converger indican que el problema está en el contrato.
   - **OK** → paso 5.

## 5. Cierre — agente `analyst`, modo `close`

Lanza `analyst` en modo `close`. Luego verifica con `git -C "$WT" status` que solo cambió `project-docs/` y haz el checkpoint: `git -C "$WT" commit -am "docs($ID): spec delta and sprint closure"`.

### ◆ GATE 2 — Andrey aprueba

Muestra:
- `git -C "$WT" diff dev...HEAD --stat`;
- el veredicto final de QA con los MENOR pendientes;
- los cambios de spec, sprint y roadmap.

**Detente y espera.** Con un "sí" explícito:

```
git checkout dev
git merge --squash "us/$ID"
git commit        # mensaje: feat($ID): <nombre de la US>, más el trailer Co-Authored-By
git worktree remove "../Pos-vertical-wt/$ID"
git branch -D "us/$ID"
```

`changes/$ID/` desaparece con el worktree; es lo esperado. **No hagas push** a menos que Andrey lo pida.

## Reglas del orquestador

- Nunca saltes un gate. Nunca hagas merge sin el "sí" del GATE 2.
- No corrijas el trabajo de un agente con tus propias manos: relánzalo con el motivo.
- Si un agente reporta algo que el contrato no resuelve, pregúntale a Andrey; no decidas alcance.
- Reporta fielmente: si un paso se omitió o falló, dilo.
