# POS Vertical — Gaps para Primer Cliente
> Generado: 2026-05-11

## Estado general
2 bloqueadores reales · 4 gaps funcionales críticos · Facturación electrónica sin probar en sandbox

---

## A — Bloqueadores de producción

| # | Gap | Ubicación | Impacto real |
|---|-----|-----------|--------------|
| 1 | **Impresión térmica usa `window.print()`** | `frontend/components/TicketPreview.tsx:74` | Don Carlos tiene Epson TM-T20 o Bixolon. `window.print()` abre el diálogo del browser — no imprime en térmica 58mm. Cada venta = frustración. |
| 2 | **Facturación electrónica sin probar en sandbox** | `project-docs/fix-gaps.md` (checklist final, todo `[ ]`) | El código compila pero nadie ha llamado a Factus. Los nombres de campos del payload son estimaciones (`numbering_range_id`, `tribute_id`). Si el cliente necesita emitir facturas, esto falla en producción. |

**Nota JWT:** `internal/middleware/middleware.go:22` tiene fallback hardcoded pero ya lee `POS_JWT_SECRET` env var. No es bloqueante para cliente local — documenta que deben setear la variable antes de instalar.

---

## B — Gaps funcionales (primeros 30 días)

| Prioridad | Gap | Por qué duele |
|-----------|-----|---------------|
| 🔴 Alta | **Descuento por ítem en POS** | El tendero da descuento a clientes frecuentes. Sin esto hacen matemática a mano o usan otra solución. |
| 🔴 Alta | **Precio mayorista sin mecanismo de activación** | El campo existe en BD, pero es invisible en UI. Don Carlos tiene clientes mayoristas — les cobra lo mismo que al minorista. |
| 🟡 Media | **Reactivar usuario desactivado** | Solo se puede desactivar. Si alguien desactiva un usuario por error, no hay salida desde UI. |
| 🟡 Media | **Filtro sesión en movimientos de caja** | Muestra todos los movimientos históricos, no solo los de la sesión abierta. Confunde al cajero al cuadrar caja. |

---

## C — Brecha vs competidores (lo que importa para primer cliente)

| Ellos tienen | Este POS | Veredicto |
|---|---|---|
| Facturación electrónica lista y probada (Alegra) | Código listo, sin probar en Factus | 🔴 Bloqueante si cliente es distribuidora o emite facturas |
| Thermal printing nativo (Fenix POS) | `window.print()` browser | 🔴 Bloqueante para cualquier cliente con impresora |
| App móvil para dueño (Alegra) | Responsive web funcional | 🟡 Cubre el caso básico |
| Internet requerido (Alegra, Loggro) | 100% offline | ✅ Ventaja fuerte |
| Mensualidad $80K–$200K COP/mes (todos) | Licencia única potencial | ✅ Diferenciador de precio |
| Fiado como afterthought (todos) | Fiado como feature principal | ✅ Diferenciador de producto |

---

## D — Diferenciadores para profundizar

El fiado sigue siendo el núcleo — ningún competidor barato lo hace bien. Para hacerlo más poderoso:

1. **Notas por cliente** — "no fiar más de $150K", "paga los viernes". Campo `notes` en perfil del cliente. ~2 horas de trabajo, impacto enorme para Don Carlos.
2. **Alerta de tope de crédito en POS** — antes de confirmar una venta a crédito, mostrar si el cliente superó su límite configurado. No existe en ningún competidor barato.
3. **Precio mayorista automático** — activar `wholesale_price` por tipo de cliente o por volumen. El campo existe, falta la lógica de cuándo aplicarlo.

---

## Recomendación inmediata (esta semana)

**En este orden:**

1. **ESC/POS thermal** — pregunta qué impresora tiene el primer cliente target. Si es Epson TM-T20 (la más común en Colombia), implementa ESC/POS via WebUSB o via un servidor local en un puerto. Sin esto, el POS no es un POS para nadie con impresora.

2. **Factus sandbox** — si el cliente no necesita factura electrónica (tienda de barrio), puede esperar. Si sí la necesita, verificar el payload contra `developers.factus.com.co` antes de instalar.

3. **Descuento por ítem** — 1–2 días de trabajo, muy pedido en campo.

**Conclusión:** Si el primer cliente es una tienda de barrio sin impresora térmica y sin obligación de factura electrónica, se puede entregar esta semana. Si tiene impresora, no.
