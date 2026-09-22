# project-docs — Índice

> **Actualizado:** 2026-09-21

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
| **¿Qué construimos después?** | [roadmap.md](roadmap.md) — única fuente |
| ¿Cómo se trabaja un sprint? | [BACKLOG-Development-Workflow.md](BACKLOG-Development-Workflow.md) |
| ¿Cuál es el stack y las convenciones de código? | [`../CLAUDE.md`](../CLAUDE.md) |

## Sprints

`SPRINT-XX-*.md`, del 01 al 14. Son unidades de cambio: al cerrarse quedan congelados como registro de qué se construyó y con qué criterios de aceptación.

El estado de cada uno está en la tabla de progreso de [BACKLOG-Development-Workflow.md](BACKLOG-Development-Workflow.md). La deuda abierta está consolidada en [roadmap.md](roadmap.md).

> Los sprints viven en la raíz de `project-docs/` y no en una subcarpeta porque el skill `ralph-workflow` los busca con el glob `project-docs/SPRINT-XX-*.md`.

## Investigación de mercado

Fechados, de lectura, **nunca fuente de verdad sobre qué existe en el código**. Sus afirmaciones de "esto no existe" se verifican con `grep` antes de usarse.

| Documento | Fecha | Qué cubre |
|---|---|---|
| `better-pos.md` | 2026-05-11 | Primeros gaps detectados contra competidores |
| `finds-betters.md` | 2026-05-11 | Features que el tendero pide |
| `find-pos.md` | ~2026-05 | Verticales, rentabilidad, US-040 a US-048 |
| `better-finds-pos.md` | 2026-09-08 | **El más reciente y completo.** Bloques A–K: competencia mundial y local, quick wins, multi-caja LAN (E1), asistente local (E2) |
| `facElect-finds.md` | 2026-05 | Investigación DIAN y proveedores de facturación |

Los cuatro primeros tienen nombres parecidos e indistinguibles entre sí. Si solo vas a leer uno, es **`better-finds-pos.md`**.

## Diseño y arquitectura

| Documento | Estado |
|---|---|
| `DESIGN-Mesas-Pedidos-KDS.md` | Diseño aprobado del módulo restaurante (2026-09-20), insumo de SPRINT-13 |
| `fix-gaps.md` | Plan de implementación de facturación electrónica. **El nombre engaña**: no es sobre "gaps" genéricos. Referencia de implementación, mayo 2026 |
| `ARQUITECTURA.md` | Foto del código al 2026-04-17. Histórico |

## Histórico

| Documento | Por qué no es vigente |
|---|---|
| `PLAN.md` | Checklist de la era MVP (1.298 líneas). Describe código de abril–mayo 2026. Ejemplo concreto: en `:1134` reporta un bug de `cash_in` en el cierre de caja **que ya está arreglado** (`internal/service/register.go:113`) |
| `PRD.md` | PRD del MVP. Las personas siguen vigentes y se extrajeron a `spec/personas.md`; el resto es historia |
| `mvp-context.md` | Visión inicial. Lo vigente se extrajo a `spec/product.md` |

---

## Cómo agregar un documento

1. **¿Es una regla que debe ser cierta siempre?** → `spec/`, y se actualiza con el código.
2. **¿Es lo que viene?** → no crees un documento: va a `roadmap.md`.
3. **¿Es una unidad de trabajo?** → `SPRINT-XX-Tema.md`.
4. **¿Es investigación?** → nombre con fecha, sello `research`, y no se cita como estado del código.

Todo documento nuevo abre con su sello:

```markdown
> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** <tema>
> **Actualizado:** YYYY-MM-DD
```
