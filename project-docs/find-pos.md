# POS Vertical — Research de Mercado y Diferenciadores

> **Tipo:** research · **Vigente:** NO · **Fecha:** ~mayo 2026
> Investigación de verticales y rentabilidad. **Sus afirmaciones de "esto no existe" se verifican con `grep` antes de usarse.** El doc de investigación más reciente y completo es `better-finds-pos.md`. Lo que viene está en [roadmap.md](roadmap.md).

> Última actualización: 2026-05-14
> Fuentes: Reddit (r/POS, r/smallbusiness, r/Entrepreneur), WebSearch (24 páginas), research LATAM fintechs

---

## Contexto del público objetivo

Tres nichos con necesidades distintas — todos sub-atendidos por el mercado barato:

| Nicho | Dolor principal | Diferenciador ganador |
|---|---|---|
| **Tienda de barrio** | Fiado sin control, sin internet confiable | Offline-first + fiado inteligente |
| **Ferretería** | Miles de SKUs, unidades mixtas, contratistas | Inventario granular + cuentas corrientes |
| **Maquillaje / belleza** | Variantes de tono/presentación, tendencias TikTok | Variantes sin duplicar SKU + alertas de rotación |

---

## Lo que la gente dice que le falta en los POS actuales

### Inventario
- Básico a menos que pagues plan premium (Square, Shopify POS)
- Lectura de chip y tap-to-pay falla dentro de 6 meses de uso (queja recurrente en Reddit)
- Sin alertas inteligentes — avisan "stock bajo" pero no "te quedas sin esto en 2 días"
- No manejan unidades mixtas (metro, litro, docena, unidad del mismo producto) — crítico para ferreterías
- Variantes de tono/color/presentación duplican SKUs manualmente — crítico para maquillaje

### POS / Cobro
- Interfaces lentas: "takes minutes to ring up products during busy times"
- Sin precio por tipo de cliente (mayorista, contratista, especial) en el flujo rápido
- Sin descuento por ítem — los tenderos descuentan a clientes frecuentes a ojo

### Fiado y crédito
- El fiado es un afterthought en todos los competidores baratos (Alegra, Loggro, Square)
- No hay gestión de "cuentas corrientes" para contratistas en ferreterías
- No muestran antigüedad de la deuda — solo el saldo total
- No avisan antes de vender a alguien que ya debe demasiado

### Reportes y rentabilidad
- Muestran ventas brutas, no ganancia real
- No calculan margen por producto (requiere campo de costo)
- No detectan productos sin rotación (capital muerto)
- No priorizan clientes con deuda urgente

### Infraestructura
- Cloud-only: sin internet, no funciona — crítico en Colombia
- Hardware costoso: terminales caros, impresoras requeridas
- Updates que rompen cosas sin aviso

---

## Lo que la gente AMA cuando lo tiene

1. **Dashboard accionable simple** — no gráficas: números del día de un vistazo
2. **Alertas de restock automáticas** basadas en velocidad de venta real
3. **Loyalty ultra-simple** — descuento directo sin puntos ni apps
4. **Historial del cliente en pantalla al cobrar** — saber quién es antes de cobrar
5. **Reportes de rentabilidad** — margen real, no solo ventas
6. **Funciona offline** — sincroniza cuando hay señal

---

## Lo que ya tiene este POS (SPRINT-07)

| Feature | US | Estado |
|---|---|---|
| Dashboard "cuaderno digital" (vendido, fiado, cobrado, efectivo, top producto) | US-038 | ✅ |
| Notas y tope de crédito por cliente | US-036 | ✅ |
| Precio mayorista activable en carrito | US-037 | ✅ |
| WhatsApp cobro con link directo | US-035 | ✅ |
| Facturación electrónica (Factus) | US-033/034 | 🔄 En validación sandbox |

---

## Huecos del mercado — nadie lo hace barato

### Hueco 1 — Rentabilidad real
**Qué falta:** mostrar ganancia bruta (ventas − costo de lo vendido) y margen por producto.

**Requisito técnico:** campo `cost_price` en tabla `products`. Si ya existe, el cálculo es trivial.

```
Vendido hoy:         $340.000
Costo de lo vendido: $198.000
Ganancia bruta:      $142.000   ← esto es lo que el tendero quiere ver
Margen:              41.8%

Top rentables:
1. Aceite de cocina  → 38% margen │ 24 unidades
2. Pañales Huggies   → 22% margen │ 41 unidades
3. Arroz 1kg         → 11% margen │ 89 unidades  ← más vendido ≠ más rentable
```

**Impacto:** alto. Ningún competidor en el segmento low-cost colombiano lo muestra así.

---

### Hueco 2 — Alertas de stock por velocidad de venta
**Qué falta:** no alertar cuando hay "3 unidades" sino cuando "a este ritmo te quedas sin X en 2 días."

**Lógica:** `stock_actual / promedio_ventas_diarias_7dias = días restantes`. Si < umbral → alerta.

**Impacto:** medio-alto. Simple de calcular, muy diferenciador vs "stock mínimo" estático.

---

### Hueco 3 — Historial del cliente en el POS al cobrar
**Qué falta:** al seleccionar un cliente en el carrito, mostrar:
- Última visita: hace X días
- Lo que suele llevar: "Arroz + Leche + Aceite"
- Deuda actual + antigüedad ("debe $80.000 desde hace 22 días")

**Nota:** ya tienes notas y tope de crédito (US-036). Esto es el siguiente paso natural.

---

### Hueco 4 — Reporte de cartera envejecida
**Qué falta:** no solo cuánto debe cada cliente sino hace cuánto lleva sin pagar.

```
Cartera vencida:
Pedro Ramírez    $80.000   22 días   ← cobrar urgente
Ana García       $45.000    8 días
Luis Martínez    $20.000    3 días
```

**Requisito técnico:** guardar fecha de la última compra a crédito por cliente (no solo el saldo).

---

### Hueco 5 — Variantes de producto (maquillaje y ferretería)
**Qué falta:** un producto con múltiples presentaciones sin duplicar SKU.

Maquillaje: Labial MAC → Tono Ruby Woo / Tono Velvet Teddy / Tono Chili — mismo producto, stock separado por variante.

Ferretería: Cable eléctrico → por metro / por rollo 100m / por rollo 50m — mismo producto, precio por unidad de medida.

**Impacto:** alto para estos nichos. Nadie en el segmento low-cost colombiano lo tiene.

---

### Hueco 6 — Sugerencia de pedido al proveedor
**Qué falta:** "basado en tus ventas de esta semana, deberías pedir: 2 cajas de Arroz Diana, 1 caja de Aceite, 3 paquetes de Pañales T3"

**Lógica:** productos donde `stock_actual < promedio_semanal * 2` → aparecen en lista de pedido sugerido.

**Impacto:** alto. Completamente diferenciador. Ningún POS low-cost colombiano lo hace.

---

### Hueco 7 — Catálogo compartible por WhatsApp
**Qué falta:** link público con productos + precios para enviar a clientes desde el sistema.

`pos.negocio.com/catalogo/abc123` → página simple con productos habilitados, precio, foto opcional.

**Extensión natural de US-035** (WhatsApp cobro). Mismo flujo, diferente propósito.

**Impacto:** medio-alto para maquillaje y ferreterías que venden a clientes frecuentes remotos.

---

### Hueco 8 — Offline-first real
**Qué falta:** funcionar completamente sin internet y sincronizar al reconectar.

**Estado actual del POS:** 100% offline (ventaja ya existente vs Alegra, Loggro).
**Próximo paso:** hacer explícito en UI cuándo está offline vs online, y cuándo fue el último sync si aplica.

---

## Competidores y sus puntos débiles

| Competidor | Fortaleza | Debilidad explotable |
|---|---|---|
| **Alegra** | Facturación electrónica probada, app móvil | Cloud-only, $80K-200K/mes, fiado básico |
| **Loggro** | Integración contable | Cloud-only, complejo para tendero |
| **Square** | Free tier, hardware confiable | Sin soporte DIAN, sin fiado, sin español CO |
| **Shopify POS** | Omnicanal potente | Caro, "not designed for real retail" (Reddit 2026) |
| **NabePOS** | SaaS colombiano específico para tiendas de barrio | SaaS (requiere internet + mensualidad) |
| **Fenix POS** | Thermal printing nativo | Sin diferenciadores modernos |

**Ventaja no replicable fácil:** offline + fiado inteligente + precio único (no mensualidad).

---

## Roadmap sugerido — próximos 3 sprints

### SPRINT-08 — Rentabilidad y cartera

| US | Feature | Esfuerzo | Impacto |
|---|---|---|---|
| US-039 | `cost_price` en productos + ganancia bruta en dashboard | 2 días | 🔴 Alto |
| US-040 | Top productos por margen (no solo por ventas) | 1 día | 🔴 Alto |
| US-041 | Cartera envejecida — deuda + días desde última compra crédito | 2 días | 🔴 Alto |
| US-042 | Alerta de stock por velocidad de venta (días restantes) | 2 días | 🟡 Medio |

### SPRINT-09 — Historial de cliente y pedidos

| US | Feature | Esfuerzo | Impacto |
|---|---|---|---|
| US-043 | Historial + "lo de siempre" del cliente en POS | 2 días | 🔴 Alto |
| US-044 | Sugerencia de pedido al proveedor | 3 días | 🔴 Alto |
| US-045 | Catálogo compartible por WhatsApp (link público) | 3 días | 🟡 Medio |

### SPRINT-10 — Nichos ferretería / maquillaje

| US | Feature | Esfuerzo | Impacto |
|---|---|---|---|
| US-046 | Variantes de producto (tono, talla, unidad de medida) | 4 días | 🔴 Alto para nicho |
| US-047 | Cuentas corrientes para contratistas (ferretería) | 3 días | 🔴 Alto para nicho |
| US-048 | Productos sin rotación — reporte de capital muerto | 1 día | 🟡 Medio |

---

## Argumento de venta por nicho

**Tienda de barrio:**
> "El único POS que maneja el fiado como usted lo hace en su cabeza, funciona sin internet, y avisa por WhatsApp cuando un cliente debe."

**Ferretería:**
> "Controla miles de productos, vende por metro o por rollo, y maneja las cuentas de sus contratistas sin planillas."

**Maquillaje:**
> "Inventario por tono y presentación, alerta cuando se te acaba lo que más vendes, y catálogo listo para enviar por WhatsApp."
