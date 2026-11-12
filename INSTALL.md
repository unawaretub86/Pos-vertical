# Guía de Instalación — Sistema POS

## Requisitos

- Computador con **Windows 10 o superior**
- Navegador web (Chrome, Edge o Firefox)
- No requiere instalar nada más

---

## Instalación

1. Copie el archivo `pos.exe` a una carpeta de su elección, por ejemplo:
   `C:\POS\pos.exe`

2. Haga doble clic en `pos.exe` para ejecutarlo.
   > Si Windows muestra una advertencia de seguridad, haga clic en **"Más información"** y luego en **"Ejecutar de todas formas"**.

3. El programa se inicia y abre el navegador. Quedará abierta una ventana negra de consola: **minimícela, no la cierre**, porque al cerrarla el sistema se detiene.

---

## Inicio automático con Windows

Para que el sistema arranque solo cuando el computador se enciende y entra a Windows:

1. Abra el Símbolo del sistema (CMD) en la carpeta del programa, por ejemplo `C:\POS`, y ejecute:
   ```
   pos.exe --install-autostart
   ```
   Debe aparecer `Inicio automático instalado`. Si Windows pide permisos de administrador, repita el comando desde un CMD abierto como administrador.
2. Para quitarlo: `pos.exe --uninstall-autostart`.
3. El sistema arranca cuando el usuario de Windows **inicia sesión**. Para que el computador entre solo después de un corte de luz, configure el inicio de sesión automático: presione `Windows + R`, escriba `netplwiz`, desmarque "Los usuarios deben escribir su nombre y contraseña para usar este equipo" y acepte.
4. No cierre la ventana de consola (minimícela). Si va a usar la herramienta de limpieza de datos (`reset-data`), cierre `pos.exe` primero.

---

## Acceso desde el celular o tablet (misma WiFi)

1. En Ajustes activa "Permitir acceso desde otros dispositivos de mi red WiFi" y guarda.
2. Cierra y vuelve a abrir `pos.exe` (el cambio no se aplica en caliente).
3. Ajustes muestra la dirección, por ejemplo `http://192.168.1.15:8080`. Ábrela en el otro dispositivo.
4. Si no carga, permite `pos.exe` en el Firewall de Windows (redes privadas).
5. Cambia la contraseña inicial `admin/admin`.

## Primer uso

1. Abra su navegador y vaya a la dirección:
   **http://localhost:8080**

2. Ingrese con las credenciales iniciales:
   - Usuario: `admin`
   - Contraseña: `admin`

3. El sistema le pedirá que cambie su contraseña. Elija una contraseña nueva (mínimo 6 caracteres) y confírmela.

4. Configure su negocio: ingrese el nombre, NIT y moneda. Luego cree su primer cajero y registre sus productos.

> A partir de este momento el sistema está listo para usarse.

---

## Copias de seguridad (Backup)

El sistema crea automáticamente una copia de seguridad de sus datos cada día.

Los archivos de respaldo se encuentran en la misma carpeta donde está `pos.exe`:

```
C:\POS\
  pos.exe
  data.db              ← base de datos principal
  data.db.backup-*     ← copias automáticas (últimas 30)
```

Se guarda una copia por día y se conservan las últimas 30. Puede elegir otra carpeta (por ejemplo una memoria USB o `D:\Respaldos`) en **Ajustes → Copias de seguridad**. Si esa carpeta no está disponible, la copia se guarda junto a `pos.exe` y Ajustes y el inicio muestran un aviso.

**Importante:** una copia en el mismo computador no protege contra un disco dañado. Cada semana copie un archivo `data.db.backup-*` a una memoria USB o a Google Drive.

El archivo `data.db.pre-migration-*` es la copia que el sistema hace justo antes de una actualización; no se borra automáticamente, y puede eliminarla usted cuando ya no la necesite.

Para hacer una copia manual, simplemente copie el archivo `data.db` a otro lugar (USB, Google Drive, etc.) con el programa cerrado.

---

## Soporte

Si tiene algún problema con el sistema, comuníquese con su proveedor de software.
