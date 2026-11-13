# Manual de usuario — POS

Sistema de punto de venta para tiendas, minimercados y distribuidoras. Funciona **sin internet**, en su propio computador, y **lleva el fiado** (las ventas a crédito de sus clientes) sin cuaderno.

> Actualizado: 13/11/2026.

---

## Contenido

1. [Antes de empezar](#1-antes-de-empezar)
2. [Encender y apagar el sistema](#2-encender-y-apagar-el-sistema)
3. [Primera vez](#3-primera-vez)
4. [Usuarios: dueño y cajero](#4-usuarios-dueño-y-cajero)
5. [Productos e inventario](#5-productos-e-inventario)
6. [Vender](#6-vender)
7. [Fiado y abonos](#7-fiado-y-abonos)
8. [Caja: abrir, movimientos y cierre](#8-caja-abrir-movimientos-y-cierre)
9. [Anular una venta](#9-anular-una-venta)
10. [Lo que ve el dueño](#10-lo-que-ve-el-dueño)
11. [Impresión de tickets](#11-impresión-de-tickets)
12. [Copias de seguridad](#12-copias-de-seguridad)
13. [Usar el sistema desde el celular](#13-usar-el-sistema-desde-el-celular)
14. [Solución de problemas](#14-solución-de-problemas)
15. [Preguntas frecuentes](#15-preguntas-frecuentes)

---

## 1. Antes de empezar

- Computador con Windows 10 u 11.
- Navegador: Chrome, Edge o Firefox.
- No necesita internet ni pagar mensualidad.
- Toda su información queda en un solo archivo, `data.db`, en la misma carpeta que `pos.exe`. Ese archivo es suyo.

---

## 2. Encender y apagar el sistema

**Encender.** Si el instalador dejó activado el inicio automático, el sistema arranca solo cuando el computador entra a Windows. Si no, haga doble clic en `pos.exe`. Se abre el navegador en `http://localhost:8080`; si no abre solo, escriba esa dirección.

**La ventana negra.** Al encender aparece una ventana negra (la consola). **Minimícela, no la cierre:** si la cierra, el sistema se apaga. Si ve en ella un aviso sobre la zona horaria, avise a soporte: el computador debe estar en la hora de Colombia para que "hoy" y el cierre de caja cuadren.

**Apagar.** Cierre el navegador y luego la ventana negra.

---

## 3. Primera vez

La primera vez que abre el sistema aparece un asistente de dos pasos:

1. **Su usuario:** nombre completo, nombre de usuario y contraseña (mínimo 4 caracteres). Este usuario es el **administrador** (el dueño).
2. **Su negocio:** nombre del negocio y NIT.

Después entre a **Configuración** y complete lo demás: dirección, logo, encabezado del ticket, impresora y carpeta de copias de seguridad.

---

## 4. Usuarios: dueño y cajero

Hay dos tipos de usuario:

| Rol | Qué puede hacer |
|---|---|
| **Administrador** (dueño) | Todo: dashboard, clientes, cartera, inventario, compras, reportes, usuarios, auditoría y configuración. Es el único que crea clientes y productos, recibe abonos, cambia precios en la venta, anula ventas y registra gastos. |
| **Cajero** | En el menú solo tiene **Punto de Venta** y **Caja**: vender (también fiar a un cliente ya creado), abrir y cerrar caja, registrar entradas y salidas de efectivo. |

**Qué ve el cajero y qué no.** No ve el dashboard, la ganancia, los reportes, la cartera ni el inventario. Sí ve: lo que debe cada cliente al momento de fiarle (**Debe: ...**), las cajas con sus movimientos (también las anteriores, en **Ver histórico**), el efectivo esperado al cerrar y el ticket de cierre con las ventas por método de pago.

Cree un usuario para cada persona en **Usuarios** (menú lateral). No comparta el usuario del dueño. Si alguien deja de trabajar, desactívelo: su historial se conserva.

Si se equivoca 10 veces seguidas con la contraseña, ese equipo queda bloqueado 5 minutos. Si olvidó la contraseña, soporte la restablece con la herramienta `reset-password.exe`.

---

## 5. Productos e inventario

Todo está en **Inventario** (menú lateral).

### Crear un producto

Use el botón de nuevo producto. Campos principales:

- **Nombre** y, si tiene, **código de barras** (no se puede repetir).
- **Tipo:** producto (lleva inventario) o servicio (no lleva inventario).
- **Unidad:** Unidad, Kilogramo, Libra o Litro.
- **Precio de venta**, **costo** y, si vende al por mayor, **precio mayorista**. Todo en pesos enteros.
- **IVA:** 0 %, 5 % o 19 %. **El IVA se suma encima del precio de venta.** Si usted no cobra IVA, o el precio que escribe ya lo incluye, deje **0 %**; si pone 19 % a un precio que ya lo trae, le cobra 19 % de más al cliente.
- **Stock** y **stock mínimo:** cuando el stock llega al mínimo o baja de él, el producto aparece marcado como bajo stock mínimo.

> **Ponga el costo.** Sin costo, el sistema no cuenta ese producto en la ganancia (para no mostrarle una ganancia falsa) y le avisa cuántos productos no tienen costo.

### Cargar muchos productos de una vez

- **Desde Excel:** Inventario › Productos › **Importar productos**. Descargue la plantilla, llénela en Excel y **guárdela como CSV** (un archivo `.xlsx` no se acepta). Súbala, revise la vista previa y confirme. Si una fila tiene error, no se importa nada hasta corregirlo.
- **Catálogo base de tienda:** en la misma ventana, **Cargar catálogo base de tienda** carga 158 productos comunes de tienda de barrio en 12 categorías. **Todos los precios del catálogo base son de $1.000 de referencia. Usted debe ajustarlos antes de vender.**

### Entradas de mercancía y compras

- **Entrada de inventario:** suma unidades a un producto (por ejemplo, cuando llega un pedido).
- **Compras** (menú lateral): registra la compra a un proveedor y suma el inventario de todos sus productos. Los proveedores se crean en la misma sección.

---

## 6. Vender

Primero debe haber una **caja abierta** (sección 8): sin caja abierta el sistema no deja cobrar. Luego entre a **Punto de Venta**.

1. Busque el producto por nombre o escanee su código de barras. Con el lector, cada lectura agrega el producto al carrito.
2. Ajuste cantidades. Si el producto tiene precio mayorista, puede escoger ese precio en la línea.
3. **Descuentos:** cualquier usuario puede poner un **Descuento** en una línea. Si el dueño lo activó en Configuración, también se puede descontar sobre el total. El dueño, además, puede cambiar el precio de un producto en la venta. En la auditoría quedan el descuento sobre el total y los cambios de precio; **el descuento por línea no queda registrado**, así que revise el total de las ventas si le preocupa.
4. Escoja cómo le pagan: **Efectivo**, **Transferencia** (Nequi, Daviplata o banco), **Tarjeta** o **Credito** (fiado).
   - Con efectivo, escriba lo **Recibido** y el sistema muestra el **Cambio**.
   - Si le pagan una parte en efectivo y otra por Nequi, use **+ Agregar otro método de pago**.
5. Presione **Cobrar**. El ticket se imprime según la configuración de impresión.

**Atender a otro cliente sin perder el carrito:** use **Dejar en espera** y retómelo después.

**Si la sesión se cierra** mientras tenía productos en el carrito, al volver a entrar el carrito sigue ahí (hasta 12 horas).

---

## 7. Fiado y abonos

El fiado es la razón de ser de este sistema. El saldo de cada cliente se calcula siempre a partir de sus ventas y sus abonos, así que **todas las pantallas muestran el mismo número**.

### Crear un cliente

El dueño, en **Clientes**, crea el cliente con nombre y celular; el celular sirve para cobrarle por WhatsApp. Puede ponerle un **tope de crédito** opcional. El cajero no puede crear clientes: solo fiarle a uno ya creado.

### Pasar las deudas del cuaderno (deuda anterior)

Si el cliente ya le debía antes de usar el sistema, regístrelo como **deuda anterior**: al crear el cliente (campos *Deuda anterior (cuaderno)* y *Fecha de la deuda*) o después, desde el detalle del cliente con **Registrar deuda anterior**. Si se equivocó, use **Corregir**. Solo el dueño puede hacerlo. La deuda anterior no cuenta como venta del día.

### Fiar en una venta

En el cobro escoja **Credito** y seleccione el cliente. El sistema le muestra:

- Cuánto debe y hace cuántos días (por ejemplo **Debe: $450.000 (26 días)**).
- Su última visita y **Lo de siempre** (lo que más compra). Con **Agregar lo de siempre** lo pone en el carrito.

Si con esta venta el cliente pasa su tope, o tiene deuda de más de 30 días, el sistema pregunta **¿Fiar de todas formas?**. Usted decide: el sistema avisa, pero **nunca le impide fiar**.

### Recibir un abono

Los abonos los registra el **dueño**: desde **Clientes** (botón **Abonar**), desde el detalle del cliente (**Registrar abono**) o desde el dashboard (**Pagar**). En esta versión el cajero no tiene en su menú una pantalla para recibir abonos.

1. Escoja un monto rápido (**Total**, **Mitad**, $50.000, $20.000) o escriba el valor. No puede ser mayor que la deuda.
2. Escoja cómo pagó: **Efectivo**, **Transferencia** (Nequi, Daviplata o banco) o **Tarjeta**. Esto es importante: solo los abonos en efectivo entran al conteo de la caja.
3. Revise el nuevo saldo y confirme.

Si registra el abono con la caja abierta, un abono en efectivo entra al conteo de esa caja. Si no hay caja abierta, el abono baja la deuda pero no queda en ningún cierre.

### Ver quién debe

- **Clientes:** cada cliente con su saldo y los filtros **Hasta 14 días**, **15–30 días** y **Más de 30 días**.
- **Cartera** (solo el dueño): todos los que deben, del mayor al menor.
- Si al cliente le queda plata a favor (por ejemplo, se anuló una venta fiada que ya había pagado), aparece **Saldo a favor** en verde.

### Cobrar por WhatsApp

En el detalle del cliente, **Cobrar** abre su WhatsApp con el mensaje listo: *"Hola Ana, le escribimos de Tienda Don Carlos. Le recordamos que tiene un saldo pendiente de $45.000. Gracias."* Usted lo revisa y lo envía. El sistema no envía nada solo y no tiene costo.

---

## 8. Caja: abrir, movimientos y cierre

Entre a **Caja**.

**Abrir caja.** Escriba la base (el efectivo con que empieza) y presione **Abrir Caja**. Solo puede haber una caja abierta; si tiene varios equipos en la misma WiFi, todos venden a esa misma caja.

**Entradas y salidas.** Con **Registrar Movimiento** anote dinero que entra o sale y no es una venta: **Entrada** (por ejemplo, el dueño trae sencillo) o **Salida** (por ejemplo, pagar un domicilio). Escriba siempre una nota.

**Gastos del negocio.** El dueño los registra en **Reportes › Gastos del negocio › Registrar gasto**, con su categoría: servicios, arriendo, nómina, proveedor u otro. Si la caja está abierta, el gasto se descuenta del efectivo esperado.

**Cerrar caja.**

1. Presione **Cerrar Caja**.
2. Cuente billetes y monedas e ingréselos por denominación.
3. El sistema compara lo contado con lo esperado:

   ```
   Esperado = Base + Ventas en efectivo + Abonos en efectivo + Entradas − Salidas
   ```

4. Si sobra dinero es **sobrante** y puede cerrar.
5. **Si falta dinero, el sistema no deja cerrar** y muestra *"No se puede cerrar la caja con faltante"*. Haga esto:
   - Vuelva a contar: casi siempre es un billete mal contado o una venta por Nequi que se cobró como efectivo.
   - Si de verdad falta plata, llame al dueño. Para cerrar, registre una **Salida** por la diferencia con la nota *"Faltante de cierre"*: así el faltante queda escrito en los movimientos de la caja y en la auditoría, y la caja cierra.
6. Confirme. Puede imprimir el ticket de cierre y, si el dueño configuró su correo, le llega el resumen por e-mail.

Las ventas y abonos por transferencia, tarjeta o fiado **no** entran al conteo, porque esa plata no está en el cajón.

---

## 9. Anular una venta

Solo el **administrador** puede anular. En el dashboard, en **Ventas recientes**, presione **Anular** en la venta y confirme.

Al anular:

- El inventario de los productos vuelve.
- Si era fiada, la deuda del cliente baja.
- Si era en efectivo y su caja sigue abierta, el esperado de la caja baja. Si esa caja ya se cerró, su cierre no cambia: devuelva la plata al cliente y anote una **Salida** en la caja abierta.
- La venta queda marcada como anulada en el historial y en la auditoría. No se borra.

Solo se pueden anular las ventas que aparecen en la lista de ventas recientes. Para una venta más antigua, llame a soporte.

---

## 10. Lo que ve el dueño

Solo con el usuario administrador.

- **Dashboard:** lo vendido hoy, número de ventas, efectivo en caja, fiado nuevo y fiado cobrado hoy, el producto más vendido, la ganancia bruta y neta, los clientes con deuda más vieja (con botón **Pagar**) y las ventas recientes.
- **Reportes:** ventas por día, semana, mes o el rango que escoja (con exportación a CSV, que abre en Excel), productos más vendidos, cartera, gastos e historial de cajas con sus diferencias.
- **Lo que más vende no siempre es lo que más le deja.** Además del más vendido, el dashboard le muestra los productos con mayor margen (el porcentaje que le queda de cada venta).
- **Asistente:** escriba preguntas como *"cuánto vendí hoy"*, *"quién me debe"* o *"qué se está acabando"*. Responde sin internet. Si pregunta por Nequi, le responde con el total de todas las transferencias (Nequi, Daviplata y bancos juntos).
- **Historial de cajas:** en **Reportes** (y en **Caja › Ver histórico**, que también ve el cajero) están las cajas anteriores con sus movimientos y diferencias. Un faltante resuelto con una **Salida** aparece como ese movimiento (*"Faltante de cierre"*), no como diferencia.
- **Auditoría:** quién anuló ventas, hizo descuentos sobre el total, cambió precios, sacó efectivo, importó productos o corrigió deudas anteriores, y cuándo.

---

## 11. Impresión de tickets

Por defecto el ticket se imprime con el diálogo del navegador: elija la impresora y confirme. Sirve cualquier impresora que Windows reconozca, también "Microsoft Print to PDF".

**Impresora térmica (58 u 80 mm).** Si sale cortado, en Windows vaya a *Configuración › Bluetooth y dispositivos › Impresoras y escáneres*, abra las *Preferencias de impresión* y escoja el tamaño de papel de su impresora.

**Impresión directa (sin diálogo).** En **Configuración › Formato de tickets**, **Método de impresión**:

- **Navegador:** el diálogo de siempre.
- **ESC/POS red:** impresora con cable de red o WiFi. Escriba su **IP** (por ejemplo `192.168.1.50`) y el **puerto** (normalmente `9100`).
- **ESC/POS Windows:** impresora instalada en Windows, por ejemplo por USB. Escriba el **nombre exacto** que aparece en *Impresoras y escáneres*.

Escoja el **Ancho del papel**, presione **Guardar cambios** y luego **Imprimir prueba** (este botón solo aparece con los métodos ESC/POS): debe salir `PRUEBA DE IMPRESION` con tildes y ñ bien escritas. Si la impresora falla, la venta igual queda registrada; puede reimprimir con **Imprimir** o volver al método **Navegador**.

---

## 12. Copias de seguridad

**El sistema hace una copia automática cada día** y guarda las últimas 30. Cuando una actualización cambia la base de datos, guarda además una copia especial (`data.db.pre-migration-*`) que no se borra sola.

- **Dónde quedan:** junto a `pos.exe`, o en la carpeta que escoja en **Configuración › Copias de seguridad** (por ejemplo una memoria USB o `D:\Respaldos`). Si esa carpeta no está disponible, la copia se guarda junto a `pos.exe` y el dashboard muestra un aviso.
- **Muy importante:** una copia en el mismo computador no lo protege si el disco se daña o le roban el equipo. **Cada semana copie el archivo `data.db.backup-*` más reciente a una memoria USB o a Google Drive.**

**Restaurar una copia** (mejor con ayuda de soporte):

1. Apague el sistema (cierre la ventana negra).
2. Cree una carpeta, por ejemplo `C:\POS\anterior`, y **mueva** a ella los tres archivos `data.db`, `data.db-wal` y `data.db-shm` (los dos últimos pueden no existir). **No deje `data.db-wal` en la carpeta:** si queda, el sistema le vuelve a poner los datos viejos encima de la copia.
3. Copie el archivo de respaldo (`data.db.backup-...`) en la carpeta de `pos.exe` y cámbiele el nombre a `data.db`.
4. Encienda el sistema y revise que estén las ventas de ese día.

> **Para guardar una copia fuera del computador, copie siempre un archivo `data.db.backup-*`, nunca `data.db`.** El archivo `data.db` solo no tiene las ventas más recientes (estas quedan en `data.db-wal`), así que una copia suya puede salir incompleta o vacía.

---

## 13. Usar el sistema desde el celular

Un celular o tablet conectado a la **misma WiFi** del negocio puede usar el sistema:

1. En **Configuración** active **Permitir acceso desde otros dispositivos de mi red WiFi** y guarde.
2. Apague y vuelva a encender el sistema.
3. Configuración muestra la dirección (por ejemplo `http://192.168.1.15:8080`). Ábrala en el celular.
4. Si no carga, permita `pos.exe` en el Firewall de Windows (redes privadas).

Desde fuera del negocio (otra red o datos móviles) **no** se puede entrar.

---

## 14. Solución de problemas

| Problema | Qué hacer |
|---|---|
| El navegador no abre solo | Escriba `http://localhost:8080`. |
| "No se puede acceder a este sitio" | El sistema está apagado: abra `pos.exe` (o reinicie el computador si tiene inicio automático). |
| La sesión se cerró | Vuelva a entrar. El carrito se conserva. |
| No deja cobrar: "no hay caja abierta" | Abra la caja en **Caja**. |
| "No se puede cerrar la caja con faltante" | Vuelva a contar; si de verdad falta, siga la sección 8, paso 5. |
| "Debe haber una caja abierta para registrar un abono" | Abra la caja antes de registrar el abono. |
| El dashboard avisa que la copia de seguridad falló | Revise que la memoria USB esté conectada, o cambie la carpeta en Configuración › Copias de seguridad. |
| El ticket sale cortado | Ajuste el tamaño de papel (sección 11). |
| La impresora térmica no imprime | Use **Imprimir prueba** en Configuración › Formato de tickets; mientras tanto cambie a **Navegador**. |
| Olvidé la contraseña | Llame a soporte. |
| Las ventas de la noche salen en el día equivocado | El computador no está en la hora de Colombia. Llame a soporte. |

---

## 15. Preguntas frecuentes

**¿Necesito internet?**
No. Solo se usa internet si configura el envío del cierre de caja a su correo, y si no hay conexión el cierre igual se hace.

**¿Se pierde algo si se va la luz?**
Lo que ya se cobró queda guardado. Al volver la luz, encienda el computador (si tiene inicio automático, el sistema arranca solo) y siga.

**¿Puedo usarlo en varios equipos?**
Sí, en la misma WiFi y con una sola caja (sección 13). No funciona entre sucursales ni por internet.

**¿El sistema le cobra intereses al cliente o le bloquea el fiado?**
No. Avisa cuando pasa el tope o tiene deuda vieja; la decisión es suya.

**¿Hace factura electrónica DIAN?**
El sistema imprime tickets de venta. La factura electrónica no está activa en esta versión.

**¿De quién son los datos?**
Suyos. Están en su computador, en el archivo `data.db`.
