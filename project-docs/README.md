# project-docs — Índice

> **Actualizado:** 2026-09-29

**Lee esto antes de creerle a cualquier documento de esta carpeta.** No todos son vigentes: hay specs que deben ser ciertos siempre, investigación fechada de hace meses, e historia del MVP. Antes estaban todos mezclados y escritos en el mismo presente, y eso produjo planes con afirmaciones falsas sobre el código.

Cada documento abre con un sello de estado que dice cuál es cuál.

---

## Los cuatro tipos

| Tipo | Regla | Confiar para |
|---|---|---|
| **spec** | Debe ser cierto siempre. Cambia en el mismo commit que cambia el comportamiento. | Reglas de negocio, decisiones de producto |
| **vigente** | Se mantiene al día. | Qué viene, cómo se trabaja |
| **sprint** | Unidad de cambio. Congelado al cerrarse. | Qué se construyó y con qué criterios |
| **research** / **histórico** | Fue cierto en una fecha. **Nunca** fuente de verdad sobre el código actual. | Contexto, ideas, por qué se decidió algo |

---

## Qué manda sobre qué

| Pregunta | Documento |
|---|---|
| ¿Qué es el producto? ¿Qué NO es? | [spec/product.md](spec/product.md) |
| ¿Para quién escribo esta US? ¿Quién puede ver esto? | [spec/personas.md](spec/personas.md) |
| ¿Cómo se calcula el saldo de un cliente? | [spec/domain/fiado.md](spec/domain/fiado.md) |
| ¿Cómo se cuadra la caja? ¿Qué categorías de gasto hay? | [spec/domain/caja.md](spec/domain/caja.md) |
| ¿Cómo se calcula ganancia y margen? | [spec/domain/rentabilidad.md](spec/domain/rentabilidad.md) |
| ¿Cómo encaja la facturación con el "offline"? | [spec/domain/facturacion-dian.md](spec/domain/facturacion-dian.md) |
| ¿Cómo se dice esto en colombiano? | [spec/domain/glosario.md](spec/domain/glosario.md) |
| ¿Qué es el modo restaurante? ¿Cómo se deriva el estado de una mesa? | [spec/domain/restaurante.md](spec/domain/restaurante.md) |
| **¿Qué construimos después?** | [roadmap.md](roadmap.md) — única fuente |
| ¿Cómo se trabaja un sprint? | [workflow.md](workflow.md) |
| ¿Cuál es el stack y las convenciones de código? | [`../CLAUDE.md`](../CLAUDE.md) |

## Sprints

`sprints/SPRINT-XX-*.md`, del 01 al 14. Son unidades de cambio: al cerrarse quedan congelados como registro de qué se construyó y con qué criterios de aceptación.

El estado de cada uno está en la tabla de progreso de [workflow.md](workflow.md). La deuda abierta está consolidada en [roadmap.md](roadmap.md).

> Los sprints viven en `project-docs/sprints/`. Los skills `ralph-workflow` y `task-planner` los buscan ahí (actualizados el 2026-09-23).

## Investigación de mercado

Fechados, de lectura, **nunca fuente de verdad sobre qué existe en el código**. Sus afirmaciones de "esto no existe" se verifican con `grep` antes de usarse.

| Archivo Actual | Nombre Anterior | Fecha | Qué cubre |
|---|---|---|---|
| [`2026-05-11-gaps-vs-competidores.md`](research/2026-05-11-gaps-vs-competidores.md) | `better-pos.md` | 2026-05-11 | Primeros gaps detectados contra competidores |
| [`2026-05-11-features-pedidas-por-tendero.md`](research/2026-05-11-features-pedidas-por-tendero.md) | `finds-betters.md` | 2026-05-11 | Features que el tendero pide |
| [`2026-05-verticales-y-rentabilidad.md`](research/2026-05-verticales-y-rentabilidad.md) | `find-pos.md` | ~2026-05 | Verticales, rentabilidad, US-040 a US-048 |
| [`2026-09-08-investigacion-mercado.md`](research/2026-09-08-investigacion-mercado.md) | `better-finds-pos.md` | 2026-09-08 | **El más reciente y completo.** Bloques A–K: competencia mundial y local, quick wins, multi-caja LAN (E1), asistente local (E2) |
| [`2026-05-dian-proveedores.md`](research/2026-05-dian-proveedores.md) | `facElect-finds.md` | 2026-05 | Investigación DIAN y proveedores de facturación |

Los nombres anteriores eran parecidos e indistinguibles entre sí. Si solo vas a leer uno de estos documentos, es **[`2026-09-08-investigacion-mercado.md`](research/2026-09-08-investigacion-mercado.md)** (antes `better-finds-pos.md`).

## Diseño y arquitectura

| Archivo Actual | Nombre Anterior | Estado |
|---|---|---|
| [`2026-09-20-mesas-pedidos-kds.md`](design/2026-09-20-mesas-pedidos-kds.md) | `DESIGN-Mesas-Pedidos-KDS.md` | Diseño aprobado del módulo restaurante (2026-09-20), insumo de SPRINT-13 |
| [`tosdd-plan.md`](tosdd-plan.md) | — | **Propuesta** (2026-09-23), pendiente de aprobación: orquestación agéntica con SDD (4 agentes, gates del PO, piloto US-059). No vigente hasta que se apruebe e incorpore a `workflow.md` |
| [`pipeline-bench.md`](pipeline-bench.md) | — | **Vigente.** Benchmark de cada corrida del pipeline `/us`: tokens, tiempo, paradas, defectos encontrados y ajustes aplicados |
| [`facturacion-electronica.md`](reference/facturacion-electronica.md) | `fix-gaps.md` | Plan de implementación de facturación electrónica. **El nombre engañaba**: no es sobre "gaps" genéricos. Referencia de implementación, mayo 2026 |
| [`2026-04-17-arquitectura.md`](archive/2026-04-17-arquitectura.md) | `ARQUITECTURA.md` | Foto del código al 2026-04-17. Histórico |

## Histórico

| Archivo Actual | Nombre Anterior | Por qué no es vigente |
|---|---|---|
| [`2026-05-plan-mvp.md`](archive/2026-05-plan-mvp.md) | `PLAN.md` | Checklist de la era MVP (1.298 líneas). Describe código de abril–mayo 2026. Ejemplo concreto: en `:1134` reporta un bug de `cash_in` en el cierre de caja **que ya está arreglado** (`internal/service/register.go:113`) |
| [`2026-05-prd-mvp.md`](archive/2026-05-prd-mvp.md) | `PRD.md` | PRD del MVP. Las personas siguen vigentes y se extrajeron a `spec/personas.md`; el resto es historia |
| [`2026-05-mvp-context.md`](archive/2026-05-mvp-context.md) | `mvp-context.md` | Visión inicial. Lo vigente se extrajo a `spec/product.md` |

---

## Cómo agregar un documento

1. **¿Es una regla que debe ser cierta siempre?** → `spec/`, y se actualiza con el código.
2. **¿Es lo que viene?** → no crees un documento: va a `roadmap.md`.
3. **¿Es una unidad de trabajo?** → `sprints/SPRINT-XX-Tema.md`.
4. **¿Es investigación?** → nombre con fecha, sello `research`, y no se cita como estado del código.

Todo documento nuevo abre con su sello:

```markdown
> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** <tema>
> **Actualizado:** YYYY-MM-DD
```
