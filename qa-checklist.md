# QA Checklist — POS Vertical

> Creado 29 julio 2026. Lista de todo lo que existe hoy en el código (`internal/handler`, `internal/model`, `frontend/app`) para probar antes de vender/instalar a más clientes.
> Marcar `[x]` cuando esté probado con las manos (no solo "el código lo tiene"). Anotar bugs encontrados debajo de cada sección.

---

## 1. Autenticación y usuarios

- [ ] Login con usuario/clave válidos entra al sistema.
- [ ] Login con clave incorrecta muestra error, no entra.
- [ ] Login con usuario inexistente muestra error genérico (no revela si el usuario existe).
- [ ] `/me` devuelve el usuario correcto tras login.
- [ ] Cambio de contraseña propia (`changeMyPassword`) funciona y la nueva clave sirve para el próximo login.
- [ ] Admin crea usuario nuevo (`admin` o `cashier`).
- [ ] Admin edita usuario existente (nombre, rol).
- [ ] Admin desactiva usuario (`deactivateUser`) — el usuario desactivado ya no puede loguear.
- [ ] Listado de usuarios (`listUsers`) muestra todos, incluidos inactivos (verificar si se filtran o no).
- [ ] `reset-password.exe` reinicia clave de un usuario sin necesitar login (caso: tendero se le olvida la clave).
- [ ] Rol `cashier` NO puede acceder a pantallas de admin (usuarios, config fiscal) — verificar bloqueo real, no solo oculto en el menú.

---

## 2. Productos e inventario

- [ ] Crear producto (nombre, precio, costo, stock inicial, categoría, código de barras).
- [ ] Editar producto existente — cambios se reflejan en ventas nuevas, no en ventas ya hechas.
- [ ] Eliminar producto — verificar qué pasa si el producto ya tiene ventas asociadas (no debe romper el historial).
- [ ] Buscar producto por código de barras (`getProductByBarcode`) — con lector físico si hay uno disponible.
- [ ] Listado de productos con filtro/búsqueda por nombre.
- [ ] Entrada de stock (`createStockEntry`) — sumar inventario, verificar que el stock se actualiza.
- [ ] Productos con stock bajo mínimo aparecen en `listLowStockProducts` y en el dashboard.
- [ ] Producto sin costo configurado (`cost = 0`) — verificar que no rompe el cálculo de ganancia, solo lo excluye y lo cuenta en `unconfigured_cost_count`.
- [ ] Categorías: crear, editar, eliminar. Eliminar categoría con productos asociados — verificar comportamiento (¿bloquea o deja huérfanos?).

---

## 3. Ventas (POS)

- [ ] 🤖 Venta completa en efectivo — total correcto (`TestSaleService_Create_AllPaymentMethods`, `TestSaleService_Create_RecalculatesTotalCorrectly`). Descuenta stock: sigue 100% manual, no hay test de repositorio que lo verifique.
- [ ] 🤖 Venta completa fiada (`credit`) — se asocia a un cliente (`TestSaleService_Create_CreditWithCustomer_Succeeds`). "No descuenta caja" sigue manual.
- [ ] Venta con múltiples productos y cantidades decimales (ej. 0.5 kg si aplica a báscula/granel).
- [ ] 🤖 Venta sin cliente seleccionado en pago fiado — debe bloquear (`TestValidateSale_CreditRequiresCustomer`, `TestSaleService_Create_CreditWithoutCustomer_ReturnsError`).
- [ ] 🤖 (parcial) Anular venta (`voidSale`) — `TestVoidSale_WritesAuditLog` cubre que el endpoint corre y escribe auditoría; NO verifica que el stock se revierta ni que una segunda anulación sea rechazada — sigue manual para esos dos puntos.
- [ ] Anular venta con nota de crédito (`voidSaleWithCreditNote`) si aplica a facturación electrónica.
- [ ] Ver detalle de una venta (`getSale`).
- [ ] Imprimir/ver ticket (`getTicket`) — formato correcto, datos del negocio (nombre, NIT) aparecen bien.
- [ ] Listado de ventas (`listSales`) con filtros de fecha.
- [ ] **Caso crítico ya identificado en pre-venta:** apagar el PC a mitad de una venta, reabrir — no se pierde nada, no queda venta a medias en estado inconsistente.
- [ ] 🤖 Venta con producto que no tiene stock suficiente — bloquea (`TestSaleService_Create_ExceedsStock_ReturnsError`).
- [ ] 🤖 Venta con IVA + descuento global simultáneos — número exacto verificado a mano (`TestSaleService_Create_TaxPlusGlobalDiscount`, SPRINT-11 US-050).

---

## 4. Clientes y fiado (cartera)

- [ ] 🤖 Crear cliente (nombre, teléfono, límite de crédito si existe) (`TestCreate_WithNotesAndCreditLimit`, `TestCreate_WithoutCreditLimit`).
- [ ] Editar cliente.
- [ ] Eliminar cliente — verificar qué pasa si tiene ventas fiadas activas (no debería poder borrarse con deuda pendiente, o debe advertir).
- [ ] 🤖 Ver saldo de un cliente (`getCustomerBalance`) — coincide con suma de ventas fiadas menos pagos (`TestBalance_CreditLimitWarningScenario`); caso venta mixta con IVA + porción a crédito (`TestSaleService_Create_MixedPayment_WithTax`, SPRINT-11 US-050).
- [ ] 🤖 Registrar abono/pago de cliente (`createCustomerPayment`) — reduce el saldo correctamente (`TestCreatePayment_OK`, `TestCreatePayment_ExactBalance`).
- [ ] 🤖 Listado de pagos de un cliente (`listCustomerPayments`) — (`TestListPayments_ReturnsAll`, `TestListPayments_Empty`).
- [ ] Detalle de cliente (`getCustomerDetail`) — muestra ventas + pagos combinados, saldo correcto.
- [ ] 🤖 Pago mayor al saldo de deuda — bloquea (`TestCreatePayment_ExceedsBalance`).
- [ ] Cartera envejecida en dashboard (`aging_receivables`) — días de mora calculados correctamente contra la venta fiada más antigua sin pagar.
- [ ] Portafolio de cartera (`getPortfolio` en reports) — total y conteo de clientes con deuda coincide con lo real.
- [ ] 🤖 Tope de crédito del carrito (frontend) — cliente + carrito supera límite bloquea visualmente (`exceedsCreditLimit`, `frontend/lib/saleCalc.test.ts`, SPRINT-11 US-050).

---

## 5. Caja (register)

- [ ] Abrir caja (`openRegister`) con monto inicial.
- [ ] No se puede abrir una segunda caja si ya hay una abierta (verificar).
- [ ] Ver caja abierta (`getOpenRegister`).
- [ ] Registrar movimiento de caja — entrada y salida (`addMovement`, tipo `cash_out` al menos).
- [ ] Cerrar caja (`closeRegister`) — cuadre final correcto (ventas efectivo + movimientos).
- [ ] Listado de movimientos de una caja (`listMovements`).
- [ ] Listado histórico de cajas (`listRegisters`) y detalle de una caja cerrada (`getRegister`).
- [ ] Reporte de cajas (`getRegistersReport`) coincide con los cierres individuales.
- [ ] Efectivo estimado del dashboard (`cash_estimated` = ventas efectivo − cash_out del día) cuadra con la caja real.

---

## 6. Reportes y dashboard

- [ ] 🤖 (parcial) Dashboard resumen del día (`/dashboard/today`): ganancia bruta y ventas de hoy cubiertas (`TestGetDashboardToday_GrossProfit`); fiado nuevo/cobrado y efectivo estimado siguen manuales.
- [ ] Producto más vendido del día — correcto por cantidad, no por monto.
- [ ] 🤖 Ganancia bruta del día (`gross_profit`) — cálculo `(precio − costo) × cantidad` correcto, solo productos con costo configurado (`TestGetDashboardToday_GrossProfit`, incluye caso de producto con `cost=0` excluido y contado en `unconfigured_cost_count`).
- [ ] 🤖 Top 5 productos por margen (`top_by_margin`) — orden mayor a menor margen (`TestGetDashboardToday_GrossProfit`).
- [ ] Top 5 productos con stock bajo (`low_stock`) — ordenado por déficit (min_stock − stock).
- [ ] Dashboard general (`/dashboard`): hoy vs. ayer, caja abierta, cartera, últimas 8 ventas.
- [ ] 🤖 (parcial) Reporte de ventas por rango de fechas (`getSalesReport`) — el desglose de pagos mixtos por método está cubierto (`TestReportRepo_SalesSummary_SplitsMixedPayments`); agrupación día/semana/mes sigue manual.
- [ ] Top productos por rango de fechas (`getTopProducts`), no solo del día.
- [ ] 🤖 (parcial) Reportes con rango de fechas vacío (sin ventas) — cubierto para `/dashboard/today` (`TestGetDashboardToday_NoSales`); el resto de reportes (`getSalesReport`, `getTopProducts`, `getPortfolio`) siguen manuales.

---

## 7. Facturación electrónica (Factus / DIAN)

- [ ] Config fiscal: ver (`getFiscalConfig`) y actualizar (`updateFiscalConfig`) — NIT, resolución, datos del negocio.
- [ ] Generar factura electrónica de una venta (`getSaleFactura`) — se conecta bien con Factus (o el proveedor configurado).
- [ ] Descargar/ver PDF de factura (`getSaleFacturaPDF`) — formato válido, datos correctos.
- [ ] Probar con credenciales de Factus inválidas o sin internet — el sistema no debe crashear, debe dar error claro y dejar seguir vendiendo (fiscal no debe bloquear el POS).
- [ ] Nota de crédito por anulación (ligado a `voidSaleWithCreditNote`) llega correctamente al proveedor fiscal.

> Nota: según `pos-validation.md`, facturación DIAN es explícitamente NO bloqueante para el piloto. Probar igual porque ya hay cliente real pagando (pre-sale.md) que puede necesitarlo.

---

## 8. Configuración inicial (setup) y negocio

- [ ] Wizard de instalación en limpio (`getSetup` / `updateSetup`) — nombre del negocio, NIT, moneda.
- [ ] Instalación completa cronometrada desde carpeta limpia siguiendo `INSTALL.md` — meta <30 min (pendiente en `pre-sale.md`).
- [ ] Cambiar configuración del negocio después de ya tener ventas — no rompe datos históricos.

---

## 9. Frontend / pantallas (`frontend/app`)

- [ ] `login` — validaciones de formulario, mensajes de error legibles para el tendero (no error técnico crudo).
- [ ] `pos` (pantalla de venta) — flujo completo usable en un PC de mostrador normal, sin mouse gamer ni pantalla grande.
- [ ] `inventory` — listado, alta, edición de productos usable sin conocimientos técnicos.
- [ ] `customers` + `customers/[id]` — ver cliente, historial, registrar pago desde la UI.
- [ ] `register` (caja) — abrir/cerrar caja desde la UI, sin necesitar Postman/consola.
- [ ] `reports` — filtros de fecha usables, números legibles (formato moneda COP).
- [ ] `portfolio` — vista de cartera/deudores clara para el tendero.
- [ ] `users` — solo visible/accesible para admin.
- [ ] `settings` — cambios se guardan y persisten tras cerrar/reabrir.
- [ ] `change-password` — flujo usable sin soporte.
- [ ] `setup` — primera vez que se abre la app, sin datos, no crashea.
- [ ] Responsive/usable en Android gama media si el negocio necesita acceso desde celular (mencionado como pendiente en `pos-validation.md` para pilotos multi-dispositivo).

---

## 10. Persistencia, backups y resiliencia

- [ ] Cerrar la app y reabrir — todos los datos siguen (ventas, clientes, caja, productos).
- [ ] Backup automático o manual de `data.db` existe y es restaurable (probar restaurar uno de `data.db.backup-*` en limpio).
- [ ] Apagar el PC de golpe (no cierre normal) a mitad de una operación — reabrir sin corrupción de la base.
- [ ] Migración de base de datos (mencionada en commit `833a922`) corre limpio sobre una base con datos reales, no solo sobre una vacía.

---

## 11. Instalación y actualización (para cuando haya más de 1 cliente)

- [ ] Instalar `pos.exe` en un PC limpio sin Go instalado — no debe requerir toolchain, solo el ejecutable.
- [ ] `reset-password.exe` corre igual, standalone, sin Go instalado.
- [ ] Definir y probar mecanismo de actualización (pendiente de diseño — ver conversación sobre updates/licencia). Por ahora: documentar proceso manual de reemplazo de `pos.exe` sin perder `data.db`.

---

## Cómo usar este checklist

1. Ejecutar cada ítem con datos reales de prueba (no confiar en "el código lo tiene", como ya se acordó para el cliente del minimercado).
2. Cada bug encontrado: anotar aquí mismo debajo de la sección, con pasos para reproducir.
3. Antes de vender/instalar a cliente nuevo: secciones 1-6, 9 y 10 deben estar en verde. Secciones 7 y 11 solo si el cliente las necesita.
</content>
