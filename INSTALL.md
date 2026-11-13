# Guía de instalación — Sistema POS

> Actualizado: 13/11/2026. Para el uso diario vea `MANUAL-USUARIO.md`.

## Requisitos

- Computador con **Windows 10 o superior**.
- Navegador web (Chrome, Edge o Firefox).
- **Zona horaria de Windows: (UTC-05:00) Bogotá, Lima, Quito.** Revísela en *Configuración › Hora e idioma › Fecha y hora*. Si está en otra zona, las ventas de la noche quedan en el día equivocado y el cierre de caja no cuadra.
- No requiere internet ni instalar nada más.

---

## Instalación

1. Copie `pos.exe` a una carpeta fija, por ejemplo `C:\POS\pos.exe`. No lo deje en *Descargas* ni en el escritorio.
2. Haga doble clic en `pos.exe`.
   > Si Windows muestra una advertencia de seguridad, haga clic en **"Más información"** y luego en **"Ejecutar de todas formas"**.
3. Se abre el navegador y queda abierta una ventana negra de consola: **minimícela, no la cierre**, porque al cerrarla el sistema se detiene.
4. En la ventana negra debe aparecer la zona horaria `UTC-05:00`. Si aparece un aviso de zona horaria, corrija la zona de Windows (Requisitos) y vuelva a abrir `pos.exe`.

---

## Primer uso

1. El navegador abre `http://localhost:8080` y muestra el asistente de configuración inicial:
   - **Paso 1:** nombre completo, usuario y contraseña del dueño (mínimo 4 caracteres). Este será el **administrador**.
   - **Paso 2:** nombre del negocio y NIT.
2. Entre a **Configuración** y complete dirección, logo, encabezado del ticket e impresora en **Formato de tickets**. Con impresión directa (ESC/POS) use **Guardar cambios** y luego **Imprimir prueba**; con el método **Navegador**, haga una venta de prueba e imprima su ticket.
3. Cree el usuario del cajero en **Usuarios**.
4. **Productos.** Cárguelos desde un archivo de Excel guardado como **CSV** (no se acepta `.xlsx`) o con el catálogo base: Inventario › Productos › **Importar productos** › **Cargar catálogo base de tienda**. **Todos los precios del catálogo base son de $1.000 de referencia: usted debe ajustarlos antes de vender.**
5. **Fiados del cuaderno.** Cree cada cliente que debe y registre su **deuda anterior** con la fecha en que empezó la deuda.

> Si alguna vez entra con la contraseña `admin`, el sistema le exige cambiarla (mínimo 6 caracteres) antes de continuar.

---

## Inicio automático con Windows

Para que el sistema arranque solo cuando el computador se enciende y entra a Windows:

1. Abra el Símbolo del sistema (CMD) en la carpeta del programa, por ejemplo `C:\POS`, y ejecute:
   ```
   pos.exe --install-autostart
   ```
   Debe aparecer `Inicio automático instalado`. Si Windows niega el permiso, repita el comando desde un CMD abierto como administrador.
2. Para quitarlo: `pos.exe --uninstall-autostart`.
3. El sistema arranca cuando el usuario de Windows **inicia sesión**. Para que el computador entre solo después de un corte de luz, configure el inicio de sesión automático: presione `Windows + R`, escriba `netplwiz`, desmarque "Los usuarios deben escribir su nombre y contraseña para usar este equipo" y acepte.
4. Reinicie el computador para comprobarlo: en menos de un minuto debe abrirse la ventana negra y el navegador.

---

## Copias de seguridad (Backup)

El sistema crea automáticamente una copia de seguridad cada día y conserva las últimas 30.

```
C:\POS\
  pos.exe
  data.db                    ← base de datos principal
  data.db.backup-*           ← copias automáticas diarias (últimas 30)
  data.db-wal, data.db-shm   ← archivos de trabajo de la base de datos (no los borre con el sistema encendido)
  data.db.pre-migration-*    ← copia antes de una actualización que cambia la base de datos (no se borra sola)
  data.db.pre-reset-*        ← copia antes de la limpieza de datos de prueba (no se borra sola)
```

- Puede elegir otra carpeta para las copias diarias (por ejemplo una memoria USB o `D:\Respaldos`) en **Configuración → Copias de seguridad**. Si esa carpeta no está disponible, la copia se guarda junto a `pos.exe`, y Configuración y el dashboard muestran un aviso.
- El archivo `data.db.pre-migration-*` es la copia que el sistema hace justo antes de una actualización que cambia la base de datos; no se borra automáticamente, y puede eliminarla usted cuando ya no la necesite.
- **Importante:** una copia en el mismo computador no protege contra un disco dañado o un robo. Cada semana copie un archivo `data.db.backup-*` a una memoria USB o a Google Drive.
- **Para una copia manual, copie el archivo `data.db.backup-*` más reciente, no `data.db`.** El archivo `data.db` solo no tiene las ventas más recientes (quedan en `data.db-wal`) y su copia puede salir incompleta.
- **Para restaurar:** cierre `pos.exe`; mueva a otra carpeta `data.db`, `data.db-wal` y `data.db-shm`; copie el respaldo en la carpeta de `pos.exe` con el nombre `data.db`; abra `pos.exe`. Si deja `data.db-wal` en la carpeta, el sistema vuelve a poner los datos viejos encima del respaldo.

---

## Acceso desde el celular o tablet (misma WiFi)

1. En **Configuración** active "Permitir acceso desde otros dispositivos de mi red WiFi" y guarde.
2. Cierre y vuelva a abrir `pos.exe` (el cambio no se aplica en caliente).
3. Configuración muestra la dirección, por ejemplo `http://192.168.1.15:8080`. Ábrala en el otro dispositivo.
4. Si no carga, permita `pos.exe` en el Firewall de Windows (redes privadas).

---

## Herramientas de soporte (solo el instalador)

Van en la memoria USB de soporte, no en el computador del cliente:

- **`reset-password.exe`:** restablece la contraseña de un usuario que la olvidó.
- **`reset-data.exe`:** borra las ventas, cajas, abonos y gastos de prueba antes de empezar a vender de verdad. Conserva productos, clientes, deudas anteriores y la configuración, y antes hace una copia `data.db.pre-reset-*`. **Cierre `pos.exe` antes de usarla.**

---

## Actualizar a una versión nueva

1. Cierre `pos.exe` (la ventana negra).
2. Reemplace `pos.exe` por la versión nueva, en la misma carpeta. No toque `data.db`.
3. Abra `pos.exe`. Si la versión nueva cambia la base de datos, primero guarda una copia `data.db.pre-migration-*` y luego la actualiza.

---

## Soporte

Si tiene algún problema con el sistema, comuníquese con soporte:

- WhatsApp: **[número de soporte]**
- Horario: **[horario de soporte]**
