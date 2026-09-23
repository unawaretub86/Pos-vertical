> **Tipo:** vigente · **Manda sobre:** qué falta para producción · **Actualizado:** 2026-09-21

Este documento lista limitaciones y deuda **verificadas** del producto actual; lo que se va a construir a futuro vive en [roadmap.md](roadmap.md), que es otra cosa.

## Deuda abierta (no es roadmap, es lo que quedó a medias)

| US / Sprint | Qué | Desde |
|---|---|---|
| US-034 | Factus: validación completa en sandbox | SPRINT-07 — requiere acceso externo |
| SPRINT-05 | Mobile Responsive, marcado pendiente aunque la base de estilos ya se usa | SPRINT-05 |
| US-042c, US-043, US-045 | Velocidad de venta, historial del cliente en POS, catálogo por WhatsApp | SPRINT-09 |
| US-053 | Resumen de cierre de caja por correo — hay trabajo sin commitear en `internal/mailer/` | SPRINT-12 |
| US-054 | Acceso al POS por WiFi local | SPRINT-12, reasignada a SPRINT-13 |
| US-055 a US-058 | Módulo restaurante completo | SPRINT-13 |
| US-059 a US-062 | Asistente local | SPRINT-14 |

## Documentos con información desactualizada

| Gap | Evidencia | Estado |
|---|---|---|
| `archive/2026-05-plan-mvp.md` reporta un bug de `cash_in` en el cierre de caja | `archive/2026-05-plan-mvp.md:1134` vs `internal/service/register.go:113` | **Ya arreglado**, el doc es el que está desactualizado |

## Por verificar

Aquí van los gaps sospechados pero no confirmados contra el código. Cada uno se confirma con `archivo:línea` antes de subir a la tabla principal.
