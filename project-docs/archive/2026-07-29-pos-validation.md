# Plan de Validación POS — Resumen de decisiones y checklist de arranque

> Creado 23 julio 2026. Resultado de la sesión de revisión de `idea-forge-repair-shop-review.md`.
> Este documento se lleva al proyecto **pos-vertical** para ejecutar la verificación y el plan de 14 días.

---

## 1. Contexto: qué se descubrió y qué se decidió

### Situación real (dicha con honestidad)

- **Cero clientes, cero ventas, nada desplegado en uso real** — ni en POS ni en taller.
- Dos codebases, un pitch fallido (Julián fuera), tres rondas de análisis de mercado — y ninguna conversación con alguien que pague.
- El cuello de botella nunca fue el software: es **distribución y contacto con clientes reales**. Cambiar de vertical no escapa de eso.

### Hallazgos de mercado (búsqueda 23 jul 2026)

| Hallazgo | Implicación |
|---|---|
| **Pitz entra a Colombia** (fundadora colombiana, $5M USD, 1.400+ talleres, meta 8.000 en 2026, marketplace de autopartes lanzado feb 2026) | El vertical de taller ya no es "competencia sin VC". Ventana de 12-18 meses si se retoma. |
| **DIAN ofrece facturación electrónica gratis** (<2.000 facturas/año, portal Muisca) | "Cumple con DIAN" no es gancho de venta por sí solo. El gancho es la **integración**: factura + caja + fiado + historial sin doble digitación. |
| Precio de mercado software de taller Colombia: **$80.000–$300.000 COP/mes** | Entrar a $30.000–$60.000 COP/mes es una cuña válida por debajo del piso del mercado. |
| Ningún competidor maneja **fiado/crédito de barrio** | El fiado es el diferenciador real. Pitch de una línea: *"el único sistema que entiende el fiado."* |

### Decisión tomada

1. **Fuerza principal en POS.** Taller queda **en pausa, no muerto** — el motor (caja, fiado, inventario) es compartido; todo lo que mejore en POS le sirve al taller si se reabre.
2. **Congelar desarrollo de features 30 días.** Solo se arregla lo que bloquee el checklist de la sección 3.
3. **Meta única: 1 negocio real usando el POS el día 14** (gratis está bien).
4. No hay cuarta actualización del doc de mercado. Lo que falta no está en Google.

### El miedo del costo, con números

- VPS $5-12 USD/mes (~$20-50k COP) o free tier (Supabase/Neon) aguanta 20-50 negocios pequeños multi-tenant.
- **1 solo cliente a $50k/mes ya paga la infra.** Con 10 clientes: margen ~90%.
- Regla dura: **tope $50.000 COP/mes de bolsillo propio.** No se sube de infra hasta tener cliente pagando.
- Offline/local: problema del cliente 20, no del cliente 0. PWA con cache básico cubre por ahora. No construir sync offline para clientes que no existen.

---

## 2. Reencuadre clave: "listo" ¿para qué?

El POS **no** necesita estar listo para 100 clientes pagando. Necesita estar listo para **un piloto gratis** con trato explícito: *"está en desarrollo, lo usas gratis 3 meses, me dices qué falla."*

El piloto gratis es licencia para ser imperfecto. Los bugs que encuentre el tendero son el roadmap real. "No está listo" sin checklist concreto es evitación disfrazada de prudencia.

---

## 3. Checklist de verificación (ejecutar en pos-vertical)

### Mínimo funcional — sin esto NO hay piloto

- [ ] **Venta:** registrar una venta completa (productos, total, pago) sin crashear.
- [ ] **Fiado:** registrar venta fiada a un cliente y ver cuánto debe cada cliente (lista de deudores con saldos).
- [ ] **Persistencia:** cerrar la app/sesión y reabrir — los datos siguen ahí. Nada se pierde.

### Listo para licencias de prueba — necesario para instalar a 2-3 pilotos

- [ ] **Acceso por negocio:** cada piloto entra con su propia cuenta/usuario y **solo ve sus datos** (si es multi-tenant, verificar aislamiento; si es instancia por cliente, verificar que levantar una nueva toma <1 hora).
- [ ] **Desplegado y accesible:** hay una URL real que funciona desde el celular del tendero (no solo `localhost`). Probado en un Android de gama media.
- [ ] **Alta de negocio en frío:** puedo crear el negocio, su usuario y dejarlo operando en una visita de ~30 min (productos iniciales cargados a mano está bien).
- [ ] **Backup:** existe respaldo automático o manual documentado de la BD. Perderle los datos a un piloto mata la confianza para siempre.
- [ ] **Anular/corregir:** el tendero puede corregir una venta mal registrada (anulación básica). Se equivocan mucho al inicio.

### Explícitamente NO necesario para el piloto (no tocar)

- Facturación electrónica DIAN, reportes avanzados, modo offline, multi-sucursal, pagos en línea, landing page bonita, panel de administración de licencias. Nada de esto bloquea al cliente 1.

### Regla de trabajo

Cada ítem que falle → se anota, se estima, y **solo se arregla si bloquea el checklist**. Todo lo demás va a una lista de "después del piloto".

---

## 4. Plan de 14 días

| Días | Acción | Entregable |
|---|---|---|
| **1-3** | Ejecutar checklist sección 3 en pos-vertical. Arreglar solo bloqueantes. | Checklist con ✅/❌ y fixes hechos |
| **4** | Grabar video demo 60-90 seg: pantalla registrando venta fiada + lista de deudores. Sin edición, voz propia. | Video listo para WhatsApp/FB |
| **5-7** | Unirse a 5 grupos de Facebook ("tenderos Colombia", "comerciantes [ciudad]", "emprendedores [ciudad]"). Publicar oferta de piloto. **Sábado:** visitar 5 tiendas físicas del barrio y preguntar *"¿cómo llevas las cuentas del fiado?"* | Publicaciones hechas + 5 conversaciones de calle |
| **8-14** | A quien responda: demo por WhatsApp/videollamada. Instalar a los primeros 2-3 negocios. | **1 negocio real usándolo el día 14** |

### Texto para los grupos (no vender — pedir ayuda)

> "Estoy desarrollando una app gratuita para llevar el fiado y las cuentas de la tienda. Busco 5 negocios que la prueben gratis 3 meses a cambio de contarme qué les sirve y qué no. ¿A alguien le interesa?"

### Guion de calle (tienda física, cero presión)

1. Comprar algo.
2. "¿Cómo llevas las cuentas del fiado? ¿Cuaderno?"
3. Escuchar. Si hay dolor: "Estoy haciendo una app para eso, ¿te muestro 2 minutos?"
4. Cerrar con WhatsApp, no con venta.

---

## 5. Criterios de decisión al día 14/30

- **1+ negocio usándolo activamente** → seguir: iterar con su feedback, buscar los siguientes 2-3. El precio ($30-50k/mes) se valida al convertir pilotos en el mes 4.
- **Interés pero nadie instaló** → diagnosticar cuál fue la fricción real (¿demo? ¿confianza? ¿el problema no duele?) antes de tocar código.
- **Cero interés en 30 días con esfuerzo real de contacto** → respuesta válida: es un proyecto de aprendizaje, no (todavía) un negocio. Se aprendieron stack, arquitectura y análisis de mercado. Cerrar sin culpa o replantear.

---

## 6. Fuentes de la investigación (23 jul 2026)

- Pitz a Colombia: valoraanalitik.com — "Pitz acelera su llegada para transformar los talleres en Colombia"
- Pitz $5M y marketplace: startuplinks.world, clusterindustrial.com.mx (feb 2026)
- Precios software taller Colombia: blog.itsquery.com
- DIAN facturación gratis y obligados 2026: datafinsolutions.com, blog.misfacturas.com.co
