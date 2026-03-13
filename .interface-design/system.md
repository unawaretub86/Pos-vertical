# Sistema de Diseño — POS Genérico

## Referencia Visual

El prototipo completo está en `/pos-prototype.html`. Contiene 7 pantallas navegables con modo claro/oscuro funcional. **Esta es la fuente de verdad visual** antes de que exista el frontend real.

Pantallas incluidas:
- Punto de Venta (POS) — pantalla principal
- Dashboard
- Caja Registradora
- Clientes / Cartera
- Inventario
- Reportes
- Configuración

---

## Dirección y Personalidad

**¿Quién lo usa?**
- **Laura (cajera):** en el mostrador, procesa 50–150 ventas/día. Necesita velocidad. Vive en la pantalla de POS y Caja.
- **Don Carlos (dueño):** revisa reportes, gestiona el fiado, cuadra la caja. Necesita claridad y confianza en los números.

**¿Cómo debe sentirse?**
Moderno y a la vanguardia — a la altura de herramientas como Linear o Vercel — pero cálido y confiable. No es una app de Silicon Valley, es la herramienta de trabajo de un negocio colombiano. Profesional sin ser frío.

**Tema:** Claro u oscuro, seleccionable por el usuario. Persiste en `localStorage`.

---

## Paleta de Colores

### Modo Oscuro
```css
--canvas:          #0D0D0D
--surface-1:       #161616
--surface-2:       #1E1E1E
--surface-3:       #252525
--border:          rgba(255,255,255,0.07)
--border-emphasis: rgba(255,255,255,0.13)
--text-primary:    #F0EDE8
--text-secondary:  #9B9691
--text-muted:      #5C5954
--accent:          #F5A623
--accent-dim:      rgba(245,166,35,0.12)
--success:         #4CAF82
--success-dim:     rgba(76,175,130,0.12)
--danger:          #E05454
--danger-dim:      rgba(224,84,84,0.12)
--warning:         #F5A623
```

### Modo Claro
```css
--canvas:          #F5F3EF
--surface-1:       #FEFCF9
--surface-2:       #F0EDE8
--surface-3:       #E8E4DE
--border:          rgba(0,0,0,0.07)
--border-emphasis: rgba(0,0,0,0.13)
--text-primary:    #1A1714
--text-secondary:  #5C5550
--text-muted:      #9B9691
--accent:          #D4890A
--accent-dim:      rgba(212,137,10,0.10)
--success:         #2D8A5E
--success-dim:     rgba(45,138,94,0.10)
--danger:          #C23B3B
--danger-dim:      rgba(194,59,59,0.08)
--warning:         #D4890A
```

### Por qué Amber-Gold como acento
El color de los billetes colombianos. Transmite confianza en el manejo del dinero. Cálido sin ser informal. Se usa SOLO para: botones primarios, estado activo en nav, total en el carrito, y elementos de marca.

---

## Tipografía

**Fuente:** Inter (Google Fonts)

| Uso | Tamaño | Peso | Observación |
|-----|--------|------|-------------|
| Títulos de página | 20px | 600 | |
| Encabezados de sección | 14px | 600 | letter-spacing: -0.01em |
| Cuerpo / labels | 13px | 400 | |
| Labels pequeños | 12px | 500 | |
| **Montos / Cifras** | variable | 600 | `font-variant-numeric: tabular-nums` |

**Regla crítica:** Todos los valores monetarios usan `font-variant-numeric: tabular-nums; font-feature-settings: "tnum"` y se alinean a la derecha. Nunca a la izquierda.

---

## Espaciado

**Unidad base: 4px**

| Nombre | Valor | Uso |
|--------|-------|-----|
| xs | 4px | Gap entre elementos inline |
| sm | 8px | Padding interno de badges/chips |
| md | 12px | Padding de inputs y botones |
| lg | 16px | Padding de cards |
| xl | 24px | Separación entre secciones |
| 2xl | 32px | Separación mayor |
| 3xl | 48px | Separación entre áreas |

---

## Profundidad (Depth Strategy)

**Solo bordes — sin sombras en elementos normales.**

Razón: este es software local de escritorio, no una app en la nube. Los bordes dan estructura sin afectar rendimiento y se ven bien tanto en monitores de oficina como laptops económicos.

Excepción: modales y dropdowns pueden tener `box-shadow: 0 8px 32px rgba(0,0,0,0.4)` en oscuro / `rgba(0,0,0,0.12)` en claro.

---

## Border Radius

| Elemento | Radio |
|----------|-------|
| Inputs, botones | 6px |
| Cards, paneles | 8px |
| Modales | 12px |
| Badges/pills | 999px |

---

## Navegación

**Sidebar izquierdo angosto** con el mismo fondo que el canvas (no diferente), separado por un borde derecho sutil.

Estructura:
```
[Logo / Marca]
─────────────
Punto de Venta   ← default para cajeros
Caja
Clientes
Inventario       ← solo admin
Reportes         ← solo admin
Configuración    ← solo admin
─────────────
[Avatar] [Nombre] [Rol]
[Toggle claro/oscuro]
```

**Estado activo:** color accent + fondo accent-dim + borde izquierdo 2px accent.

**Regla de roles:**
- Cajero ve: POS, Caja
- Admin ve: todo

---

## Layout del POS (Pantalla Principal)

División **60% / 40%** horizontal:

**Panel izquierdo (búsqueda de productos):**
- Input de búsqueda grande arriba
- Tabs de categorías filtro
- Grid de productos 3 columnas
- Card de producto: nombre + precio (accent, monospace) + badge de stock

**Badge de stock:**
- Verde: > 10 unidades
- Amarillo (warning): ≤ 10
- Rojo (danger): ≤ 3

**Panel derecho (carrito):**
- Header: número de venta + selector de cliente
- Lista de ítems: producto | controles qty | precio
- Subtotal → Descuento → IVA → **Total** (grande, accent)
- Selector método de pago: 4 botones (Efectivo / Transferencia / Tarjeta / Crédito)
- Efectivo: muestra campo "Recibido" + "Cambio"
- Botón **"Cobrar $XX.XXX"** — full width, accent, el elemento más prominente de la pantalla

---

## Formato de Moneda

**COP — Pesos colombianos, sin decimales**

| Correcto | Incorrecto |
|----------|-----------|
| $1.500 | $1,500 |
| $45.000 | $45.000,00 |
| $1.245.000 | $1245000 |

Función de formato:
```typescript
const formatCOP = (amount: number): string => {
  return '$' + amount.toLocaleString('es-CO', { maximumFractionDigits: 0 });
};
```

---

## Estados de Componentes

Todo elemento interactivo debe tener:
- `default` — estado base
- `hover` — cambio sutil de fondo (surface +1 nivel)
- `active/pressed` — más oscuro
- `focus` — outline 2px accent
- `disabled` — opacity 0.4, cursor not-allowed

Todo elemento de datos debe tener:
- `loading` — skeleton o spinner
- `empty` — ilustración/ícono + texto descriptivo
- `error` — mensaje en danger color

---

## Iconografía

**Set:** Lucide Icons (inline SVG, 16px o 20px según contexto)

Regla: los íconos clarifican, no decoran. Si quitar el ícono no pierde significado, quitarlo.

Íconos con fondo contenedor (20x20px, border-radius 6px, surface-3 de fondo) para íconos standalone en dashboards.

---

## Animaciones

| Tipo | Duración | Easing |
|------|----------|--------|
| Hover states | 100ms | ease |
| Cambio de pantalla | 150ms | ease-out |
| Modales | 200ms | ease-out |
| Toast notifications | 250ms | ease-out |

Sin bounce, sin spring en contexto profesional.

---

## Toasts / Notificaciones

- Aparecen en esquina inferior derecha
- Auto-dismiss a los 3 segundos
- Tipos: success (verde), error (rojo), info (neutral)
- Máximo 3 simultáneos

---

## Componentes Clave

### Botón Primario
```
background: accent
color: #000 (negro sobre dorado)
padding: 10px 16px
border-radius: 6px
font-weight: 600
font-size: 14px
```

### Botón Secundario
```
background: transparent
border: 1px solid border-emphasis
color: text-primary
padding: 10px 16px
border-radius: 6px
```

### Input
```
background: surface-3 (inset, más oscuro que el entorno)
border: 1px solid border
border-radius: 6px
padding: 10px 12px
color: text-primary
focus: border-color accent, outline none
```

### Card
```
background: surface-1
border: 1px solid border
border-radius: 8px
padding: 16px
```
