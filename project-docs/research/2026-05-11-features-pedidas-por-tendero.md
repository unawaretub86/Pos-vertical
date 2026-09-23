# POS Vertical — Ideas para Diferenciarse

> **Tipo:** research · **Vigente:** NO · **Fecha:** 2026-05-11
> Investigación de mercado. **Sus afirmaciones de "esto no existe" se verifican con `grep` antes de usarse.** El doc de investigación más reciente y completo es `better-finds-pos.md`. Lo que viene está en [roadmap.md](roadmap.md).

> Generado: 2026-05-11

---

## 1. El Fiado como producto estrella (nadie lo hace bien)

**Fiado inteligente con memoria social:**
- Notas por cliente: "Le fío máx $50K", "Paga viernes", "No fiar más — debe desde enero"
- Historial de comportamiento: promedio días para pagar, veces que no pagó
- Alerta visual en POS cuando el cliente tiene deuda antes de hacer venta nueva
- Vista "Cartera del día" — quién debe pagar esta semana, ordenado por urgencia

**WhatsApp con un clic:**
- Botón en cada cliente deudor → genera mensaje prellenado con deuda exacta
- No requiere API — `wa.me/57NUMERO?text=...` funciona offline, abre WhatsApp del PC
- Ningún POS colombiano barato lo hace. Esto solo ya vende el sistema.

---

## 2. Precios que reflejan la realidad del negocio

Los tenderos tienen 3-5 precios mentales, no uno:

| Tipo | Descripción |
|------|-------------|
| Menudeo | Precio normal |
| Al por mayor | Compra 6+, precio diferente |
| Cliente especial | Juan siempre paga X |
| "A como está" | Producto próximo a vencer |
| Costo + margen | Para no vender a pérdida |

**Feature:** al agregar al carrito, dropdown con precios rápidos. No un formulario — 1 clic.

---

## 3. Dashboard que habla el idioma del tendero

No gráficas de BI — números que el dueño entiende al despertarse:

```
HOY
Vendido:           $340.000
Fiado nuevo:        $45.000
Cobrado de fiado:   $80.000
Efectivo en caja:  $215.000
Lo más vendido:    Arroz Diana x25
```

Argumento de venta: **"El cuaderno digital"**. No "dashboard analítico".

---

## 4. Modo "Vendedor rápido" para horas pico

- Vista minimalista: buscar producto → cantidad → cobrar
- Teclado numérico grande (funciona con lector de código de barras)
- Sin mouse si quiere — todo navegable con Enter/Tab/números
- Objetivo: tiempo por transacción < 15 segundos

---

## 5. Backup que el dueño entiende

"Sus datos están seguros" es un mensaje, no una feature. Hacerlo visible:

- Botón "Guardar en USB" — copia la DB a cualquier USB conectado, con fecha en nombre
- Contador visible: "Último backup: hace 3 días" en rojo si pasa 7 días
- Google Drive opcional con un clic (OAuth, sin configuración técnica)

---

## 6. Onboarding que funciona sin manual

Primera vez que abren el .exe:

1. Logo del negocio → arrastra imagen
2. NIT / nombre / dirección → 3 campos
3. Impresora → detecta automático o "por ahora no"
4. Primer producto → guiado con ejemplos
5. Primera venta de prueba con dinero ficticio

Objetivo: alguien no técnico, funcionando en 10 minutos.

---

## 7. Reportes de supervivencia del negocio

No reportes contables — reportes que evitan problemas reales:

- **"Productos que no han rotado en 30 días"** → capital muerto
- **"Margen real por producto"** → cuánto gana en cada cosa
- **"Clientes que deben hace más de X días"** → cartera vencida
- **"Diferencia entre inventario esperado y real"** → detecta hurto interno

---

## Argumento de venta único

> *"El único POS que maneja el fiado como usted lo hace en su cabeza, funciona sin internet, y cabe en una USB."*

Los competidores cloud venden tecnología. Este vende **control del negocio**.

---

## Prioridad de implementación

| # | Feature | Esfuerzo | Impacto |
|---|---------|----------|---------|
| 1 | WhatsApp botón (link directo, sin API) | 1 día | 🔴 Enorme |
| 2 | Notas + comportamiento por cliente deudor | 2 días | 🔴 Alto |
| 3 | Precios múltiples con dropdown rápido | 3 días | 🔴 Alto |
| 4 | Dashboard "cuaderno digital" | 2 días | 🟡 Medio |
| 5 | Onboarding guiado | 3 días | 🟡 Medio |
