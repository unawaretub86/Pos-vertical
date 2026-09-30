# Dominio — Servidor y acceso LAN

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** bind del servidor, acceso desde otros dispositivos, servicio del frontend embebido
> **Actualizado:** 2026-09-30 · Verificado contra el código (US-054: `cmd/pos/main.go`, `web/static.go`, `internal/netutil/lan.go`, `internal/handler/setup.go`).

## Acceso LAN

- `business_config.lan_access_enabled = 0` (o config ilegible): escucha en `127.0.0.1:8080`. `= 1`: `0.0.0.0:8080`. Se lee al arrancar; cambiarlo aplica al reiniciar `pos.exe`.
- El navegador se abre siempre en `http://localhost:8080`.
- `GET /api/setup/lan-info` (solo admin) devuelve `{ip, url}`; `ip` es la primera IPv4 privada (10/8, 172.16/12, 192.168/16); nunca loopback ni 169.254.x. Sin IP: ambos vacíos y Ajustes muestra ayuda.
- Con el toggle activo Ajustes avisa de cambiar la contraseña inicial `admin/admin`.

## Frontend embebido

Orden de resolución: archivo exacto -> `<ruta>.html` (sin redirigir) -> `customers/_.html` o `catalogo/_.html` para `/customers/*` y `/catalogo/*` -> 404 si empieza por `/_next/` o tiene extensión -> `index.html`.

La UI usa `/api` relativo salvo que `NEXT_PUBLIC_API_URL` esté definido (dev en :3000).
