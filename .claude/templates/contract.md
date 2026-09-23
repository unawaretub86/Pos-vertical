# Contrato — US-XXX: <nombre>

> **Sprint:** `project-docs/sprints/SPRINT-XX-*.md` · **Nivel:** M | L
> **Worktree:** <ruta absoluta> · **Rama:** us/US-XXX · **Fecha:** YYYY-MM-DD

## 1. Objetivo

Una o dos frases: qué cambia para el usuario (o para el dev, si es refactor) y por qué.

## 2. Estado verificado

Lo que existe HOY, con `archivo:línea`. Nada de "según el doc".

| Afirmación | Evidencia |
|---|---|
| El dashboard calcula ventas del día con SQL inline | `internal/handler/dashboard.go:NN` |

Si el sprint afirma algo que el código contradice, anotarlo aquí como **DISCREPANCIA** y levantarlo en el GATE 1.

## 3. Criterios de aceptación (EARS)

Formato: `CUANDO <disparador>, el sistema DEBE <respuesta>` · `MIENTRAS <estado>, el sistema DEBE …` · `SI <condición no deseada>, ENTONCES el sistema DEBE …`

- **AC-1** CUANDO … , el sistema DEBE … .
- **AC-2** SI … , ENTONCES el sistema DEBE … .

## 4. Tests requeridos

Cada AC tiene al menos un test. Números concretos, nunca aserciones vagas (CLAUDE.md, Tests).

| ID | Cubre | Archivo | Caso (entrada → salida esperada) |
|---|---|---|---|
| T-1 | AC-1 | `internal/service/x_test.go` | 3 abonos de $30.000 + $20.000 + $10.000 → saldo 60.000 |
| T-2 | AC-2 | `internal/handler/x_test.go` | venta a las 23:59:59 y otra a las 00:00:00 → rango `[from, to)` cuenta solo la primera |

Fixture: `setupTestDB(t)` con `database.New(":memory:")` y seed por `INSERT` (`internal/handler/dashboard_test.go:14`).

## 5. Archivos a tocar

| Archivo | Cambio | Quién |
|---|---|---|
| `internal/repository/x.go` | nuevo método `Y` | implementer |
| `internal/repository/x_test.go` | T-1 | test-author |

**Migración:** ninguna | migración N (verificar el último número en `internal/database/migrate.go` antes de escribirla).

## 6. Delta de spec

Qué regla de `project-docs/spec/` cambia. Si no cambia ninguna, escribir **"Sin cambio de spec"** y por qué.

- `spec/domain/<archivo>.md` § <sección>: antes «…» → después «…».

## 7. Fuera de alcance

Lo que parece relacionado pero NO entra. Lo que merezca hacerse va a `project-docs/roadmap.md` al cierre.

## 8. Preguntas para el PO (GATE 1)

- [ ] …
