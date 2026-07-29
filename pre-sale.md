# Pre-venta — Primer prospecto real (minimercado, pareja de esposos)

> Creado 27 julio 2026. Prospecto real conseguido, dispuesto a pagar (~1M COP es el rango de mercado que maneja).
> Objetivo: instalación exitosa, no piloto gratis. Este documento reemplaza/complementa el enfoque "gratis 3 meses" de `pos-validation.md` para este cliente específico.

---

## 1. Datos del prospecto

- **Tipo de negocio:** Minimercado.
- **Quiénes:** Pareja de esposos, ambos atienden.
- **Hardware:** PC con Windows en el mostrador (confirmado). → No se necesita fix de bind LAN, `localhost:8080` sirve tal cual.
- **Catálogo:** "Demasiados productos" — no cabe carga manual en la visita. Requiere pre-carga.
- **Número de contacto:** ⏳ pendiente, se consigue mañana.

---

## 2. Preguntas para el prospecto (mañana, por WhatsApp o en persona)

1. **¿Los dos van a cobrar, o uno maneja el negocio y el otro solo ayuda en mostrador?**
   → Define si se crean 2 usuarios `admin` o 1 `admin` + 1 `cashier`.
2. **Pedir foto/lista/nota de voz de los 40-60 productos que más rotan** (bebidas, aseo, abarrotes básicos).
   → Se carga en el ensayo, antes de la visita — no frente a ellos.

### Pregunta que se hace EN la visita, no antes

3. **"¿Qué te dan los programas de 1M que conoces?"**
   → No preguntar antes. Su respuesta es munición de venta en vivo y define expectativas reales a gestionar (soporte, actualizaciones, etc).

---

## 3. Decisión comercial

- **No regalar.** Cliente espera pagar, valora el software en ~1M COP. Piloto gratis quema esa percepción de valor.
- **No cobrar 1M de entrada tampoco.** Precio sugerido: **400.000–500.000 COP "lanzamiento"**, con trato explícito: *"el precio normal es 1M, tú eres mi primer cliente, pagas la mitad a cambio de feedback y referidos."*
- **Opcional:** 50% al instalar, 50% a los 30 días si está satisfecho — baja su riesgo, te da excusa formal para volver y medir adopción.
- **Alcance a definir por escrito:**
  - Incluye: instalación, capacitación inicial, soporte por WhatsApp durante X meses.
  - No incluye: facturación electrónica DIAN, hardware (lector, impresora), soporte fuera del horario acordado.
- **Hoja de entrega de 1 página** para firmar ambos: alcance, precio, qué no incluye, datos son del cliente. Sin abogados, solo claridad. ⏳ pendiente redactar.

---

## 4. Checklist técnico

### Ya resuelto / verificado

- [x] Roles en el sistema: solo `admin` y `cashier` (`internal/service/user.go`). Cada esposo con su propio usuario, no compartir login.
- [x] `reset-password.exe` compilado (antes solo corría con `go run`, exigía Go instalado en el PC del cliente). Va en el USB de soporte junto a `pos.exe`.
- [x] Bind LAN — **descartado para este cliente**, PC está en el mostrador, no se necesita acceso desde celular.
- [x] Venta, fiado, persistencia, backup diario, anulación de venta — verificado en código (build + tests pasan).

### Pendiente antes de la visita

- [ ] **Ensayo completo cronometrado de instalación** — carpeta limpia, `pos.exe`, seguir `INSTALL.md` paso a paso como si fueras el tendero. Pieza más importante que falta. Meta: bajar de 30 min.
- [ ] Cuando llegue la lista de productos del prospecto → cargarla en el ensayo, no en la visita.
- [ ] Preparar datos demo (10 productos, 2 clientes con fiado) para la demo de 5 min antes de instalar en limpio.
- [ ] Verificar edge case: apagar el PC a mitad de una venta, reabrir, confirmar que no se pierde nada.
- [ ] Probar anulación de venta de punta a punta con las manos, no solo confiar en el código.
- [ ] Redactar hoja de entrega de 1 página (precio, alcance, firma).

---

## 5. Guion del día de la visita

1. Demo 5 min con datos de prueba: venta + venta fiada + lista de deudores. Antes de instalar nada.
2. Preguntar #3 (arriba) — qué esperan de un programa de 1M. Ajustar pitch en vivo con la respuesta.
3. Instalar en su PC: wizard con datos reales del negocio (nombre, NIT, moneda).
4. Catálogo ya pre-cargado (de la lista que mandaron). Cargar 5-10 productos en vivo junto a ellos — el objetivo es que **ellos aprendan a hacerlo**, no que tú cargues todo.
5. Crear los 2 usuarios con el rol que definieron en la pregunta #1.
6. Registrar 2-3 ventas reales juntos, una de ellas fiada. Mostrar cómo anular una venta mal registrada.
7. Mostrar dónde queda el backup (`data.db.backup-*`) y qué hacer si el PC falla.
8. Firmar hoja de entrega, cobrar primer pago.

---

## 6. Seguimiento semana 1

- **Día 2:** mensaje proactivo — "¿cómo van las ventas? ¿algo raro?" No esperar a que reporten.
- **Día 7:** visita o llamada. Indicador real de adopción: ¿siguen registrando TODO en el sistema, o volvieron al cuaderno para el fiado?
- Cada bug reportado: anotar, estimar, arreglar solo si bloquea el uso diario.

---

## 7. Kit de soporte a llevar

- `pos.exe`
- `reset-password.exe`
- USB con ambos + copia de `INSTALL.md` impreso o en PDF
- Catálogo pre-cargado (o el .db ya preparado con productos, si se arma con anticipación)
