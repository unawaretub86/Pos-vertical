# POS-Vertical: Contexto de Negocio/Producto, Death List Comercial & Lush Launch Roadmap

Documento estratégico construido a partir del estado real de la [rama `dev` de Pos-vertical](https://github.com/unawaretub86/Pos-vertical/tree/dev) y sus documentos de validación de negocio y producto ([pos-validation.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/pos-validation.md), [qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md), [PRD.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/PRD.md) y [mvp-context.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/mvp-context.md)).

## 1. Todo el Contexto de Negocio y Producto Extraído del Repositorio (`dev`)

### 1.1 Tesis de Negocio y Propuesta de Valor Central

* **Dolor del mercado**: Los comercios pequeños y medianos (PYMES) en Colombia operan con cuadernos de papel o sistemas POS en la nube que cobran mensualidades recurrentes, exigen internet estable y tienen curvas de aprendizaje complejas para cajeros o tenderos ([PRD.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/PRD.md), [mvp-context.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/mvp-context.md)).
* **Promesa de valor ("Zero-Monthly-Fee & 100% Offline")**: Un sistema POS local que funciona sin conexión a internet, sin pagos mensuales obligatorios, empaquetado en un único ejecutable (`pos.exe` generado desde [Makefile](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/dev/Makefile)) cuyo tiempo de puesta en marcha e instalación (`INSTALL.md` en la [rama `dev`](https://github.com/unawaretub86/Pos-vertical/tree/dev)) debe ser **menor a 30 minutos**.
* **Foco operativo hiper-local**: Prioriza los dos flujos donde más dinero pierden los comercios de barrio: el **descontrol de la cartera de clientes ("fiado")** y el **descuadre en el arqueo de caja diario** entre efectivo físico y transferencias (Nequi / Daviplata / Bancolombia), validado en [pos-validation.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/pos-validation.md) y [qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md).

### 1.2 Verticales Objetivo (ICP — Ideal Customer Profile)

| Vertical Objetivo | Caso de Uso Crítico en el Producto | Flujo Dominante en POS-Vertical | Fuente en Repositorio |
| :--- | :--- | :--- | :--- |
| **1. Tiendas de Barrio / Minimercados** | Alta rotación diaria de productos unidad/peso y préstamos informales a vecinos ("el cuaderno de fiado") | Carrito rápido de venta + asignación de saldo a cuenta de cliente ("Fiado") + abonos parciales | [PRD.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/PRD.md) & [mvp-context.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/mvp-context.md) |
| **2. Distribuidoras de Alimentos / Abarrotes** | Ventas de mayor volumen con pagos mixtos (efectivo + transferencia) y créditos a corto plazo | Control de inventario / stock, cartera por cobrar (`getPortfolio`) y ticket térmico de entrega | [mvp-context.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/mvp-context.md) & [qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md) |
| **3. Talleres de Bicicletas y Repuestos** | Venta combinada de repuestos físicos y mano de obra / servicios rápidos con recibo físico | Catálogo flexible, impresión térmica ESC/POS y cierre de caja diario (`MANUAL-USUARIO.md` en [rama `dev`](https://github.com/unawaretub86/Pos-vertical/tree/dev)) | [PRD.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/PRD.md) & [pos-validation.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/pos-validation.md) |

### 1.3 Módulos de Producto Implementados en `dev` (Las 7 Pantallas y Capacidades de Negocio)

Según la arquitectura de producto en la [rama `dev`](https://github.com/unawaretub86/Pos-vertical/tree/dev), [frontend/](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/tree/dev/frontend), [.interface-design/system.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/.interface-design/system.md) y [qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md), el producto se compone de:

1. **Punto de Venta / Caja Rápida (Carrito de Ventas)**: Búsqueda rápida de productos, cálculo de cambio/vueltas y cierre de venta en tres modalidades: **Efectivo**, **Transferencia** y **Fiado (Crédito)**.
2. **Gestión de Cartera y Clientes ("El Cuaderno Digital de Fiado")**: Registro de clientes deudores, cupo o saldo acumulado, historial de deudas y registro de abonos parciales o totales que impactan el flujo de caja del día (`getPortfolio`).
3. **Arqueo y Cierre de Caja Diario**: Apertura de turno con base de efectivo y cuadre al final del día comparando efectivo esperado vs. efectivo real contado, separando ventas de contado, transferencias y recaudo de fiados.
4. **Dashboard del Día (`/dashboard/today`)**: Vista ejecutiva para el dueño del negocio con ventas del día, fiado nuevo otorgado, fiado cobrado y efectivo estimado en gaveta ([qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md)).
5. **Catálogo e Inventario de Productos**: Creación/edición de referencias, precios, control de existencias y alertas de bajo stock.
6. **Reportes de Negocio (`getSalesReport`, `getTopProducts`, `getPortfolio`)**: Análisis de ventas por periodo (día, semana, mes), ranking de productos más vendidos y estado de la cartera pendiente ([qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md)).
7. **Impresión Térmica ESC/POS y Configuración**: Emisión de tirillas/tickets físicos sin depender de drivers pesados de navegador y gestión de usuarios/roles locales (`MANUAL-USUARIO.md` en [rama `dev`](https://github.com/unawaretub86/Pos-vertical/tree/dev)).

## 2. Business & Product Death List (Qué Matar o Congelar a Nivel de Producto y Negocio)

Esta **Death List de Negocio y Producto** identifica las distracciones comerciales, fricciones de usuario y promesas prematuras que ponen en riesgo la adopción en los primeros comercios piloto.

| Ítem en la Death List de Producto / Negocio | Evidencia en `dev` | Decisión de Producto (Kill / Freeze Action) | Por Qué Mata el Negocio si NO se Elimina |
| :--- | :--- | :--- | :--- |
| **1. Prometer Facturación Electrónica DIAN (Factus) en el MVP inicial** | Documentado como postergado/mockup en [pos-validation.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/pos-validation.md) | **Congelar y ocultar de la oferta v1.0**: Posicionar el MVP estrictamente como **Control Interno de Caja, Inventario y Cartera Offline** | La facturación DIAN exige internet, certificados, resoluciones y soporte fiscal continuo; frena el cierre comercial rápido y rompe la promesa 100% offline |
| **2. Intentar vender a los 3 verticales al mismo tiempo en el Día 1** | [PRD.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/PRD.md) lista tiendas, distribuidoras y talleres de bicicletas | **Matar la dispersión comercial en el piloto**: Elegir **1 solo vertical ancla** (ej. Tiendas de Barrio / Minimercados con dolor agudo de "fiado") para los primeros 5 pilotos | Los talleres de bicicletas requieren órdenes de servicio/taller, mientras que las tiendas requieren velocidad de mostrador y fiado; atacar los 3 diluye el onboarding |
| **3. Entregar reportes y Resumen de Hoy (`/dashboard/today`) a medias** | En [qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md) faltan cierres de `/dashboard/today` y filtros día/semana/mes en `getSalesReport` | **Bloquear salida a calle hasta cerrar el cuadre diario**: El dueño compra el software para saber cuánta plata hay en el cajón y cuánto le deben hoy | Si en el primer día de uso el efectivo estimado no cuadra con los abonos de fiado recibidos, el tendero pierde la confianza y vuelve al cuaderno de papel |
| **4. Onboarding técnico o manuales largos para el tendero** | Existen `INSTALL.md` y `MANUAL-USUARIO.md` en la [rama `dev`](https://github.com/unawaretub86/Pos-vertical/tree/dev) | **Eliminar pasos manuales de consola para el cliente final**: Entregar instalación "llave en mano" (< 30 min) con acceso directo en el escritorio y catálogo semilla precargado | Un tendero o cajero no ejecuta comandos ni configura puertos; si no puede cobrar su primera venta en 5 minutos, abandona el producto |
| **5. Dejar el usuario `admin/admin` abierto sin personalización de negocio** | Semilla inicial documentada en [pos-validation.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/pos-validation.md) | **Eliminar el arranque genérico**: Incluir un asistente de primer uso (Nombre de la tienda, NIT/Teléfono para el ticket térmico y nueva clave/PIN del dueño) | Un ticket impreso sin el nombre real del negocio ni control de acceso entre dueño y cajero resta valor percibido inmediato |
| **6. Funciones multi-sucursal o sincronización cloud prematura** | Alcance acotado en [mvp-context.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/mvp-context.md) y [DEVELOPMENT-PLAN.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/DEVELOPMENT-PLAN.md) | **Matar cualquier desarrollo cloud en Fase Piloto**: Mantener 1 PC = 1 Licencia Local | Añade costos de infraestructura (servidores) a un producto cuya ventaja competitiva es no cobrar mensualidades obligatorias |

## 3. Lush Business & Product Launch Roadmap (Hoja de Ruta de Producto y Go-To-Market)

Hoja de ruta enfocada 100% en **valor de producto, validación con clientes reales, empaquetado comercial y monetización** a partir de la [rama `dev`](https://github.com/unawaretub86/Pos-vertical/tree/dev).

### Etapa 1: Pulido de "Momentos de Verdad" del Producto (Semana 1)

**Objetivo de Producto:** Asegurar que los 3 momentos que enamoran al dueño del comercio (Vender en <10s, Fiar/Cobrar en <15s y Cuadrar Caja en <2 min) funcionen sin fricción.

* [ ] **Cerrar el "Momento del Dueño" en `/dashboard/today`**: Terminar la visualización exacta de **Ventas de Contado + Abonos de Fiado Cobrados Hoy = Efectivo/Transferencia Esperada**, separada del **Fiado Nuevo Otorgado Hoy** ([qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md)).
* [ ] **Cerrar el "Momento de Análisis" en Reportes**: Habilitar la vista por **Día, Semana y Mes** en `getSalesReport` y asegurar que abrir `getTopProducts` o `getPortfolio` sin elegir fechas muestre de inmediato el corte actual del negocio ([qa-checklist.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/qa-checklist.md)).
* [ ] **Personalización del Ticket Térmico ESC/POS**: Asegurar que el recibo impreso ([pos-validation.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/pos-validation.md)) muestre claramente el encabezado del comercio, el detalle de la compra y, si es venta a crédito ("fiado"), el **saldo anterior, valor de la compra hoy y nuevo saldo total adeudado** por el cliente.

### Etapa 2: Empaquetado Comercial y Kit de Onboarding "< 30 Minutos" (Semana 2)

**Objetivo de Negocio:** Convertir el binario de la [rama `dev`](https://github.com/unawaretub86/Pos-vertical/tree/dev) ([Makefile](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/dev/Makefile)) en un producto comercializable que se instala en una sola visita presencial o sesión remota corta.

* [ ] **Plantillas de Catálogo por Vertical (Seed Data Comercial)**: Preparar 3 archivos/semillas iniciales de productos frecuentes (1: Tienda de barrio/abarrotes, 2: Distribuidora de alimentos, 3: Taller de bicicletas) para no entregar el POS vacío y ahorrarle al cliente 3 horas de digitación inicial.
* [ ] **Experiencia "Doble Clic y Vender"**: Empaquetar `pos.exe` junto con el instructivo de `INSTALL.md` y `MANUAL-USUARIO.md` ([rama `dev`](https://github.com/unawaretub86/Pos-vertical/tree/dev)) en un instalador que cree el ícono en el escritorio de Windows, configure el respaldo automático diario de la base de datos y abra el POS listo para operar.
* [ ] **Definición de Oferta Comercial Piloto (Pricing & Packaging)**:
  * **Licencia Vitalicia Local (Pago Único)**: Instalación del software offline + configuración inicial + capacitación corta con `MANUAL-USUARIO.md`.
  * **Combo Hardware + Software (Upsell)**: Venta con margen sobre impresora térmica USB 58mm/80mm + lector de código de barras + cajón monedero preconfigurados con el protocolo ESC/POS ([pos-validation.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/pos-validation.md)).

### Etapa 3: Piloto Controlado en Calle con 3–5 Comercios Reales (Semanas 3 y 4)

**Objetivo de Validación:** Medir uso diario real, retención y ahorro de dinero en comercios reales según las metas de [PRD.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/PRD.md) y [mvp-context.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/mvp-context.md).

| KPI de Negocio y Producto en el Piloto | Meta Objetivo ([PRD.md](https://www.google.com/search?q=https://github.com/unawaretub86/Pos-vertical/blob/master/PRD.md)) | Cómo se Mide en Terreno |
| :--- | :--- | :--- |
| **Tiempo de Instalación y Puesta en Marcha** | **< 30 minutos** por local | Cronometrado desde copiar `pos.exe` hasta imprimir el primer ticket de prueba en mostrador |
| **Velocidad de Venta en Hora Pico** | **< 10 segundos** (contado) / **< 15 segundos** (fiado) | Observación presencial en hora pico y revisión de marcas de tiempo de transacciones |
| **Migración del "Cuaderno de Fiado"** | **100% de deudores activos** cargados en la Semana 1 | Cantidad de clientes con saldo registrado en el módulo de Cartera (`getPortfolio`) |
| **Adherencia al Arqueo de Caja Diario** | **7 de 7 días** cerrando caja en el sistema | Registro consecutivo de cierres de turno sin regresar a cuentas en papel |
| **Cero Pérdida Operativa por Internet/Luz** | **100% disponibilidad offline** | Operación continua en local sin caídas ni desincronización de saldos |

### Etapa 4: Escalamiento Comercial y Módulos de Ingreso Recurrente Opcional (Mes 2+)

**Objetivo de Crecimiento:** Una vez validado el núcleo offline en tiendas piloto, activar vías de monetización recurrente opcionales sin romper la simplicidad del producto base.

* [ ] **Add-on 1 — Respaldo en la Nube y Reporte al WhatsApp del Dueño (Suscripción Baja Mensual)**: El POS sigue siendo 100% offline en la caja, pero cuando detecta internet al cerrar el turno, sube una copia encriptada de la base de datos y envía al dueño el resumen de `/dashboard/today` (ventas, fiado otorgado, abonos y efectivo en caja).
* [ ] **Add-on 2 — Módulo de Facturación Electrónica DIAN vía Factus (Por Paquete de Documentos o Mensualidad)**: Activación opcional del conector evaluado en [pos-validation.md](https://github.com/unawaretub86/Pos-vertical/blob/dev/pos-validation.md) únicamente para los comercios que por volumen o exigencia contable requieran emitir factura electrónica DIAN.
* [ ] **Add-on 3 — Modo Multi-Caja en Red Local (LAN)**: Permitir que una distribuidora o tienda grande conecte 2 puntos de pago en el mismo local contra el mismo ejecutable `pos.exe` en la red local sin pagar servidores en la nube.
