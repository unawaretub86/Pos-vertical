# SPRINT-04 — Prod Ready: Binary Integration & Production Polish

**Objetivo:** Cerrar los gaps que impiden entregar el producto a un cliente real. Al final de este sprint, `pos.exe` es el único artefacto que el cliente necesita.

**Fecha inicio:** 2026-04-09

---

## Prioridad 1 — Bloqueador de distribución

### US-019: Embed frontend estático en el binario Go

- [x] Exportar Next.js como estático y embeber en el binario con `//go:embed`
- **Problema:** Hoy el frontend y el backend se levantan por separado. El modelo de distribución requiere un solo `.exe`.
- **Acceptance Criteria:**
  - `next.config.js` tiene `output: 'export'` y `distDir: '../frontend-dist'`
  - El binario Go embebe `frontend-dist/**` con `//go:embed`
  - Rutas de API siguen bajo `/api/...`; todo lo demás sirve el frontend estático
  - La ruta raíz `/` sirve `index.html`; rutas de Next.js (SPA) hacen fallback a `index.html`
  - `go build` produce un único `.exe` que al ejecutarse abre correctamente en `localhost:8080`
  - No requiere Node.js ni ninguna dependencia en la máquina del cliente
- **Notas técnicas:**
  - Usar `net/http` `FileServer` con `fs.Sub` sobre el embed FS
  - El handler de fallback SPA debe interceptar 404s del FS y servir `index.html`
  - El script de build (Makefile o `.bat`) debe correr `npm run build` antes de `go build`

---

## Prioridad 2 — Completar Sprint 03

### US-017: CSV export for reports

- [x] Agregar botón de exportar CSV en la página de reportes
- **Acceptance Criteria:**
  - Botón "Exportar CSV" en `/reports` descarga el resumen de ventas filtrado
  - Columnas: fecha, total_revenue, total_cost, gross_margin, total_count
  - Respeta el rango de fechas y agrupación seleccionados
  - Encoding UTF-8 con BOM (para compatibilidad con Excel en Colombia)
  - Nombre de archivo: `reporte-ventas-YYYY-MM-DD.csv`
  - Generado en el frontend (no requiere nuevo endpoint)

### US-018: Automatic local backup (scheduled)

- [x] El binario hace una copia automática de `data.db` diariamente
- **Acceptance Criteria:**
  - Al arrancar y cada 24h, el binario copia `data.db` → `data.db.backup-YYYYMMDD`
  - Solo se mantienen los últimos 7 backups (elimina el más antiguo si hay más de 7)
  - Si el directorio no es escribible, loguea el error y continúa sin crashear
  - No interfiere con el WAL ni con operaciones en curso (checkpoint antes de copiar)

---

## Prioridad 3 — Seguridad básica

### US-020: Cambio de contraseña obligatorio en primer login

- [x] Forzar cambio de contraseña cuando el usuario usa la contraseña default `admin`
- **Problema:** El binario crea el usuario admin con contraseña `admin` (hardcoded en migración 1). Un cliente real nunca la cambiaría si no se le fuerza.
- **Acceptance Criteria:**
  - Al hacer login con contraseña `admin`, el backend responde `200` con campo `force_password_change: true`
  - El frontend redirige al usuario a una pantalla de cambio de contraseña antes de acceder al sistema
  - La pantalla de cambio requiere nueva contraseña (mín. 6 caracteres) + confirmación
  - Hasta no cambiarla, no se puede navegar a ninguna otra ruta
  - `PUT /api/users/me/password` acepta `{ current_password, new_password }` y actualiza el hash
  - No aplica a usuarios creados después del setup inicial

---

## Prioridad 4 — Documentación mínima

### US-021: README de instalación para el cliente

- [x] Crear `INSTALL.md` con instrucciones para instalar y usar el POS
- **Acceptance Criteria:**
  - Documento en español, lenguaje no técnico
  - Secciones: Requisitos (solo Windows), Instalación (copiar `.exe`, ejecutar), Primer uso (abrir navegador, setup), Backup (dónde están los archivos), Soporte (contacto)
  - Máximo 1 página impresa
  - Incluye captura de pantalla del paso de setup (o instrucción clara de qué esperar)

---

## Registro de progreso

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-019 | Embed frontend en binario | ✅ Completo | 2026-04-09 |
| US-017 | CSV export for reports | ✅ Completo | 2026-04-09 |
| US-018 | Auto backup scheduled | ✅ Completo | 2026-04-09 |
| US-020 | Cambio contraseña primer login | ✅ Completo | 2026-04-09 |
| US-021 | README instalación cliente | ✅ Completo | 2026-04-09 |
