# Dominio — Facturación electrónica DIAN

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** cómo la facturación se relaciona con la venta
> **Actualizado:** 2026-09-21 · Verificado contra `internal/factelect/` en esa fecha.

Este documento cubre **la regla de producto**. El detalle de implementación y la investigación de proveedores viven en `project-docs/reference/facturacion-electronica.md` (antes `fix-gaps.md`) y `project-docs/research/2026-05-dian-proveedores.md` (antes `facElect-finds.md`), ambos de mayo 2026 y marcados como referencia de implementación, no como estado actual.

---

## La excepción a "100% offline"

La facturación electrónica es **la única parte del producto que toca internet**, porque la ley lo exige: la DIAN valida a través de un proveedor autorizado. No hay forma offline de emitir una factura electrónica válida.

**Por eso está diseñada para no bloquear la venta.** La venta se registra y se cobra local e inmediatamente; el documento electrónico se emite después, en segundo plano (`internal/handler/sale.go` dispara la facturación en goroutine tras crear la venta, y un poller en `internal/factelect/poller.go` reintenta).

Si el proveedor está caído, Don Carlos sigue vendiendo. Esa es la regla que no se negocia.

## Activación

Está **apagada por defecto**. `FiscalConfig.Enabled` controla todo: con la bandera en falso, el servicio ni siquiera se construye (`internal/handler/handler.go`, en `New`).

Un tendero que no factura electrónicamente nunca ve nada de esto. Es deliberado: la mayoría del mercado objetivo no está obligada todavía.

## Configuración fiscal

Reside en una sola fila (`FiscalConfig`): NIT, razón social, dirección, CIIU, resolución DIAN, prefijo, rango autorizado desde/hasta, fechas de resolución y vencimiento, credenciales del proveedor y ambiente (pruebas o producción).

El **rango autorizado** y la **fecha de vencimiento de la resolución** son límites legales reales: cuando se agotan o vencen, la DIAN rechaza. Cualquier trabajo futuro sobre facturación debería avisar antes de que ocurra, no después.

## Documentos

Los documentos electrónicos emitidos se guardan en su propio repositorio (`internal/repository/eledoc.go`), separados de `sales`. Una venta puede existir sin documento electrónico; un documento siempre apunta a una venta.

La **nota crédito** existe para anular una venta ya facturada (`POST /sales/{id}/void-with-credit-note`): una factura emitida no se borra, se contrarresta.

---

## Estado y deuda

- **US-034 — validación completa en sandbox de Factus: pendiente desde SPRINT-07.** Requiere acceso externo, no se puede cerrar sin credenciales reales.
- La estructura de código está lista; lo que falta es la validación contra el ambiente de pruebas del proveedor.
