# Dominio — Rentabilidad (costo, margen, ganancia)

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** ganancia bruta, ganancia neta, margen
> **Actualizado:** 2026-09-21 · Verificado contra el código en esa fecha.

"Cuánto vendí" no es "cuánto gané", y la diferencia entre las dos es lo que hace que un tendero deje de vender a pérdida sin saberlo. `find-pos.md:41` lo identificó como un hueco que los competidores locales no cubren.

---

## Regla: ganancia bruta

```
ganancia bruta = SUM( (sale_items.unit_price − products.cost) * sale_items.quantity )
                 solo para productos con cost > 0
                 solo ventas con status = 'completed'
```

Fuente: `internal/handler/dashboard.go` (query de ganancia bruta).

**El filtro `cost > 0` es la regla crítica.** Un producto sin costo configurado no suma ganancia — se excluye del cálculo en vez de asumir costo cero, porque asumir cero reportaría 100% de margen y sería una mentira peligrosa.

Por eso toda respuesta de ganancia o margen debe acompañarse del **conteo de productos sin costo configurado**, para que el dueño sepa que la cifra real es mayor que la mostrada.

## Regla: ganancia neta

```
ganancia neta = ganancia bruta − gastos del mismo rango
```

Ver [caja.md](caja.md) para las categorías de gasto.

## Regla: margen

```
margen % = (ganancia bruta / ingresos) * 100
           0 si ingresos = 0
```

Por producto, en el top por margen:

```
margen_pct(producto) = ((AVG(unit_price) − cost) / AVG(unit_price)) * 100
```

Se usa `AVG(unit_price)` porque el mismo producto puede haberse vendido a precios distintos (descuento, mayorista) dentro del rango.

## El contraste que vende

**El producto que más se vende casi nunca es el que más deja.** `find-pos.md:91` lo documenta con el caso real: arroz a 11% de margen y 89 unidades contra aceite a 38% y 24 unidades.

Cualquier pantalla o respuesta que muestre "lo más vendido" debería poder mostrar ese contraste. Es la diferencia entre entregar un reporte y entregar un consejo.

## Regla: snapshot de precio

`sale_items.unit_price` es una **copia** del precio al momento de la venta. Cambiar el precio o el costo de un producto **no altera ventas ya registradas**. Cualquier cálculo histórico de margen usa el precio de entonces y el costo **actual** del producto — es una aproximación conocida, no un bug.

---

## Qué NO existe todavía

- Historial de costo por producto. El costo es un valor único y actual, así que el margen histórico usa el costo de hoy contra el precio de entonces.
- Productos sin rotación / capital muerto — US-048, pendiente de SPRINT-09.
- Margen por categoría o por proveedor.
