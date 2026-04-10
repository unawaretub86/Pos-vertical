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

3. El programa se iniciará en segundo plano (no abre ninguna ventana).

---

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
  data.db.backup-*     ← copias automáticas (últimas 7)
```

Para hacer una copia manual, simplemente copie el archivo `data.db` a otro lugar (USB, Google Drive, etc.) con el programa cerrado.

---

## Soporte

Si tiene algún problema con el sistema, comuníquese con su proveedor de software.
