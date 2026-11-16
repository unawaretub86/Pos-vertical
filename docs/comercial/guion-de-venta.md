# Guion de venta y demo

> Actualizado: 13/11/2026. Lo que se promete aquí está verificado contra el código en `project-docs/features.md`. **Si algo no está en esa lista, no se promete.**

---

## 1. La frase de entrada (10 segundos)

> "Es un sistema para su tienda que **le lleva el fiado sin cuaderno**, funciona **sin internet** y **no se paga cada mes**: se paga una vez y queda suyo."

## 2. Las tres promesas

| Promesa | Cómo se demuestra en la visita |
|---|---|
| **Funciona sin internet** | Desconecte el WiFi delante del cliente y haga una venta. |
| **Lleva el fiado de verdad** | Fíe una venta, muestre el saldo y los días, reciba un abono y mande el recordatorio por WhatsApp. |
| **Sin mensualidad** | Todo queda en su computador, en un archivo que es suyo. |

## 3. Lo que le duele al tendero, y lo que el sistema hace

| El problema | La respuesta del sistema |
|---|---|
| "El cuaderno se pierde, se moja, no se entiende la letra." | Cada cliente con su saldo, sus compras y sus abonos. La deuda vieja del cuaderno se pasa como **deuda anterior**, con su fecha. |
| "No sé quién me debe más ni desde cuándo." | Filtros **Hasta 14 días**, **15–30 días** y **Más de 30 días**, y la cartera ordenada de mayor a menor. |
| "Me da pena cobrar." | **Cobrar** abre WhatsApp con un mensaje amable ya escrito. Usted solo lo envía. |
| "Fío de más sin darme cuenta." | Tope de crédito por cliente y aviso **¿Fiar de todas formas?** cuando lo pasa o la deuda tiene más de 30 días. La decisión sigue siendo suya. |
| "La caja nunca me cuadra." | Cierre con conteo de billetes y monedas: esperado contra contado. Si falta plata, el sistema avisa, pide confirmar y deja el faltante escrito en el ticket y el historial. Las ventas por Nequi, tarjeta y fiado no se mezclan con el efectivo. |
| "No sé cuánto me gano." | Ganancia real del día. Si un producto no tiene costo, no lo inventa: le dice cuántos faltan. Y le muestra que **lo que más vende no es lo que más le deja**. |
| "El cajero ve todo." | El cajero solo tiene **Punto de Venta** y **Caja**: no ve el dashboard, la ganancia, los reportes, la cartera ni el inventario. (Sí ve la deuda del cliente al fiarle y los movimientos de las cajas.) |
| "Si se va la luz, pierdo todo." | Lo cobrado queda guardado, el sistema arranca solo al prender el computador y hace una copia de seguridad diaria. |
| "Cargar los productos es eterno." | Importación desde Excel (guardado como CSV) y un catálogo base de 158 productos de tienda (con precios de referencia que usted ajusta). |
| "Quiero saber cómo voy sin revisar papeles." | Pregúntele al asistente: *"cuánto vendí hoy"*, *"quién me debe"*, *"qué se está acabando"*. Responde sin internet. |

## 4. Demo de 5 minutos (con datos de prueba, antes de instalar)

Prepare antes, con el usuario del dueño: 10 productos con costo; 2 clientes con fiado, uno de ellos con una **deuda anterior** fechada hace más de 30 días (las ventas no se pueden poner con fecha atrás) y al menos una compra previa, para que aparezca **Lo de siempre**; una caja abierta.

1. **Venta en efectivo** (30 s): escanear o buscar 2 productos, **Efectivo**, **Recibido**, ver el **Cambio**, **Cobrar**, sale el ticket.
2. **Venta fiada** (60 s): **Fiado**, escoger el cliente con deuda vieja. Mostrar **Debe: ... (35 días)**, **Lo de siempre** y el aviso **¿Fiar de todas formas?**. Fiar.
3. **Abono** (45 s): en **Clientes**, **Abonar**, monto **Mitad**, método **Transferencia**. Mostrar el nuevo saldo.
4. **Cobro por WhatsApp** (30 s): detalle del cliente, **Cobrar**, mostrar el mensaje listo.
5. **Sin internet** (15 s): desconectar el WiFi y repetir una venta.
6. **El dueño** (60 s): dashboard con lo vendido, el fiado nuevo y cobrado, la ganancia; preguntar al asistente *"quién me debe"*.
7. **Cierre** (30 s): **Cerrar Caja**, contar, ver la diferencia. Cuente un poco de más para mostrar un sobrante; si cuenta de menos, el sistema avisa el faltante y pide confirmar antes de cerrar.

Después pregunte: **"¿Qué le dan los programas que conoce?"** Su respuesta le dice qué espera en soporte y en precio.

## 5. Objeciones

| Le dicen | Usted responde |
|---|---|
| "Ya uso un programa con mensualidad." | "Este se paga una vez. ¿Cuánto paga al mes? Multiplíquelo por 12 y compárelo. Y aquí el fiado está bien hecho." (Haga la cuenta con la cifra que le den; no cite precios de la competencia de memoria.) |
| "¿Y si se daña el computador?" | "Hay copia diaria y le enseño a guardar una semanal en USB o Drive. Con esa copia queda funcionando en otro computador." |
| "¿Hace factura electrónica?" | "Esta versión imprime tickets de venta. Si la DIAN le exige factura electrónica, lo conversamos aparte." (No prometa fecha.) |
| "¿Y después del primer año me toca seguir pagando?" | "No le toca: el programa es suyo y sigue funcionando. Si quiere seguir con mi respaldo, son $180.000 al año **[confirmar]**, como un SOAT: soporte, actualizaciones y, si se le daña el computador, se lo dejo funcionando con su fiado. Si no, cada atención se cobra aparte." |
| "¿Lo puedo ver desde mi casa?" | "No: funciona en el negocio, en el computador y en los celulares de la misma WiFi. Por eso no necesita internet ni mensualidad." |
| "¿Puedo tener dos cajas?" | "Puede vender desde varios equipos en la misma WiFi, todos a la misma caja." |
| "¿Le cobra intereses al cliente?" | "No. Le avisa cuando la deuda está vieja o pasa el tope; usted decide." |
| "Es muy caro." | "Es precio de cliente fundador: la mitad del precio normal, y la mitad del pago la hace a los 30 días si el sistema le cuadró la caja." |

## 6. No prometa

- Factura electrónica DIAN, contabilidad o nómina.
- Acceso desde fuera del negocio, nube o varias sucursales.
- Varias cajas abiertas al tiempo.
- Envío automático de WhatsApp, o conexión con datáfono o Nequi (se registra el método, no se conecta).
- Bloqueo del fiado por tope, intereses o puntaje de clientes.
- Anular cualquier venta vieja desde una pantalla de historial (solo las recientes).
- Que el cajero vea la cartera o la lista de clientes con saldos (recibe abonos desde **Punto de Venta › Abonar** con la caja abierta, viendo solo el saldo de ese cliente; la pantalla **Clientes** y la cartera son del dueño).
- Que el catálogo base trae precios reales (todos son $1.000 de referencia).
- Modo restaurante o catálogo en línea: existen pero no se están vendiendo.

## 7. Cierre de la venta

1. Confirmar el paquete y el precio (`oferta-y-contrato.md`).
2. Agendar la instalación y pedir **antes** la lista de productos que más rotan y el cuaderno de fiados (foto o lista).
3. Definir quiénes usan el sistema: ¿un administrador y un cajero, o dos administradores?
