# SPRINT-16 — Seguridad, Integridad y Producción

**Objetivo:** Eliminar los bloqueadores de paso a producción identificados en la revisión de QA de seguridad e integridad (gaps.md). Nadie puede salir a vender si el sistema permite forjar tokens o corrompe los backups en caliente.

**Fecha inicio:** (por definir)

**Referencias:** `project-docs/gaps.md` (Advisor de Producción).

---

## Tareas

| US | Nombre | Estado | Fecha |
|----|--------|--------|-------|
| US-070 | Seguridad: JWT secret auto-generado, no hardcodeado | [x] Completo | 2026-09-29 |
| US-071 | Integridad: Apertura de caja transaccional (Tx) | [x] Completo | 2026-09-29 |
| US-072 | Integridad: Backup SQLite online (`VACUUM INTO`) | [x] Completo | 2026-09-29 |
| US-073 | UX: Filtro de movimientos de caja por sesión actual | [x] Ya cubierto (verificado 2026-09-30) | 2026-09-30 |

**Esfuerzo total estimado: 2.5 días** (Sprint crítico de estabilización del core).

---

## Orden de implementación recomendado

| Orden | US | Razón |
|-------|----|-------|
| **1** | US-070 | Riesgo crítico. Modifica la autenticación global, hay que estabilizarlo primero. |
| **2** | US-072 | Un backup corrupto arruina el negocio. El refactor de DB es pequeño pero crucial. |
| **3** | US-071 | Corrige el silent error de cajas. |
| **4** | US-073 | Mejora visual requerida por UX; las cajas con muchos movimientos traban la lectura diaria. |

---

## US-070 — Seguridad: JWT secret auto-generado

**Como** dueño del sistema,
**quiero** que mi instalación tenga una llave criptográfica única,
**para** que nadie que analice el binario público pueda falsificar sesiones en mi servidor local.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Crítico (Bloqueador Prod).

**Estado verificado:**
- `internal/middleware/middleware.go:22`: `getJWTSecret()` retorna `"pos-secret-key-change-in-prod"`.
- `business_config` tiene una fila única (`id=1`) donde ya se guardan configuraciones del negocio.

**Regla de negocio:**
- En el arranque (`setup.go` o `appmw`), si no existe un `jwt_secret` en `business_config`, generar una cadena aleatoria criptográficamente segura (ej: 32 bytes en base64) y guardarla.
- Todos los tokens emitidos se firman y validan con ese secreto de la DB, cacheado en memoria para no penalizar el middleware.

**DoD:**
- [x] Nueva columna `jwt_secret` en `business_config` (Migración 16).
- [x] Lógica de generación automática si el campo está vacío.
- [x] Eliminar el string hardcodeado de `middleware.go`.

---

## US-071 — Integridad: Apertura de caja transaccional

**Como** cajero,
**quiero** que si falla el registro de la base (sencillo inicial), la caja no quede reportada como abierta falsamente,
**para** no causar huecos contables en mi cierre.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Crítico (Corrupción de estado).

**Estado verificado:**
- `internal/service/register.go:65`: Si `AddMovement` falla durante `OpenRegister`, hace un `Close` silencioso y retorna un error, pero si el `Close` falla, la DB queda inconsistente.

**Regla de negocio:**
- La creación de la sesión (`Register`) y el `RegisterMovement` inicial deben ir en una única transacción de SQLite (`sql.Tx`).
- Si algo falla, el `ROLLBACK` garantiza que la caja nunca se abrió contablemente.

**DoD:**
- [x] Modificar `repository.Register.Create` o agregar método que permita inyectar TX, o hacerlo en el propio Service usando un Unit of Work.
- [x] Pruebas unitarias que fallen el movimiento forzosamente y verifiquen que la caja no se crea.

---

## US-072 — Integridad: Backup SQLite online

**Como** administrador,
**quiero** que el respaldo diario automático capture la base de datos completa sin detener el POS,
**para** que los datos del WAL (transacciones recientes) no se pierdan o corrompan la copia.

**Esfuerzo estimado:** 0.5 días. **Impacto:** Crítico (Pérdida de datos).

**Estado verificado:**
- `internal/database/migrate.go:130`: `backupDB` usa `io.Copy(out, src)`.
- SQLite en modo WAL genera archivos `-wal` y `-shm`. El copiado bruto los ignora y rompe la consistencia.

**Regla de negocio:**
- Reemplazar `io.Copy` por la ejecución de la sentencia `VACUUM INTO 'ruta_backup.db'`.
- SQLite garantiza que `VACUUM INTO` genera una copia consistente y consolidada (sin necesidad de copiar el WAL) incluso si hay escrituras simultáneas.

**DoD:**
- [x] Cambiar `backupDB` para ejecutar `db.Exec("VACUUM INTO ?", dst)`.
- [x] Eliminar lógica vieja y probar ejecución manual del backup.

---

## US-073 — UX: Filtro de movimientos de caja por sesión

**Como** cajero,
**quiero** que al abrir mi turno solo vea los movimientos de ingresos y egresos de mi sesión actual,
**para** poder cuadrar mi efectivo real contra el sistema sin confundirme con los días anteriores.

**Esfuerzo estimado:** 1 día. **Impacto:** Medio (Usabilidad).

**Estado verificado:**
- **Corregido 2026-09-30:** esta afirmación era falsa. `GET /registers/{id}/movements` filtra por `register_id` (`internal/handler/register.go:157-168`, `internal/repository/register.go:145-146`) y `frontend/app/register/page.tsx:476,775-783` muestra solo los de la sesión actual y agrupa el historial por sesión (`:461-501`). Sin cambios de código.

**Acceptance Criteria:**
- [x] La UI de "Caja" solo lista movimientos donde `register_id` coincide con la sesión actual.
- [x] Si se consulta un historial, se puede agrupar o filtrar por sesión pasada.

**DoD:**
- [x] Endpoint de lista de movimientos soporta filtro por `register_id`.
- [x] Interfaz gráfica actualizada.
