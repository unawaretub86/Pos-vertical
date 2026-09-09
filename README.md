# POS Vertical

Sistema de Punto de Venta offline para pequeños negocios en Colombia. Corre como un único binario (`pos.exe`) sin dependencias externas.

---

## Requisitos de desarrollo

| Herramienta | Versión mínima | Para qué |
|---|---|---|
| Go | 1.21+ | Backend + compilación del binario |
| Node.js | 18+ | Compilar el frontend |
| pnpm | 9+ | Dependencias del frontend (único package manager soportado, ver `frontend/package.json#packageManager`) |
| make | cualquiera | Correr los scripts de build |

> **Windows:** `make` viene con Git for Windows (Git Bash) o se puede instalar con `winget install GnuWin32.Make`.

---

## Setup inicial (primera vez)

```bash
# 1. Clonar el repositorio
git clone <repo-url>
cd Pos-vertical

# 2. Instalar dependencias del frontend
cd frontend && pnpm install && cd ..

# 3. Descargar dependencias de Go
go mod download

# 4. Build completo
make build
```

Al terminar tendrás `pos.exe` en la raíz del proyecto.

---

## Comandos disponibles

```bash
make build          # Build completo: frontend → copia → binario Go
make frontend       # Solo compila el frontend (Next.js → web/out/)
make backend        # Solo compila el binario Go
make dev-frontend   # Servidor de desarrollo Next.js (localhost:3000)
make dev-backend    # Corre el servidor Go en modo desarrollo (localhost:8080)
```

---

## Estructura del proyecto

```
Pos-vertical/
├── cmd/pos/            # Entry point del servidor Go
├── internal/
│   ├── database/       # SQLite: conexión, migraciones, schema
│   ├── handler/        # Handlers HTTP (chi router)
│   ├── service/        # Lógica de negocio
│   ├── repository/     # Acceso a datos (interfaces)
│   ├── model/          # Structs / tipos del dominio
│   └── middleware/     # Auth JWT, CORS, logging
├── web/
│   ├── static.go       # go:embed — embebe web/out/ en el binario
│   └── out/            # ← generado por `make frontend`, NO editar a mano
├── frontend/           # Proyecto Next.js 16 (TypeScript + Tailwind)
│   ├── app/
│   │   ├── globals.css # Sistema de tokens CSS (colores, componentes)
│   │   └── layout.tsx  # Root layout con Inter font + data-theme
│   ├── out/            # ← generado por Next.js, NO editar a mano
│   └── next.config.ts
├── scripts/
│   └── copy-frontend.go  # Script cross-platform: copia frontend/out → web/out
├── project-docs/
│   ├── PLAN.md           # Backlog y checklist de desarrollo
│   ├── PRD.md            # Product Requirements Document
│   └── pos-prototype.html # Prototipo visual — fuente de verdad de UI
├── Makefile
├── go.mod
└── data.db             # Base de datos SQLite (se crea al ejecutar pos.exe)
```

---

## Arquitectura de build

El problema central es que Go no puede usar `//go:embed` con rutas `../` (no puede salir de su directorio). Por eso el flujo de build es:

```
pnpm run build
    └─→ frontend/out/          (Next.js static export)
            │
go run scripts/copy-frontend.go
            │
    └─→ web/out/               (copia, mismo nivel que web/static.go)
            │
go build ./cmd/pos
            │
    └─→ pos.exe                (binario con frontend embebido)
```

`scripts/copy-frontend.go` usa solo la librería estándar de Go — funciona igual en **Windows, Linux y Mac**. No usar `xcopy` ni `cp` directamente en el Makefile.

---

## Base de datos

- SQLite en modo WAL (Write-Ahead Logging) para mayor rendimiento
- Se crea automáticamente como `data.db` al ejecutar `pos.exe` por primera vez
- Las migraciones corren automáticamente al iniciar — no hay comando manual
- Usuario por defecto: `admin` / `admin` (cambiar en primer uso)
- **Backup:** copiar `data.db` a un lugar seguro. Es todo lo que necesitas.

---

## Modo desarrollo

Para desarrollo se corren frontend y backend por separado:

```bash
# Terminal 1 — frontend con hot reload
make dev-frontend    # localhost:3000

# Terminal 2 — backend Go
make dev-backend     # localhost:8080
```

El frontend en desarrollo apunta al backend en `localhost:8080` vía proxy configurado en Next.js.

> En producción (`pos.exe`) el backend sirve el frontend estático directamente — no hay dos servidores.

---

## Despliegue futuro (v2.0 — Linux server)

Si se despliega en un servidor Linux:
1. El `Makefile` y `scripts/copy-frontend.go` ya son cross-platform
2. Cambiar el target de compilación: `GOOS=linux GOARCH=amd64 go build -o pos ./cmd/pos`
3. El binario resultante no tiene dependencias externas — solo necesita el OS

```bash
# Build para Linux desde Windows
GOOS=linux GOARCH=amd64 make backend
```

---

## Decisiones técnicas relevantes

| Decisión | Razón |
|---|---|
| `modernc.org/sqlite` (pure Go) | No requiere CGO ni DLLs — el binario es 100% portable |
| Next.js static export | Permite embeber el frontend en el binario Go |
| `web/` package separado para embed | Go no permite `//go:embed` con rutas `../` |
| Script Go para copiar (no xcopy/cp) | Cross-platform: funciona en Windows y Linux |
| WAL mode en SQLite | Mejor rendimiento en escrituras concurrentes |
| JWT para auth | Stateless, sin necesidad de sesiones en servidor |
