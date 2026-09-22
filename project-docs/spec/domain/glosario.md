# Glosario

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** vocabulario del dominio
> **Actualizado:** 2026-09-21

Términos colombianos de tienda y su equivalente en el código. Sirve para que una US y una tabla hablen el mismo idioma, y para el banco de frases del asistente local.

| Término del tendero | Qué significa | Dónde vive en el código |
|---|---|---|
| **Fiar / fiado** | Vender a crédito, que el cliente pague después | `sale_payments.method = 'credit'` |
| **Cartera** | El total que le deben al negocio | Suma de saldos con `balance > 0` |
| **Abono** | Pago parcial contra la deuda, no contra una factura | `customer_payments` |
| **Cuadrar / cuadre** | Contar el efectivo al cerrar y compararlo con lo esperado | `registers.expected_close` vs `actual_close` |
| **Arqueo** | Lo mismo que el cuadre, término más formal | Cierre de caja |
| **Base** | El efectivo con el que arranca el turno | `registers.opening_amount` |
| **Faltante / sobrante** | Diferencia negativa / positiva del cuadre | `registers.difference` |
| **Quincena** | Los días de pago, 15 y 30. **Es el ciclo de cobro del fiado.** | No modelado todavía — roadmap `R-03` |
| **Tirilla** | El recibo impreso | `TicketPreview`, `CloseTicketPreview` |
| **Nequi / Daviplata** | Billeteras digitales colombianas, muy usadas | Hoy caen en `method = 'transfer'` |
| **Mayorista** | Precio especial por volumen o por tipo de cliente | `products.wholesale_price` |
| **Lo que más deja** | El producto de mayor margen, distinto del más vendido | Ver [rentabilidad.md](rentabilidad.md) |
| **Plata** | Dinero. Se usa mucho más que "dinero" en habla real | — |
| **Se está acabando** | Producto bajo el mínimo de stock | `stock < min_stock` |

## Nota para textos de cara al usuario

El producto le habla al tendero como le habla su proveedor, no como le habla un banco. "Plata" antes que "dinero", "cuánto te deben" antes que "cartera vigente". El vocabulario formal se reserva para documentos fiscales, donde sí se espera.
