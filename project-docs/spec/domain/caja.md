# Dominio — Caja (apertura, movimientos, cierre)

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** arqueo, cuadre, diferencia de caja
> **Actualizado:** 2026-09-21 · Verificado contra el código en esa fecha.

La caja es donde el sistema se gana o pierde la confianza del cajero. Un cuadre que da mal sin explicación hace que la tienda vuelva al cuadernillo.

---

## Estados

Una caja (`registers`) está `open` o `closed`. Solo puede haber una abierta a la vez.

## Movimientos

```
tipos válidos (register_movements.type): sale · cash_in · cash_out · opening
```

- `opening` — el monto con el que arranca el turno (base).
- `sale` — venta registrada contra esa caja.
- `cash_in` — entra plata que no es una venta. Ej.: el dueño le pasa cambio al cajero.
- `cash_out` — sale plata que no es una devolución. Ej.: pagar un domicilio, un gasto.

Los **gastos** (`expenses`) se espejan automáticamente como `cash_out` cuando hay caja abierta, para que el arqueo los recoja sin duplicar lógica de flujo de efectivo. Fuente: `internal/service/expense.go`.

## Regla: efectivo esperado al cierre

```
esperado  = base_apertura + ventas_en_efectivo + cash_in − cash_out
diferencia = contado_real − esperado
```

Fuente: `internal/service/register.go:113` y `:138`.

- `diferencia > 0` → sobrante
- `diferencia < 0` → faltante
- Solo cuenta el **efectivo**. Las ventas por transferencia, tarjeta o fiado no entran al arqueo: esa plata no está en el cajón.

> **Nota histórica:** `PLAN.md:1134` reporta que faltaba `cash_in` en esta fórmula. **Ya está incluido** (`register.go:113`). Es un ejemplo de por qué `PLAN.md` no es fuente de verdad sobre el estado actual.

## Regla: gastos

```
categorías válidas: servicios · arriendo · nomina · proveedor · otro
```

Fuente: `internal/service/expense.go:17-19`. Lista cerrada, validada en el servicio. Agregar una categoría es tocar esa constante **y** revisar los reportes que agrupan por ella.

```
ganancia neta = ganancia bruta − gastos del mismo rango
```

---

## Qué NO existe todavía

- Cierre de turno imprimible (reporte X/Z) más allá de lo cubierto en SPRINT-11.
- Múltiples cajas simultáneas — roadmap `R-18` (sync LAN).
- Arqueo por denominaciones más allá de lo entregado en SPRINT-11 (`frontend/lib/denominations.ts`).
