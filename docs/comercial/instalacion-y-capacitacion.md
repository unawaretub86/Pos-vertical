# Checklist del instalador y guion de capacitación

> Actualizado: 13/11/2026. Para el instalador (Andrey). Sigue el orden del roadmap: ensayo (G-2), go-live (G-3) y semana 1 (salida de la etapa 1). Los pasos técnicos detallados están en `INSTALL.md`.

---

## 1. Antes de la visita

☐ Compilar desde `dev` (o `master`) ya verificado:
```
make build
go build -ldflags="-s -w" -o reset-password.exe ./cmd/reset-password
go build -ldflags="-s -w" -o reset-data.exe ./cmd/reset-data
```
(`make build` solo produce `pos.exe`, `Makefile:17`; las dos herramientas se compilan aparte.)

☐ Pedir al cliente: lista o foto de los productos que más rotan, y el cuaderno de fiados (nombre, celular, cuánto debe, desde cuándo).
☐ Preparar el archivo de productos en la plantilla de importación, con precios reales y costo, guardado como CSV. Ojo con el IVA: el sistema lo suma encima del precio; si la tienda no cobra IVA, 0 %.
☐ **Ensayo completo cronometrado** en un computador igual al del cliente (G-2): carpeta vacía, `pos.exe`, asistente, inicio automático, reinicio, carga de productos y deudas anteriores, un turno completo con una anulación, cierre contra papel, `reset-data`. Meta: menos de 30 minutos desde la carpeta vacía hasta el primer ticket.

**Kit en la memoria USB:**
☐ `pos.exe` ☐ `reset-password.exe` ☐ `reset-data.exe`
☐ Archivo de productos del cliente (CSV)
☐ `INSTALL.md` y `MANUAL-USUARIO.md` en PDF
☐ Dos copias impresas de la hoja de entrega (`oferta-y-contrato.md`, parte 2)
☐ Datos de demo (10 productos, 2 clientes con fiado), en un `data.db` aparte que **no** se deja en el computador del cliente

---

## 2. Día de la instalación

| # | Paso | ☐ |
|---|---|---|
| 1 | Demo de 5 minutos con los datos de prueba (`guion-de-venta.md` § 4) | ☐ |
| 2 | Revisar la zona horaria de Windows: (UTC-05:00) Bogotá | ☐ |
| 3 | Copiar `pos.exe` a `C:\POS\`, abrirlo, completar el asistente con los datos reales del negocio | ☐ |
| 4 | Confirmar en la ventana negra `time zone: UTC-05:00` y sin aviso | ☐ |
| 5 | `pos.exe --install-autostart`, `netplwiz`, reiniciar y comprobar que arranca solo | ☐ |
| 6 | Configuración: dirección, logo, encabezado e impresora en **Formato de tickets**, **Guardar cambios**. Con ESC/POS: **Imprimir prueba** (tildes, ñ, corte). Con **Navegador**: imprimir el ticket de una venta de prueba | ☐ |
| 7 | Configuración › Copias de seguridad: carpeta en USB o `D:` si el cliente tiene una | ☐ |
| 8 | Crear usuarios: dueño (ya creado) y cajero(s) | ☐ |
| 9 | Importar productos; revisar precios y costos de los 10 que más venden | ☐ |
| 10 | Clientes con **deuda anterior**, con su fecha | ☐ |
| 11 | Lector de código de barras: 10 lecturas seguidas en **Punto de Venta** | ☐ |
| 12 | Ventas de prueba con el cliente (efectivo, fiado, mixta), una anulación y un cierre | ☐ |
| 13 | Cerrar `pos.exe`. En CMD, en `C:\POS`: `reset-data.exe` (solo simula y muestra qué borraría), revisar, y luego `reset-data.exe --apply` y escribir la confirmación que pide. Borra lo de prueba, conserva productos, clientes y deudas anteriores, y deja una copia `data.db.pre-reset-*`. Volver a abrir `pos.exe` | ☐ |
| 14 | Abrir la primera caja real con el cliente y registrar la primera venta real | ☐ |
| 15 | Capacitación de 45 minutos (sección 3) | ☐ |
| 16 | Copiar una copia de seguridad fuera del computador, delante del cliente | ☐ |
| 17 | Firmar la hoja de entrega y recibir el primer pago | ☐ |

---

## 3. Capacitación de 45 minutos

Con el dueño y el cajero, cada uno en su usuario. Ellos tocan el teclado, usted no.

| Min | Tema | Quién | Qué deben poder hacer solos |
|---|---|---|---|
| 0–5 | Encender, la ventana negra, apagar | ambos | Minimizar la consola sin cerrarla; saber que si la cierran se apaga |
| 5–15 | Vender | ambos | Venta en efectivo con cambio, transferencia, mixta, **Dejar en espera** |
| 15–20 | Fiado | ambos | Fiar con **Credito** a un cliente ya creado, ver **Debe**, responder el aviso **¿Fiar de todas formas?** |
| 20–25 | Abonos y clientes | dueño | Crear cliente, abonar escogiendo **Efectivo** o **Transferencia** (Nequi y Daviplata son transferencia). El cajero no tiene pantalla de abonos: cuando un cliente paga, avisa al dueño |
| 25–32 | Caja | ambos | Abrir, **Entrada** y **Salida** con nota, cerrar contando billetes. Con faltante no deja cerrar: recontar; si de verdad falta, **Salida** con la nota "Faltante de cierre" y avisar al dueño |
| 32–38 | Lo del dueño | dueño | Dashboard, cartera, **Cobrar** por WhatsApp, anular una venta, reportes, preguntarle al asistente |
| 38–42 | Productos | dueño | Crear un producto con costo, entrada de inventario, ver lo que se está acabando |
| 42–45 | Si algo falla | ambos | Impresora: cambiar a **Navegador**; respaldo semanal a USB; a quién escribirle |

**Lo que más se olvida (repítalo al final):**
1. No cerrar la ventana negra.
2. Abono por Nequi = **Transferencia**, no efectivo; si no, la caja no cuadra.
3. Cada semana copiar un archivo `data.db.backup-*` (nunca `data.db` solo) fuera del computador.
4. Al cajero no se le da el usuario del dueño.

---

## 4. Semana 1 y salida de la etapa 1

- **Día 2:** mensaje proactivo: "¿Cómo van las ventas? ¿Algo raro?"
- **Día 7:** visita o llamada. ¿Siguen registrando todo en el sistema o volvieron al cuaderno para el fiado?
- Cada falla reportada: anotarla, y corregir de inmediato solo lo que bloquea el uso diario.

**Medir a mano (roadmap R-175), en una libreta:**

| Medida | Meta |
|---|---|
| Tiempo desde la carpeta vacía hasta el primer ticket | menos de 30 min |
| Venta en efectivo en hora pico | menos de 10 s |
| Venta fiada en hora pico | menos de 15 s |
| Deudores activos cargados en la semana 1 | 80 % o más |
| Días seguidos con caja cerrada en el sistema y cuadrada con el papel | 7 |
| Errores "locked" o de base de datos reportados | 0 |

Con 7 días seguidos de caja cerrada y cuadrada se cobra el segundo pago y la etapa 1 queda cumplida.
