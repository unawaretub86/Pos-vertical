# Roadmap — Pos-vertical

> **Tipo:** vigente · **Manda sobre:** qué se construye después · **Actualizado:** 2026-09-21

Única fuente de verdad sobre lo que viene. Antes existía una tabla de Roadmap al final de cada documento de sprint; se copiaban entre sí, divergieron, y terminaron reclamando el mismo número de sprint para cosas distintas. **Los documentos de sprint ya no llevan roadmap propio ni reservan números futuros.**

Cada ítem tiene un ID estable (`R-NN`) que no cambia aunque se mueva de bucket.

---

## Cómo se lee

| Bucket | Significa |
|---|---|
| **Siguiente** | Candidato al próximo sprint. Andrey elige entre estos. |
| **Listo para planear** | Sin bloqueo técnico. Entra cuando haya presupuesto de sprint. |
| **Bloqueado** | Depende de algo que todavía no existe. La dependencia está nombrada. |
| **Sin fecha** | Válido pero sin señal de que bloquee la venta al primer cliente. |
| **Requiere tercero** | No ejecutable en solitario. |

---

## Siguiente

| ID | Ítem | Origen | Nota |
|---|---|---|---|
| R-01 | Continuación del módulo restaurante: para llevar con turno diario, modificadores con precio | SPRINT-13 | El turno es media US sobre tablas ya creadas. Los modificadores tocan catálogo y precios. **Solo tiene sentido después de cerrar SPRINT-13**, que está pendiente. |
| R-02 | Continuación del asistente: intent `PRODUCT_LOOKUP` ("¿cuánto queda de arroz?") | SPRINT-14 | Requiere extracción de entidad y desambiguación contra el catálogo. US completa, no un intent más. Depende de cerrar SPRINT-14. |
| R-03 | Referencia temporal "la quincena" en el asistente | SPRINT-14 | **Condicionado a la validación con Don Carlos.** Si la nombra por su cuenta, no es una fecha más: es el ciclo de cobro del fiado (15 y 30) y probablemente se pide junto a `DEBTORS`. Sería la referencia temporal de mayor valor. |

## Listo para planear

| ID | Ítem | Origen | Nota |
|---|---|---|---|
| R-10 | G8 — reportes por hora pico y por vendedor | `research/2026-09-08-investigacion-mercado.md:167` | `report.go` no agrupa por hora ni por `user_id`. ~0.5 día: dos queries `GROUP BY`. Desbloquea R-20 y es base para comisiones. |
| R-11 | F7 + I4 — PIN rápido por cajero y permisos granulares | `research/2026-09-08-investigacion-mercado.md` (bloque F) | Cambio de turno sin logout completo. Relevante cuando el negocio pasa de 1 empleado. |
| R-12 | Propina, dividir cuenta, transferir ítems entre mesas | SPRINT-13 | Complican el cobro, que en SPRINT-13 recién se estabiliza. Solo si el cliente los pide. |
| R-13 | Pantalla "qué me preguntan y no entiendo" | SPRINT-14 | Lee `assistant_queries`. Necesita semanas de uso real acumuladas para mostrar algo. |
| R-14 | Fechas exactas en el asistente ("15 de abril", "marzo", "el martes") | SPRINT-14 | Entra cuando la bitácora demuestre que se piden cuadres históricos. |
| R-15 | H2 — apartados / layaway | `research/2026-09-08-investigacion-mercado.md` (bloque H) | Extiende el diferenciador de fiado. El doc fuente lo marca como pedido del mes 2, no del día 1. |
| R-16 | G2 — kardex y ajustes de inventario | `research/2026-09-08-investigacion-mercado.md` (bloque G) | "Requisito 2026" según el doc fuente. |
| R-17 | Scoring interno del cliente fiado | `research/2026-09-08-investigacion-mercado.md` (bloque D) | Score de riesgo, límite de crédito sugerido y alerta temprana de mora, con el historial de abonos que ya se guarda. **Ningún competidor local lo hace.** Complejidad media, cero dependencia externa. Es el diferenciador más fuerte sin construir. |
| R-18 | E1 — sync multi-caja por LAN offline | `research/2026-09-08-investigacion-mercado.md` (bloque E1) | 2–5 cajas, mismo inventario, sin nube. La parte dura es resolución de conflictos. Relevante cuando el cliente crece a multi-caja. |

## Bloqueado

| ID | Ítem | Bloqueado por |
|---|---|---|
| R-20 | Intents `PEAK_HOURS` y `BY_SELLER` en el asistente | R-10 (G8). Cuando exista el agrupamiento por hora y vendedor, cada intent es una fila más en el registro. |
| R-21 | Intents `DEAD_STOCK` y `REORDER` en el asistente | US-048 (capital muerto) y US-044 (sugerencia de pedido), ambas pendientes de SPRINT-09. |
| R-22 | US-044 — sugerencia de pedido al proveedor | Desbloqueada desde el cierre de SPRINT-11 (US-052 dio proveedores). Pendiente de programar; pertenece a SPRINT-09, que sigue abierto. |

## Sin fecha

| ID | Ítem | Por qué |
|---|---|---|
| R-30 | G3–G7 — vencimientos, combos, etiquetas, historial de precio, foto de producto | Features de nicho (farmacia, ferretería). Sin señal de que bloqueen la primera venta. |
| R-31 | H3–H5 — domicilios, impuesto de bolsas, cotizaciones | Igual: nicho o volumen que el cliente actual no tiene. |
| R-32 | Plano visual de mesas arrastrable, rol "mesero", comanda impresa, KDS por SSE | Cosmético o dependiente de que el cliente crezca. El plano vende en demo pero no resuelve lo que la lista no resuelva. |
| R-33 | Tier de modelo de lenguaje local en el asistente | Condicionado a dos cosas: que `assistant_queries` muestre volumen real de preguntas fuera de dominio, **y** que exista una vía sin instalar nada en el PC del cliente. Si entra, traduce texto a `{intent, periodo}` y nunca genera cifras. |
| R-34 | App móvil nativa del dueño | `K1` (acceso LAN, US-054) cubre el caso dentro del local. Fuera del local requiere nube, que contradice el posicionamiento. |

## Requiere tercero

| ID | Ítem | Qué falta |
|---|---|---|
| R-40 | H1 — recargas y pago de servicios | Alianza con agregador (Movilred / Puntored / SuRed). |
| R-41 | Capital embebido (préstamo sobre data de ventas) | Partner fintech regulado por la SFC. Los tres líderes mundiales ya lo venden; en Colombia nadie lo hace a nivel tienda de barrio. Visión de largo plazo, no se persigue en solitario. |
| R-42 | K3 — envío automático por WhatsApp API o SMTP | Cuenta business y costo recurrente. Contradice "offline sin mensualidad". Posible add-on pago si un cliente lo pide. |
| R-43 | K4 — acceso remoto en vivo (tunnel o nube) | 5+ días y rompe el posicionamiento offline. Solo cuando un cliente pregunte "¿y desde la casa?". |

---

## Deuda abierta

La deuda y las limitaciones verificadas del producto están en [gaps.md](gaps.md).
Este documento cubre lo que se va a construir, no lo que falta por cerrar.

---

## Bitácora de decisiones

Cambios de rumbo que no se deducen de ningún otro documento.

| Fecha | Decisión | Por qué |
|---|---|---|
| 2026-09-21 | El asistente local (E2) sube de "Sin fecha" a SPRINT-14 | Gancho de venta inmediato y, con motor de reglas en vez de LLM, cero riesgo en hardware de 4GB. |
| 2026-09-21 | El asistente **no** lleva LLM en V1, ni siquiera vía Ollama | Requisito del PO: nada que instalar ni descargar en el PC del cliente. Un modelo local queda en R-33 condicionado a datos de uso. |
| 2026-09-21 | SPRINT-14 tomó el número que SPRINT-13 había reservado para la continuación del restaurante | El desplazamiento no se registró en su momento; R-01 recoge lo diferido. Este roadmap existe para que no vuelva a pasar. |
| 2026-09-08 | Capital embebido se descarta como línea propia | Requiere partner fintech regulado. Solo sobrevive el scoring interno (R-17), que sí es ejecutable en solitario. |
