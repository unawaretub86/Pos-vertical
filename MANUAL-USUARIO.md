# Manual de Usuario — POS

Sistema de punto de venta para Windows. Sin internet, sin instalación, sin complicaciones.

---

## Requisitos

- Windows 10 u 11 (64 bits)
- Un navegador instalado (Chrome, Edge o Firefox)

---

## Cómo iniciar el sistema

1. Haz doble clic en `pos.exe`.
2. Se abrirá el navegador automáticamente en `http://localhost:8080`.
3. Si no abre solo, escribe esa dirección en el navegador manualmente.

> Para usar el POS desde un celular o tablet de la misma WiFi, activa el acceso en Configuración y reinicia `pos.exe` (ver INSTALL.md).

> **No cierres la ventana negra de consola** mientras usas el sistema. Es el servidor que mantiene todo funcionando.

### Primera vez

Al iniciar por primera vez se pedirá configurar:
- Nombre del negocio
- Moneda
- Usuario y contraseña del administrador

---

## Cómo cerrar el sistema

1. Cierra el navegador.
2. Cierra la ventana de consola (o presiona `Ctrl + C` adentro).

---

## Pantallas del sistema

| Pantalla | Para qué sirve |
|---|---|
| **POS** | Hacer ventas |
| **Caja** | Abrir y cerrar caja, ver movimientos de efectivo |
| **Productos** | Agregar, editar o eliminar productos |
| **Categorías** | Organizar los productos en grupos |
| **Clientes** | Registrar clientes, ver su historial y saldo |
| **Cartera** | Ver y cobrar las deudas pendientes de clientes |
| **Inventario** | Registrar entradas de mercancía |
| **Reportes** | Ver ventas por día, semana o mes |
| **Usuarios** | Crear o desactivar empleados |
| **Configuración** | Cambiar datos del negocio, logo y moneda |

---

## Hacer una venta

1. Ve a **POS** en el menú lateral.
2. Busca el producto por nombre o escanea su código de barras.
3. Ajusta la cantidad si es necesario.
4. Presiona **Cobrar**.
5. Selecciona el método de pago: Efectivo, Tarjeta o Crédito.
6. Confirma. Listo.

---

## Copia de seguridad (Backup)

**Toda la información del sistema está en un solo archivo: `data.db`**, ubicado en la misma carpeta que `pos.exe`.

### Hacer backup

1. **Cierra el sistema** primero (importante).
2. Copia el archivo `data.db` a un lugar seguro:
   - USB
   - Google Drive, Dropbox u otro servicio en la nube
   - Otra carpeta del equipo

**Recomendación:** Haz backup al menos una vez por semana.

### Restaurar un backup

1. Cierra el sistema.
2. Reemplaza el archivo `data.db` actual con tu copia de seguridad.
3. Inicia `pos.exe` normalmente.

> No copies `data.db` con el sistema abierto — el archivo puede quedar incompleto.

---

## Impresión de tickets

El sistema imprime usando el diálogo de impresión del navegador (igual que imprimir una página web).

### Cómo imprimir

1. Termina una venta.
2. En la pantalla de confirmación, presiona **Imprimir Ticket**.
3. Elige tu impresora en el diálogo y confirma.

### Impresoras compatibles

Cualquier impresora que Windows reconozca funciona:

| Tipo de impresora | Compatible | Notas |
|---|---|---|
| Impresora de oficina (laser o inkjet) | ✅ | Ticket en hoja A4 |
| Impresora térmica 80mm (Epson TM, Bixolon, Sewoo, etc.) | ✅ | Configura tamaño de papel en Windows |
| Impresora térmica 58mm | ✅ | Configura tamaño de papel en Windows |
| PDF (guardar en archivo) | ✅ | Usa "Microsoft Print to PDF" |

#### Configurar impresora térmica en Windows

Si la impresión térmica sale cortada o con márgenes incorrectos:

1. Ve a **Configuración de Windows → Bluetooth y dispositivos → Impresoras y escáneres**.
2. Selecciona tu impresora térmica → **Preferencias de impresión**.
3. En **Tamaño de papel**, elige `80mm` o `58mm` según tu modelo.
4. Guarda los cambios.

### Impresión directa (sin diálogo del navegador)

Con la impresión directa el ticket sale de la impresora térmica apenas se cobra la venta, sin abrir el diálogo del navegador y sin configurar márgenes.

1. Entra a **Configuración → Formato de tickets** (solo el administrador).
2. En **Método de impresión** elige:
   - **Navegador**: el comportamiento de siempre (valor por defecto).
   - **ESC/POS red**: impresora con cable de red o WiFi. Escribe su **IP** (ej. `192.168.1.50`) y el **puerto** (normalmente `9100`).
   - **ESC/POS Windows**: impresora instalada en Windows (por ejemplo por USB). Escribe el **nombre exacto** que aparece en *Impresoras y escáneres*.
3. Elige el **Ancho del papel**: `58 mm` o `80 mm`.
4. Presiona **Guardar** y luego **Imprimir prueba**. Debe salir un ticket que dice `PRUEBA DE IMPRESION`, con las tildes y la ñ bien escritas y el corte del papel.

Al cobrar, el ticket se envía solo a la impresora y aparece el aviso **Ticket enviado a la impresora**. El ticket sigue en pantalla: el botón **Imprimir** saca otra copia. Si la impresora está apagada o desconectada verá un aviso con el motivo; la venta ya quedó registrada y puede intentar de nuevo con **Imprimir**.

> La opción **ESC/POS Windows** solo funciona cuando el POS corre en Windows.

---

## Solución de problemas

| Problema | Qué hacer |
|---|---|
| El navegador no abre solo | Escribe `http://localhost:8080` en el navegador manualmente |
| "This site can't be reached" | Verifica que `pos.exe` esté corriendo (ventana de consola abierta) |
| La sesión se cierra sola | Es normal por inactividad. Vuelve a iniciar sesión. |
| Se perdieron datos | Restaura el último backup de `data.db` |
| El ticket no sale bien en impresora térmica | Configura el tamaño de papel correcto (ver sección de impresoras) |

---

## Preguntas frecuentes

**¿Necesito internet?**
No. El sistema funciona completamente sin conexión.

**¿Puedo usar el sistema en más de un equipo?**
El sistema está diseñado para funcionar en un solo equipo. Para usar en red, consulta con soporte técnico.

**¿Qué pasa si se va la luz?**
Al reiniciar el equipo, simplemente vuelve a abrir `pos.exe`. SQLite es resistente a cierres inesperados.

**¿Dónde está el usuario y contraseña inicial?**
`admin` / `admin`. Cámbialo en el primer uso desde **Configuración → Usuarios**.
