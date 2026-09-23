# Dominio — Fiado (crédito a clientes)

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** saldo del cliente, abonos, cartera
> **Actualizado:** 2026-09-23 · Verificado contra el código en esa fecha (US-059: métricas en `internal/repository/metrics.go`).

El fiado es **el diferenciador del producto**. Los competidores lo tratan como un campo contable; aquí es una relación con historia. Cualquier cambio en estas reglas cambia la propuesta de valor, no solo el código.

---

## Modelo

Una venta se paga con una o varias líneas en `sale_payments`, cada una con un método. Cuando el método es `credit`, esa línea **es** la deuda. No hay tabla de deudas: la deuda es la suma de lo fiado menos la suma de lo abonado.

```
métodos válidos (sale_payments.method):  cash · transfer · card · credit
                 (sales.payment_method): cash · transfer · card · credit · mixed
```

`sales.payment_method = 'mixed'` es un rótulo de la venta; el desglose real siempre vive en `sale_payments`.

## Regla: saldo del cliente

El saldo **nunca se almacena**. Se calcula al vuelo en cada consulta.

```
saldo(cliente) = SUM(sale_payments.amount donde method='credit'
                     y la venta está 'completed')
               − SUM(customer_payments.amount)
```

Fuente: `internal/repository/customer.go:24-40`.

Consecuencias que hay que respetar:

- **Una venta anulada deja de contar automáticamente**, porque el filtro es `status='completed'`. No hay que "revertir" la deuda a mano.
- **No existe un saldo desactualizado.** Cualquier pantalla que muestre saldo lo lee de la misma query. Si alguna vez aparece un saldo distinto en dos pantallas, es un bug de query, no de sincronización.
- El cliente con saldo `0` o negativo **no está en cartera**. La cartera filtra `balance > 0`.
- Orden de la cartera: saldo descendente y, en empate, nombre ascendente. Fuente: `MetricsRepository.Debtors` (`internal/repository/metrics.go:263`, orden en `:284`), misma fórmula de saldo que `customer.go:24-40`.

## Regla: antigüedad de la deuda

```
oldest_debt_at = MIN(created_at) de las ventas con línea de crédito
                 del cliente, entre las 'completed'
días_mora      = hoy − oldest_debt_at
```

Es la fecha de la deuda **más vieja sin distinguir abonos parciales**: un cliente que abona sin saldar completo conserva su fecha original. Es deliberado — mide cuánto lleva ese cliente debiendo algo, no cuánto lleva debiendo ese peso.

## Regla: límite de crédito

`customers.credit_limit` es **opcional** (`NULL` = sin límite definido) y hoy es **informativo**: advierte, no bloquea la venta.

Fuente: `internal/service/customer.go:39,54`. Escenario cubierto en `internal/service/customer_test.go:213`.

Si alguna vez se decide que bloquee, es un cambio de producto que debe aprobar Andrey: un tendero que no puede fiar por un tope que él mismo puso mal, deja de usar el sistema.

## Regla: abonos

Un abono es una fila en `customer_payments`. No se asocia a una venta específica — se abona al saldo total del cliente, no a una factura. Es como funciona el cuadernillo que el sistema reemplaza, y cambiarlo a imputación por factura sería un rediseño, no un ajuste.

---

## Qué NO existe todavía

- **Scoring de riesgo del cliente** — roadmap `R-17`. Es el diferenciador más fuerte sin construir: con el historial de abonos que ya se guarda se puede calcular score, límite sugerido y alerta temprana de mora. Ningún competidor local guarda ni analiza esto.
- **Bloqueo de venta por exceder el límite** — decisión de producto pendiente.
- **Intereses o recargos por mora** — nunca se ha considerado; el tendero colombiano no los cobra.
