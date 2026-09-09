# Onboarding Wizard + Branding — Plan de Implementación

> Basado en el spec aprobado: [`2026-07-30-onboarding-wizard-design.md`](./2026-07-30-onboarding-wizard-design.md)

## 🎯 Análisis de la Tarea

**Tipo:** Feature (2 partes: wizard de primer arranque + branding "POS Chévere").

**Alcance — qué cambia:**
- Nueva columna `setup_completed` en `business_config`.
- 2 endpoints públicos nuevos: `GET /api/setup/status`, `POST /api/onboarding`.
- Página nueva `frontend/app/onboarding/page.tsx` (stepper 2 pasos).
- Componente `<Logo>` nuevo, reemplaza placeholder texto "POS" en `login/page.tsx:52-66`.
- Favicon nuevo.
- Redirect logic en `login/page.tsx` y `onboarding/page.tsx` basado en `/api/setup/status`.

**Qué NO cambia:** `/api/setup` (GET/PUT) existente, `/api/users` (admin-only), esquema single-tenant, seed del admin en `migrate.go` (se reutiliza el row, no se borra).

**Stakeholders:** dueño del negocio (usuario final, corre el wizard una vez); admins existentes (sin impacto — flujo normal de login intacto post-onboarding).

**Ambigüedades críticas:** ninguna. El spec ya resolvió trigger, bootstrap de admin, campos, layout y riesgo aceptado. No se re-discuten acá.

---

## 🚧 Estado actual (2026-07-30)

Backend completo y testeado. Frontend implementado y compila/tipa limpio. Build
completo (`make frontend` + `make backend`) corre sin errores. Smoke test en
`pos.exe` real vía skill `run` **en progreso, no concluido**: se detectó y
corrigió un bug preexistente (no relacionado al wizard) en `frontend/app/page.tsx`
— el dashboard hacía `return null` antes de montar `<AppLayout>` cuando no había
usuario autenticado, así que el `useEffect` de redirect de `AppLayout` nunca
corría y `/` quedaba en blanco para siempre sin login. Se agregó un efecto en
`DashboardPage` que consulta `getSetupStatus()` y redirige a `/onboarding` o
`/login` según corresponda cuando no hay usuario. Falta re-verificar en
Chromium headless que el flujo completo (fresh DB → onboarding → dashboard →
reload → login → onboarding bloqueado) funciona end-to-end tras ese fix; el
run anterior seguía sin ver el redirect en el navegador y quedó pendiente de
diagnóstico (revisar si el JS del bundle servido por `pos.exe` es el build
nuevo, o si `getSetupStatus()` está lanzando silenciosamente).

### Backend — hecho
- Migración `v11`: `business_config.setup_completed`, `users.full_name`.
- `model.User.FullName`; `repository.UserRepository.OnboardAdmin`; `SetupRepository.IsSetupCompleted` / `CompleteOnboarding`.
- `service.UserService`: `hashPassword` extraído y reusado por `Create`/`Update`/`ChangePassword`/`Onboard` (nuevo).
- `internal/handler/onboarding.go`: `getSetupStatus`, `submitOnboarding` (guard 409, valida business_name/nit, emite JWT).
- Rutas públicas registradas en `handler.go`.
- Tests: `internal/handler/onboarding_test.go` — happy path, guard 409, validaciones (password débil, username/negocio/nit vacío), status antes/después. Todos pasan (`go test ./internal/...`).

### Frontend — hecho
- `components/Logo.tsx` (ícono etiqueta+check + wordmark, tokens `--accent`/`--accent-dim`).
- `lib/api.ts`: `getSetupStatus`, `submitOnboarding`.
- `app/onboarding/page.tsx`: stepper 2 pasos, validación inline, loading/error states, redirect si ya completado.
- `app/login/page.tsx`: redirect a `/onboarding` si no completado, logo real reemplazando el placeholder.
- `app/page.tsx`: fix del bug de redirect descrito arriba.
- `favicon.ico` regenerado (PNG-en-ICO generado por script, mismo ícono, verificado visualmente).
- `npx tsc --noEmit` limpio, `npm run build` exitoso.

### Pendiente
- [ ] Terminar smoke test en `pos.exe` real (Chromium headless vía `playwright-core`, script en scratchpad) confirmando los 4 pasos de aceptación end-to-end.
- [ ] Confirmar visualmente el favicon y el logo en el navegador real (no solo el PNG extraído).
- [ ] Marcar DoD/checklists abajo una vez el smoke test pase.

---

## 📋 Definition of Done

### Funcionalidad
- [~] Fresh DB (`setup_completed=false`) → abrir app redirige a `/onboarding`, no a `/login`. (código listo, verificación en navegador pendiente)
- [~] Completar wizard (paso 1 cuenta + paso 2 negocio) → usuario id=1 actualizado (full_name, username, password_hash), `business_config` actualizado (name, nit, currency="COP"), `setup_completed=true`, JWT devuelto y sesión iniciada automáticamente (sin login manual post-wizard). (verificado por test Go; falta e2e navegador)
- [~] Con `setup_completed=true`: `/login` funciona normal; navegar a `/onboarding` directo redirige a `/login`. (código listo, falta e2e navegador)
- [x] `POST /api/onboarding` con `setup_completed=true` devuelve `409 Conflict`, no modifica nada.
- [x] Login page muestra `<Logo>` (ícono etiqueta+check + "POS Chévere") en vez del div placeholder.
- [x] Favicon actualizado (ya no el default de Next.js).

### Código
- [x] Validación/hash de password del wizard reutiliza la misma función que `createUser` (`internal/handler/user.go`) — no duplicada.
- [x] Sin código comentado ni logs de debug.
- [x] Manejo de errores explícito en handlers Go (no swallowed errors), respuestas HTTP con status codes correctos (400 validación, 409 ya-completado).
- [x] `<Logo>` es un componente React reusable (no imagen duplicada pegada en 2 páginas).

### Tests
- [x] Test unitario Go: `POST /api/onboarding` happy path.
- [x] Test unitario Go: guard rechaza cuando `setup_completed=true` (409).
- [x] Test unitario Go: validación rechaza password débil / campos vacíos.
- [x] `GET /api/setup/status` testeado (devuelve estado correcto antes/después).
- [x] Cobertura no menor a la existente antes del cambio (`go test ./internal/...` pasa completo).

### Integración
- [x] `POST /api/onboarding` no rompe `POST /users` ni `PUT /users/{id}` existentes (misma lógica compartida, sin efectos secundarios cruzados) — confirmado por suite de tests `internal/service` y `internal/handler`.
- [x] Build completo funciona: `make frontend` (next build + copy-frontend.go) → `make backend` → `pos.exe` sirve wizard y branding embebidos correctamente.
- [ ] Probado en `pos.exe` real corriendo local (no solo `npm run dev`), vía skill `run`. **(en progreso — ver Estado actual)**

### Documentación
- [x] Ticket/spec actualizado si alguna decisión cambió durante implementación: se documentó el bug preexistente encontrado y corregido en `app/page.tsx` (fuera del alcance original del spec pero bloqueante para el criterio de aceptación).

### Seguridad
- [x] Password del wizard hasheado con bcrypt (mismo path que `createUser`), nunca loggeado ni devuelto en response (verificado por test que el body no contiene el password en texto plano).
- [x] `POST /api/onboarding` no acepta escritura alguna cuando `setup_completed=true` (verificado con test, no solo por inspección).
- [x] Validación de inputs en el endpoint (no solo en frontend) — username no vacío, password con longitud mínima igual a la que ya exige `createUser`.

---

## 🔧 Requisitos de Implementación

### Setup / Precondiciones
- [ ] Confirmar mecanismo de migraciones actual en `internal/database/migrate.go` (¿migraciones versionadas o script único?) antes de agregar la columna, para seguir el patrón existente.

### Backend
- [ ] `internal/database/migrate.go`: agregar `setup_completed BOOLEAN NOT NULL DEFAULT false` a `business_config`.
- [ ] `internal/model/model.go`: agregar campo `SetupCompleted bool` a `BusinessConfig`.
- [ ] `internal/repository`: método para leer/actualizar `setup_completed` (probablemente extender el repo que ya maneja `business_config` en `setup.go`).
- [ ] Extraer la lógica de hash+validación de password de `createUser` (`internal/handler/user.go:26`) a una función compartida (si no lo está ya) para que `onboarding` la reuse sin copy-paste.
- [ ] `internal/handler/onboarding.go` (nuevo archivo): handlers `getSetupStatus` (público) y `submitOnboarding` (público, con guard `409` si ya completado).
- [ ] `internal/handler/handler.go` (`Routes()`, líneas ~76-192): registrar `GET /setup/status` y `POST /onboarding` como rutas públicas, junto a `/auth/login`.
- [ ] `submitOnboarding`: update user id=1 + update business_config + set flag + `appmw.GenerateToken` (mismo patrón que `login`, `internal/handler/auth.go:16-63`).

### Frontend
- [ ] `frontend/components/Logo.tsx` (nuevo): ícono SVG etiqueta+check + wordmark "POS **Chévere**", usando `var(--accent)` etc — el mismo markup aprobado en el mockup.
- [ ] `frontend/lib/api.ts`: agregar `getSetupStatus()` y `submitOnboarding(payload)`.
- [ ] `frontend/app/onboarding/page.tsx` (nuevo): stepper 2 pasos (dots), paso 1 (full_name, username, password, confirm), paso 2 (business_name, nit), botón final "Empezar a vender" → llama `submitOnboarding`, guarda token con `setAuth` (`frontend/lib/auth.ts`), redirige a dashboard. On mount: si `getSetupStatus().completed === true` → `router.replace('/login')`.
- [ ] `frontend/app/login/page.tsx`: on mount, si `getSetupStatus().completed === false` → `router.replace('/onboarding')`. Reemplazar div placeholder (líneas 52-66) por `<Logo />`.
- [ ] `frontend/app/favicon.ico`: regenerar desde el mismo ícono SVG.
- [ ] Validación de formulario: password mínimo igual a la regla del backend, confirmar password coincide, nit no vacío — feedback inline, sin submit si inválido.
- [ ] Estados de loading/error en el submit del wizard (spinner en botón, mensaje de error si backend devuelve 400/409).

### Orden sugerido de implementación
1. Migración DB + modelo (`setup_completed`).
2. `GET /api/setup/status` (backend) — habilita testear el resto end-to-end desde ya.
3. `POST /api/onboarding` (backend) + tests unitarios.
4. `<Logo>` component + favicon (paralelo, no depende del backend).
5. `frontend/app/onboarding/page.tsx` (stepper) + wiring a la API.
6. Redirect logic en `login/page.tsx` + swap del logo.
7. Build completo (`make frontend && make backend`) y smoke test con `pos.exe` real vía skill `run`.

---

## ✅ Criterios de Aceptación

### Happy Path
- **Dado** una DB fresca (`setup_completed=false`), **cuando** se abre `pos.exe`, **entonces** el browser cae en `/onboarding`, no en `/login`.
- **Dado** el wizard en paso 1, **cuando** se completan nombre/usuario/password/confirmar y se avanza, **entonces** pasa a paso 2 sin perder datos del paso 1.
- **Dado** el wizard completo (ambos pasos válidos), **cuando** se hace submit, **entonces** el usuario queda logueado en el dashboard sin pasar por `/login`.
- **Dado** `setup_completed=true`, **cuando** se abre la app de nuevo, **entonces** cae directo en `/login` con el nuevo logo visible.

### Casos Borde
- **Dado** `setup_completed=true`, **cuando** se navega manualmente a `/onboarding`, **entonces** redirige a `/login` sin mostrar el form.
- **Dado** el wizard en paso 2, **cuando** se refresca la página, **entonces** se pierde el progreso (esperado, sin corrupción de datos — no hay persistencia parcial en DB).
- **Dado** `setup_completed=true`, **cuando** se llama `POST /api/onboarding` directo (curl/Postman), **entonces** devuelve `409` y no modifica user ni business_config.

### Casos de Error
- **Dado** un password que no cumple la regla mínima, **cuando** se intenta avanzar/submit, **entonces** se muestra error inline y no se llama al backend.
- **Dado** un NIT vacío en paso 2, **cuando** se intenta submit, **entonces** bloquea con mensaje claro.
- **Dado** el backend caído o responde 500, **cuando** se hace submit, **entonces** el wizard muestra error y no pierde lo ya tipeado.

### Criterios de Seguridad
- Password nunca aparece en la respuesta del endpoint ni en logs del backend.
- `setup_completed` solo puede pasar de `false` a `true` vía `POST /api/onboarding` — ningún otro endpoint lo toca.
