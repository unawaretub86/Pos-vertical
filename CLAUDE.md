# CLAUDE.md — Pos-vertical

POS local, offline-first, un solo ejecutable, para tiendas y distribuidoras en Colombia. El diferenciador es el **fiado** (crédito a clientes), que los competidores manejan mal. Sin nube, sin mensualidad, sin internet.

**Andrey** es Product Owner y decide alcance y prioridad. Claude es Tech Lead y desarrollador.

---

## Regla 1 — Verificar antes de afirmar

**Ninguna afirmación sobre el código entra en un plan, un doc o una respuesta sin `archivo:línea` que la respalde.**

Los documentos de `project-docs/` envejecen más rápido que el código. Si un doc dice "X no existe", se hace `grep` antes de repetirlo. Si dice "falta implementar Y", se verifica que siga faltando.

Esto no es ceremonia: al planear SPRINT-14 esta regla cazó cuatro afirmaciones falsas heredadas de docs viejos, entre ellas "no hay tests en el backend" (hay 13 archivos `_test.go`).

Preferir *"verifiqué que…"* sobre *"según el doc…"*.

## Regla 2 — Si cambia el comportamiento, cambia el spec

`project-docs/spec/` es la única carpeta que debe ser cierta siempre. Un commit que cambia una regla de negocio actualiza el spec en el mismo commit. Sin esto, `spec/` se convierte en otro doc histórico en seis meses.

## Regla 3 — Un solo roadmap

`project-docs/roadmap.md` es el único lugar donde se registra lo que viene. Los documentos de sprint **no reservan números de sprint futuros** ni llevan su propia tabla de roadmap. (Se hacía antes, y produjo que SPRINT-13 y SPRINT-14 reclamaran el mismo SPRINT-15.)

---

## Stack

**Backend** — Go 1.25, módulo `github.com/andreyo/pos`
- Router: `chi/v5` · Auth: `golang-jwt/v5` · DB: `modernc.org/sqlite`
- **`modernc.org/sqlite` es SQLite en Go puro, sin CGO.** Es lo que permite compilar un `.exe` sin toolchain de C. No introducir dependencias con CGO sin decisión explícita del PO.
- Binarios: `cmd/pos` (el producto) y `cmd/reset-password` (utilidad CLI)

**Frontend** — Next.js 16.1.6 (App Router), React 19.2.3, TypeScript
- Gestor de paquetes: **pnpm**, no npm
- Tests: vitest · Estilos: Tailwind v4 + `@base-ui/react` + shadcn

## Comandos

```bash
go build ./...          # compila backend
go test ./...           # tests backend

cd frontend
pnpm build              # compila frontend
pnpm test               # vitest
pnpm lint               # eslint
```

CI (`.github/workflows/test.yml`) corre exactamente esos cuatro en cada push. Si pasan localmente, pasa CI.

## Arquitectura

Tres capas, en este orden de dependencia:

```
internal/repository/   SQL. Interfaz pública + struct privada + NewXRepository(db *sql.DB)
internal/service/      Reglas de negocio. Recibe repositorios por interfaz.
internal/handler/      HTTP. Sin SQL, sin lógica de negocio.
```

- Todo se cablea en `handler.New` (`internal/handler/handler.go:38`).
- Las rutas se registran en `Routes()` del mismo archivo. Los grupos con `appmw.RequireAdmin` son solo para admin — ahí viven dashboard, reportes y todo lo que exponga ventas, ganancias o cartera.
- Migraciones: slice numerado en `internal/database/migrate.go`. **Nunca editar una migración ya aplicada**; siempre agregar la siguiente.
- Errores envueltos con `%w`, nunca silenciados.
- SQL con parámetros `?`, jamás concatenado.

**Frontend:**
- `frontend/app/layout.tsx` es el layout **raíz**: envuelve también `/login`, `/setup` y `/onboarding`. Lo que requiera sesión NO va ahí.
- `frontend/components/AppLayout.tsx` es el contenedor **autenticado**. Ahí va lo que solo ven usuarios logueados.
- Moneda: `formatCOP` de `frontend/lib/format.ts`. HTTP: `api` de `frontend/lib/api.ts`.
- Colores por variables CSS (`var(--canvas)`, `var(--surface-2)`, `var(--border)`, `var(--text-primary)`). Ningún color fijo, o se rompe el tema oscuro.
- Mobile-first: todo funciona en 320px.

## Tests

Patrón establecido: `setupTestDB(t)` con `database.New(":memory:")` y seed por `INSERT` literales — ver `internal/handler/dashboard_test.go:14`. Seguirlo, no inventar otro fixture.

Los tests llevan números concretos, no aserciones vagas: *"3 pagos de $30k + $20k + $10k → total 60.000, sobra 0"*.

## Convenciones

- **Texto de cara al usuario: español.** Código, nombres de variables y comentarios: inglés.
- Moneda: pesos colombianos sin decimales, separador de miles con punto → `$1.234.567`.
- Rangos de fecha: **siempre semiabiertos `[from, to)`**. Nunca `BETWEEN`, que es inclusivo en ambos extremos y duplica las ventas de medianoche.
- Zona horaria: hora local del servidor. El POS corre en la máquina de la tienda.
- Borrado de entidades referenciadas históricamente: lógico (`active = 0`), nunca `DELETE` físico.

## Flujo de trabajo

Sprints en `project-docs/SPRINT-XX-*.md`. El loop de iteración está en `project-docs/BACKLOG-Development-Workflow.md`: se toma la primera US sin marcar, se implementa contra sus criterios de aceptación, se marca `[x]` al terminar.

Antes de planear una feature, leer `project-docs/README.md` para saber qué documento manda sobre qué. **No todo lo que está en `project-docs/` es vigente** — varios documentos son investigación fechada o historia del MVP, y están marcados como tales en su encabezado.

## Trampas conocidas

- `PLAN.md` y `ARQUITECTURA.md` describen el código de abril–mayo 2026. Sirven como historia, no como estado actual. Ejemplo: `PLAN.md:1134` reporta un bug de `cash_in` en el cierre de caja que **ya está arreglado** (`internal/service/register.go:113`).
- Iterar un `map` en Go es aleatorio por diseño. Si el orden importa (evaluación de reglas, prioridades), usar un slice ordenado o los tests fallarán de forma intermitente.
- Los documentos `*finds*` (`find-pos.md`, `finds-betters.md`, `better-pos.md`, `better-finds-pos.md`) son investigación de mercado, no inventario del código. Sus afirmaciones de "esto no existe" se verifican antes de usarse.
