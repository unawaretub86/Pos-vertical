# Factura Electrónica DIAN — Research & Integration Guide

> **Tipo:** research · **Vigente:** NO · **Fecha:** mayo 2026
> Investigación de proveedores DIAN. **No es fuente de verdad sobre el código actual.** La regla de producto vigente está en [spec/domain/facturacion-dian.md](spec/domain/facturacion-dian.md).


> Investigación para integrar facturación electrónica en POS Vertical.
> Fuentes: DIAN oficial, Factus, Alanube, Facturalatam, Dataico, MATIAS API, Alegra, SOENAC, DLP Systems, EDICOM.
> Fecha: Mayo 2026

---

## 1. Contexto — Qué exige la DIAN

### Marco legal vigente
- **Resolución 000165 del 1 Nov 2023** — marco principal. Define todos los documentos electrónicos obligatorios.
- **Resolución 000202 de 2025** — añade obligación de autocompletar datos del comprador desde BD DIAN al ingresar cédula/NIT.
- Toda persona natural o jurídica responsable de IVA o INC debe facturar electrónicamente.

### Tipos de documentos (importante para POS)
| Documento | Cuándo aplica |
|-----------|--------------|
| **Factura de Venta Electrónica** | Venta a crédito, clientes con NIT, B2B |
| **Documento Equivalente Electrónico** | **Ventas en mostrador/caja — este aplica a nuestro POS** |
| Nota Crédito | Devoluciones/descuentos post-factura |
| Nota Débito | Ajustes por más valor |
| Documento Soporte | Compras a no obligados a facturar |

**El POS de mostrador emite Documento Equivalente Electrónico, no Factura de Venta.** Tiene su propio anexo técnico y calendario de implementación. Es crítico diferenciarlo en la arquitectura.

---

## 2. Cómo funciona técnicamente (el flujo real)

```
[POS captura venta]
        ↓
[Se envía JSON al proveedor API]
        ↓
[Proveedor genera XML UBL 2.1]
        ↓
[Proveedor calcula CUDE/CUFE + firma digital]
        ↓
[Proveedor transmite a DIAN (MUISCA)]
        ↓
[DIAN valida y responde ApplicationResponse]
        ↓
[Si aceptado → PDF+XML se envía al comprador]
        ↓
[Webhook/callback notifica al POS con resultado]
```

### Conceptos técnicos clave

**UBL 2.1** — formato XML estándar obligatorio que define la DIAN. Contiene todos los campos de la transacción, impuestos, partes, etc.

**CUFE / CUDE**
- CUFE = Código Único de Factura Electrónica (facturas de venta)
- CUDE = Código Único de Documento Equivalente (POS/mostrador) ← el que nos aplica
- Se calcula: SHA-384 sobre concatenación de campos del documento + clave técnica de la resolución de facturación

**Firma Digital**
- El XML debe firmarse con certificado P12 emitido por entidad acreditada por ONAC
- Garantiza integridad, autenticidad y no repudio
- El proveedor tecnológico lo maneja con su propio certificado habilitado ante DIAN

**Validación Previa**
- DIAN no acepta documentos después de emitidos — valida ANTES de que sean legalmente válidos
- DIAN devuelve `ApplicationResponse` con estado: aceptado / rechazado + motivo
- Solo después de aceptación el documento es válido y se puede entregar al comprador

**¿Directo o vía proveedor?**
Nadie implementa directo con los web services SOAP de DIAN en producción salvo los propios proveedores tecnológicos. La complejidad (UBL + CUFE + firma digital + MUISCA) justifica usar un intermediario. El integrador solo POST-ea JSON y recibe resultado.

---

## 3. Los proveedores — cómo funcionan y cuánto valen

### Modelo de negocio general
Todos los proveedores serios funcionan como **BaaS (Backend as a Service)**:
- Tu sistema envía JSON con datos de la venta
- Ellos generan XML UBL 2.1, calculan CUDE, firman, transmiten a DIAN
- Devuelven resultado + PDF + XML por webhook o response síncrona

**Cobran de dos formas:**
1. **Mensual/anual fijo** con N documentos incluidos → el más común para volumen bajo/medio
2. **Por documento** → útil para volúmenes irregulares o muy altos

### Comparativa de proveedores

| Proveedor | Modelo | Precio referencia | Fortaleza | Notas |
|-----------|--------|------------------|-----------|-------|
| **Factus** | Suscripción + docs | Cotizar | Mejor DX, OAuth2, docs excelentes | SDK en GitHub, Postman/Bruno collection, developers.factus.com.co |
| **Facturalatam** | Suscripción + docs | Cotizar | Comunidad (500+ devs en Slack) | 10K+ empresas, soporte WhatsApp/GitLab/ChatGPT bot |
| **Alanube** | BaaS volumen | Mínimo mensual acordado + por doc adicional | API-first, webhooks nativos | Multi-país (útil si se expande) |
| **Dataico** | Suscripción | Cotizar | Integra contabilidad + inventario + FE | Fuerte en Colombia, más ERP que API pura |
| **MATIAS API** | Anual | Desde $196.000 COP/año (cert. incluido) + $150 COP/doc extra | Multilenguaje (Node, PHP, Python, C#, Go) | Plan popular ~$265K/año |
| **Alegra** | Mensual | Desde $17.900 COP/mes (hasta 10M ingresos/mes) | Llave en mano, no solo API | Más para usuario final que para integrador |
| **Plemsi** | Proveedor directo DIAN | Cotizar | Autorizado DIAN directamente | |
| **SOENAC** | Open Source + servicio | Libre/cotizar | Repo público UBL 2.1 en GitHub | Bueno para entender protocolo, no necesariamente para producción |

### Precios reales encontrados
- **MATIAS API**: $196.000 COP/año con certificado digital incluido. Documentos extra: $150 COP/doc. Plan popular: ~$265.000 COP/año.
- **Alegra "Solo Facturación"**: $17.900 COP/mes para emprendedores (ingresos hasta $10M/mes). No es API pura, es UI con API.
- **Factus, Alanube, Facturalatam**: no publican precios — requieren cotización según volumen. Esto es estándar en B2B Colombia.

### Qué conviene para nuestro POS

**Recomendación: Factus como primera opción, Facturalatam como segunda.**

Razones:
- Factus tiene la mejor documentación para developers (crucial para integración rápida)
- OAuth2 estándar, SDK disponible en GitHub, Postman collection lista
- Facturalatam tiene la comunidad más activa si nos quedamos trabados
- Ambos abstraen completamente la complejidad DIAN
- MATIAS API es buena opción si el volumen es bajo y queremos precio fijo predecible

**Lo que NO hacer:**
- No integrar directo con DIAN (complejidad enorme, mantenimiento costoso)
- No usar Alegra si lo que necesitamos es API — es software de usuario final
- No usar SOENAC directo en producción sin soporte

---

## 4. Requisitos previos antes de escribir una línea de código

Estos pasos son administrativos y **toman tiempo real** — no técnicos:

### 4.1 Para el negocio que usará el POS (el facturador)
1. **RUT actualizado** — debe tener actividad económica correcta
2. **Registrarse en MUISCA** (portal DIAN) con NIT y contraseña
3. **Obtener Resolución de Facturación Electrónica** — acto administrativo que autoriza a emitir documentos electrónicos. Sin esto nada funciona legalmente.
4. **Vincular software** — registrar el POS en MUISCA como sistema autorizado
5. **Pasar set de pruebas** — DIAN exige aprobar N documentos de prueba antes de habilitar producción

### 4.2 Si queremos ser nosotros el proveedor tecnológico (habilitar POS-Vertical como PT)
Más complejo — requiere:
- RUT con objeto social explícito: "generación, transmisión, emisión, entrega y recepción de facturas electrónicas, notas débito, notas crédito y documentos electrónicos derivados"
- Ser facturador electrónico habilitado nosotros mismos primero
- Certificado digital propio de entidad acreditada ONAC
- Pasar set de pruebas con firma digital propia
- Garantizar infraestructura disponible + interoperabilidad
- Soporte técnico continuo a todos los clientes
- **Resolución 000165/2023 + Resolución 000202/2025 como marco regulatorio**

**Veredicto: para V1, NO ser proveedor tecnológico. Usar Factus o Facturalatam como PT habilitado. Ser PT es un negocio en sí mismo con obligaciones legales serias.**

---

## 5. Datos a capturar — campos obligatorios

### Para el Documento Equivalente Electrónico (lo que aplica a POS de mostrador)

#### Del negocio emisor (configuración del POS — se configura una vez)
```
- NIT del emisor
- Razón social / nombre
- Dirección del establecimiento
- Código CIIU de actividad económica
- Número de resolución DIAN de facturación
- Prefijo y rango de numeración autorizado
- Clave técnica de la resolución (para calcular CUDE)
```

#### Del comprador (por venta)
```
- Tipo de documento (CC, NIT, CE, Pasaporte, etc.) — opcional en mostrador
- Número de documento — opcional en mostrador
- Nombre / razón social — desde 2025: autocompletar vía API DIAN con cédula/NIT
- Correo electrónico — desde 2025: también autocompletar vía DIAN
- NOTA: para ventas de mostrador anónimas aplica "consumidor final"
```

#### De los items (por línea de venta)
```
- Descripción del bien/servicio
- Código del producto (estándar EAN o interno)
- Unidad de medida (DIAN tiene lista de códigos)
- Cantidad
- Precio unitario
- Descuento por ítem (si aplica)
- Código de departamento
- Tarifa de IVA aplicable (0%, 5%, 19%)
- Tarifa INC (impuesto consumo) si aplica
- Valor del impuesto calculado
```

#### Totales y formas de pago
```
- Subtotal sin impuestos
- Total IVA (desglosado por tarifa)
- Total INC (si aplica)
- Total descuentos
- Total del documento
- Forma de pago: efectivo / tarjeta débito / tarjeta crédito / transferencia / mixto
- Valor recibido (efectivo)
- Valor devuelto (cambio)
- Valor pagado por cada medio si es mixto
```

#### Metadatos del documento
```
- Número de documento (secuencial dentro de la resolución)
- Fecha y hora de emisión (UTC-5 Colombia)
- Tipo de documento: "Documento Equivalente Electrónico POS"
- Ambiente: 1=producción, 2=pruebas
- Software ID (asignado por DIAN al registrar el software)
- PIN del software
```

### Cambio obligatorio 2025/2026 — Autocompletado de comprador
Cuando el usuario ingresa la cédula o NIT del comprador, el sistema **debe** consultar la API de la DIAN para autocompletar nombre/razón social y correo. Esto lo proveen los proveedores tecnológicos como endpoint adicional, o se puede consumir directo del MUISCA.

---

## 6. Flujo de integración técnica — cómo conectar el POS

### Arquitectura recomendada

```
Frontend POS (Svelte/TS)
        ↓  HTTP POST
Backend Go (internal/sales)
        ↓  HTTP POST JSON (OAuth2)
Proveedor API (Factus/Facturalatam)
        ↓  XML UBL 2.1 + firma
DIAN MUISCA (validación)
        ↓  ApplicationResponse
Proveedor API
        ↓  Webhook POST a nuestro endpoint
Backend Go (handler /webhook/facturas)
        ↓  Actualiza estado en BD
Frontend POS (recibe confirmación, imprime si aplica)
```

### Endpoints típicos del proveedor (ejemplo Factus)

```
POST /api/v1/bills           → crear factura/doc equivalente
GET  /api/v1/bills/{id}      → consultar estado
GET  /api/v1/bills/{id}/pdf  → descargar representación gráfica
GET  /api/v1/bills/{id}/xml  → descargar XML DIAN
POST /api/v1/credit-notes    → nota crédito (devolución)
GET  /api/v1/numbering-ranges → consultar resolución activa
```

### Autenticación (OAuth2 — estándar en Factus y mayoría)
```go
// Obtener token
POST /oauth/token
Body: { client_id, client_secret, grant_type: "client_credentials" }
Response: { access_token, expires_in }

// Usar en cada request
Authorization: Bearer {access_token}
```

### Webhook — endpoint que debemos exponer
El proveedor hace POST a nuestra URL cuando DIAN responde:
```json
{
  "event": "bill.accepted",  // o bill.rejected
  "bill_id": "xxx",
  "cufe": "abc123...",
  "dian_response": "Documento aceptado",
  "pdf_url": "https://...",
  "xml_url": "https://..."
}
```

Debemos:
1. Exponer `POST /api/webhooks/facturas` en nuestro servidor
2. Verificar firma HMAC del webhook (cada proveedor tiene su método)
3. Actualizar estado de la venta en BD: `pendiente → aceptada / rechazada`
4. Si rechazada: guardar motivo, notificar al usuario del POS

### Estados de un documento
```
draft       → creado en POS, no enviado
pending     → enviado al proveedor, esperando DIAN
accepted    → DIAN aceptó ✓
rejected    → DIAN rechazó (ver motivo)
cancelled   → anulado con nota crédito
```

---

## 7. Consideraciones para nuestra arquitectura Go

### Modelo de datos — tabla `electronic_documents`
```sql
CREATE TABLE electronic_documents (
    id              TEXT PRIMARY KEY,
    sale_id         TEXT NOT NULL REFERENCES sales(id),
    type            TEXT NOT NULL, -- 'doc_equivalente', 'factura_venta', 'nota_credito'
    status          TEXT NOT NULL DEFAULT 'pending', -- draft/pending/accepted/rejected/cancelled
    provider        TEXT NOT NULL, -- 'factus', 'facturalatam', etc.
    provider_id     TEXT,          -- ID del doc en el proveedor
    cude            TEXT,          -- código único DIAN
    number          TEXT,          -- número de la resolución (ej: POS-000001)
    xml_url         TEXT,
    pdf_url         TEXT,
    dian_response   TEXT,
    rejection_reason TEXT,
    issued_at       DATETIME,
    accepted_at     DATETIME,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Configuración del negocio — tabla `fiscal_config`
```sql
CREATE TABLE fiscal_config (
    id                  INTEGER PRIMARY KEY DEFAULT 1,
    nit                 TEXT NOT NULL,
    razon_social        TEXT NOT NULL,
    direccion           TEXT NOT NULL,
    ciiu                TEXT NOT NULL,
    resolucion_dian     TEXT NOT NULL,  -- número de resolución
    prefijo             TEXT,           -- prefijo autorizado (puede ser vacío)
    rango_desde         INTEGER NOT NULL,
    rango_hasta         INTEGER NOT NULL,
    fecha_resolucion    DATE NOT NULL,
    fecha_vencimiento   DATE NOT NULL,
    clave_tecnica       TEXT NOT NULL,  -- para calcular CUDE
    ambiente            TEXT NOT NULL DEFAULT 'test', -- 'test' o 'production'
    software_id         TEXT NOT NULL,  -- asignado por DIAN
    software_pin        TEXT NOT NULL,
    provider            TEXT NOT NULL,  -- 'factus', etc.
    provider_client_id  TEXT NOT NULL,
    provider_secret     TEXT NOT NULL,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Package sugerido: `internal/factelect`
```
internal/factelect/
├── client.go        → cliente HTTP genérico contra el proveedor API
├── factus.go        → implementación específica Factus
├── mapper.go        → convierte Sale → DocumentoEquivalente (DTO del proveedor)
├── webhook.go       → handler del webhook entrante
├── types.go         → structs: DocumentEquivalente, BillRequest, BillResponse, etc.
└── config.go        → carga FiscalConfig desde BD
```

### Flujo en el handler de ventas
```go
// En sales handler, después de crear la venta:
go func() {
    doc, err := factelect.EmitFromSale(ctx, sale)
    if err != nil {
        log.Error("factelect emit failed", "sale_id", sale.ID, "err", err)
        // no bloquear la venta — guardar como 'draft' para reintento
        return
    }
    // guardar doc con status 'pending'
}()
// La venta se confirma al cajero inmediatamente
// La aceptación DIAN llega async por webhook
```

**Importante:** No bloquear la venta esperando respuesta DIAN. La emisión es async. El cajero confirma la venta; el documento se acepta segundos/minutos después.

---

## 8. Manejo de errores y edge cases

### Rechazos DIAN comunes
| Error | Causa | Acción |
|-------|-------|--------|
| Numeración fuera de rango | El número de doc excedió la resolución | Solicitar nueva resolución a DIAN urgente |
| Firma inválida | Certificado vencido o mal configurado | Proveedor debe renovar certificado |
| CUDE duplicado | Documento ya fue enviado | No reenviar, buscar el original |
| Datos del emisor incorrectos | NIT/razón social no coinciden con RUT | Actualizar fiscal_config |
| Tarifa IVA inválida | Producto con tarifa no existente | Corregir catálogo de productos |
| Formato fecha incorrecto | Zona horaria mal aplicada | Siempre usar UTC-5 (América/Bogotá) |

### Contingencia — DIAN o proveedor caído
La DIAN permite **modo contingencia**: emitir el documento cuando el servicio vuelva. El proveedor normalmente lo maneja internamente (encola y reintenta). Nuestro POS debe:
1. Permitir registrar la venta aunque FE falle (no bloquear el negocio)
2. Marcar documentos como `draft` / `pending_retry`
3. Tener job de reintento para documentos pendientes

### Anulación / devolución
- No existe "borrar" una factura electrónica aceptada por DIAN
- Devolución = emitir **Nota Crédito Electrónica** referenciando el CUDE original
- Descuento post-factura = también nota crédito
- En POS: el flujo de devolución debe disparar emisión de nota crédito, no borrar la venta

---

## 9. Pasos para implementar (orden sugerido)

### Fase 0 — Admin/legal (hacer esto ANTES de escribir código)
- [ ] Obtener resolución de facturación electrónica DIAN para el negocio de prueba
- [ ] Registrar software en MUISCA (obtener Software ID y PIN)
- [ ] Crear cuenta en Factus ambiente de pruebas (sandbox)
- [ ] Obtener client_id y client_secret del proveedor
- [ ] Tener certificado digital configurado por el proveedor

### Fase 1 — Infraestructura básica
- [ ] Crear tabla `fiscal_config` y migración
- [ ] Crear tabla `electronic_documents` y migración
- [ ] Implementar `internal/factelect/client.go` con OAuth2
- [ ] Implementar `internal/factelect/factus.go` (crear doc equivalente)
- [ ] Tests con sandbox de Factus

### Fase 2 — Integración con ventas
- [ ] Hook en `POST /api/sales` para disparar emisión async
- [ ] Implementar `POST /api/webhooks/facturas` para recibir estado DIAN
- [ ] Actualizar estado en `electronic_documents` al recibir webhook
- [ ] UI: mostrar estado FE en detalle de venta (pendiente/aceptada/rechazada)

### Fase 3 — Documentos y UI
- [ ] Endpoint `GET /api/sales/{id}/factura/pdf` → proxy al PDF del proveedor
- [ ] Endpoint `GET /api/sales/{id}/factura/xml` → proxy al XML
- [ ] Botón "Imprimir/Enviar factura" en frontend
- [ ] Campo email en venta para envío automático al comprador

### Fase 4 — Robustez
- [ ] Job de reintento para docs en `pending_retry` (cron cada 5 min)
- [ ] Alertas cuando resolución de facturación tiene < 10% de numeración restante
- [ ] Flujo de nota crédito en devoluciones
- [ ] Autocompletado de comprador con cédula/NIT (Resolución 2025)

---

## 10. Links y recursos

| Recurso | URL |
|---------|-----|
| DIAN — Sistema Factura Electrónica | micrositios.dian.gov.co/sistema-de-facturacion-electronica |
| DIAN — Resolución 165 Anexo Técnico | dian.gov.co (búsqueda en documentos normativos) |
| Factus Developers | developers.factus.com.co |
| SOENAC Open Source UBL 2.1 | github.com/soenac/api-dian |
| SDK Factus (Go/otros) | github.com/juacosoft/FactusDian-SDK |
| Facturalatam comunidad Slack | facturalatam.com/api (link al Slack) |
| Alanube Developer Docs | developer.alanube.co |

---

## 11. Decisión recomendada para POS Vertical V1

**Usar Factus como proveedor tecnológico.**

| Criterio | Por qué Factus |
|----------|---------------|
| DX (Developer Experience) | Mejor documentación del mercado, OAuth2 estándar |
| Velocidad de integración | Postman/Bruno collections listas, SDK en GitHub |
| Soporte | Comunidad activa, respuesta rápida |
| Flexibilidad | Soporta todos los tipos de documento DIAN |
| Escalabilidad | Funciona para POS, ERP, PMS — no nos limita |
| Precio | Cotizar según volumen esperado de facturas/mes |

**Si Factus no nos da términos aceptables → Facturalatam (segunda opción sólida).**

No implementar como Proveedor Tecnológico DIAN en V1. Las obligaciones legales y técnicas son un proyecto separado que requiere meses y recursos dedicados.

---

## 12. Gaps y riesgos identificados (análisis pre-implementación)

> Resultado del análisis de la arquitectura actual del POS vs. los requisitos DIAN.
> Clasificación: 🔴 Bloqueante | 🟡 Importante antes de ship | 🟢 Nice-to-have

### 🔴 Bloqueantes

**Gap 1 — `business_config` no tiene CIIU**
El CIIU (código de actividad económica) es obligatorio en cada DEE. La tabla `fiscal_config` sí lo tiene, pero la tabla existente `business_config` no. El problema: algunos datos del emisor (NIT, razón social, dirección) ya viven en `business_config` y se usan para el ticket impreso. Si no se agrega `ciiu` allá también, el mapper va a tener que consultar dos tablas distintas para armar el emisor, o habrá datos duplicados entre `business_config` y `fiscal_config` que pueden desincronizarse.

**Decisión requerida:** ¿`fiscal_config` reemplaza a `business_config` o la extiende? Recomendado: `fiscal_config` es una extensión fiscal de `business_config`. El `ciiu` va en `business_config` vía migration porque es dato del negocio, no solo fiscal.

**Gap 2 — `Sale.payment_method` es un solo string**
El DEE requiere desglose de formas de pago si el pago es mixto (efectivo + tarjeta). El modelo actual `Sale` tiene `payment_method TEXT` con valores `cash|transfer|card|credit`. No soporta "pagué $50.000 en efectivo y $30.000 con tarjeta". Dos opciones:
- **Opción A (MVP simple):** Bloquear ventas mixtas en la UI. Solo un método por venta. El DEE mapea directo.
- **Opción B (correcto):** Agregar tabla `sale_payment_methods` con (sale_id, method, amount). Cambio mayor al modelo y a la UI del cajero.

Para V1 con el taller, los pagos mixtos son frecuentes (empresa paga parte con transferencia, parte en efectivo). Si el taller es el piloto, la Opción A puede no alcanzar.

**Gap 3 — Webhook en POS offline-first**
El POS corre en `localhost:8080` en la máquina del negocio. Factus necesita hacer POST al webhook desde internet. Esto es un problema estructural: un servidor local no recibe webhooks externos.

**Soluciones posibles:**
1. **Polling activo (recomendado para V1):** En lugar de webhook, el job de reintento también hace `GET /api/v1/bills/{id}` a Factus cada 30 segundos hasta obtener estado final. Sin depender de URL pública. Agrega latencia pero funciona offline-first.
2. **Webhook + ngrok/túnel:** Solo para dev/demos. No es opción de producción.
3. **Webhook si el negocio tiene IP pública:** Algunos talleres con internet fijo SÍ tienen IP pública. Configurar webhook URL en `fiscal_config` como campo opcional. Si está vacío → polling.

**Decisión requerida:** Polling vs webhook. Afecta el diseño de `webhook.go` y el job de retry.

**Gap 4 — Void → Nota Crédito sin flujo diseñado**
El sistema tiene `DELETE /api/sales/{id}/void`. Si la venta ya tiene un DEE `accepted`, anularla requiere emitir una Nota Crédito Electrónica en DIAN referenciando el CUDE original. Si se hace void sin nota crédito, el negocio queda en incumplimiento: DIAN tiene el documento aceptado, pero el negocio lo trata como anulado sin documentarlo.

**Flujo requerido:**
```
Admin hace void de venta
  → ¿Tiene electronic_document con status='accepted'?
    → SÍ: bloquear void directo. Mostrar modal:
          "Esta venta tiene factura electrónica aceptada.
           Para anularla debes emitir una Nota Crédito."
          Botón: "Emitir Nota Crédito y Anular"
          → factelect.CreateCreditNote(cude_original, motive)
          → Si Factus acepta nota crédito → proceder con void
          → Si Factus rechaza → no anular, mostrar error
    → NO (status=draft/rejected/sin documento): void normal
```

---

### 🟡 Importantes

**Gap 5 — `fiscal_config` con un solo rango de numeración**
La tabla guarda `rango_desde` y `rango_hasta` como columnas fijas. No hay:
- Contador del número actual (`current_number`) — ¿cómo sabe el mapper qué número asignar al próximo DEE? Factus asigna el número, pero si el rango se agota, hay que solicitar uno nuevo a DIAN.
- Validación de que `fecha_vencimiento` no haya pasado — si la resolución venció, DIAN rechaza todo.
- Alerta cuando queda <10% del rango — sin esto, el dueño no sabe que está a punto de quedarse sin numeración.

**Para MVP:** Agregar al menos validación de fecha de vencimiento antes de emitir. El contador lo maneja Factus internamente; solo necesitamos saber cuándo el rango se agota consultando `GET /api/v1/numbering-ranges`.

**Gap 6 — `clave_tecnica` en `fiscal_config`**
La `clave_tecnica` se necesita para calcular el CUDE (SHA-384). Pero si Factus calcula el CUDE, ¿necesitamos enviarla en el payload de la API? Confirmar con la documentación de Factus si este campo va en el `BillRequest` o si Factus lo obtiene del registro de la resolución en su sistema. Si Factus no la necesita de nosotros: no la incluir en el `BillRequest` (pero sí guardarla por auditoría).

**Gap 7 — Email del comprador en ventas anónimas**
La tabla `customers` tendrá `email`, pero en ventas de mostrador sin customer (consumidor final), el comprador puede dar su email para recibir la factura. Ese email es capturado en el momento de la venta, no existe en ningún registro previo. El `BillRequest` necesita un campo `buyer_email` que puede venir de:
- `customers.email` si hay customer_id en la venta
- Un campo ad-hoc en el request de la venta si es consumidor final

La UI del cajero necesita un campo opcional "Email para factura" en la pantalla de cobro.

**Gap 8 — Unidad de medida DIAN en productos**
El DEE requiere unidad de medida con el código estándar DIAN (lista de ~60 valores: `94`=unidad, `KGM`=kilogramo, `MTR`=metro, etc.). El campo `unit` en la tabla `products` es texto libre (`und`, `kg`, `caja`, etc.). El mapper tiene que traducir texto libre a código DIAN. Si no hay mapeo, Factus puede rechazar.

**Solución para MVP:** Tabla de mapeo hardcodeada en `mapper.go` para los valores más comunes. Valores desconocidos → `94` (unidad) por defecto.

---

### 🟢 Nice-to-have (post-MVP)

**Gap 9 — Autocompletado del comprador (Resolución 2025)**
Al ingresar NIT/cédula en la UI, consultar endpoint de Factus (o DIAN directo) para autocompletar nombre, razón social y email. Reduce errores de digitación y cumple con la Resolución 000202/2025. Factus probablemente tiene un endpoint `GET /taxpayer/{document}` para esto.

**Gap 10 — INC (Impuesto al Consumo)**
Algunos productos (cervezas, licores, comidas preparadas) tienen INC además de IVA. El modelo `SaleItem` solo tiene `tax_rate` (IVA). El DEE puede incluir INC como tarifa adicional. Para un taller: no aplica. Para tiendas de abarrotes: puede aplicar en algunos SKUs. Agregar `inc_rate` y `inc_amount` a `sale_items` en V2 si el cliente lo necesita.

---

## 13. Cambios necesarios al modelo de datos existente (migrations)

Además de crear `fiscal_config` y `electronic_documents`, hay cambios a tablas existentes:

### Migration v8 — tablas nuevas + alteraciones

```sql
-- 1. CIIU en business_config (dato del negocio, no solo fiscal)
ALTER TABLE business_config ADD COLUMN ciiu TEXT;

-- 2. Datos tributarios en customers (para DEE con identificación del comprador)
ALTER TABLE customers ADD COLUMN email TEXT;
ALTER TABLE customers ADD COLUMN document_type TEXT DEFAULT 'CC';
-- CC | NIT | CE | TI | PASAPORTE | DIE | NIT_EXTRANJERO
ALTER TABLE customers ADD COLUMN document_number TEXT;
ALTER TABLE customers ADD COLUMN tax_regime TEXT DEFAULT 'simplified';
-- simplified = régimen simple / régimen común sin IVA
-- regular    = régimen ordinario (declara IVA)

-- 3. fiscal_config (nueva)
CREATE TABLE IF NOT EXISTS fiscal_config (
    id                  INTEGER PRIMARY KEY CHECK (id = 1),
    enabled             INTEGER NOT NULL DEFAULT 0,
    nit                 TEXT NOT NULL DEFAULT '',
    razon_social        TEXT NOT NULL DEFAULT '',
    direccion           TEXT NOT NULL DEFAULT '',
    ciiu                TEXT NOT NULL DEFAULT '',
    resolucion_dian     TEXT NOT NULL DEFAULT '',
    prefijo             TEXT DEFAULT '',
    rango_desde         INTEGER NOT NULL DEFAULT 0,
    rango_hasta         INTEGER NOT NULL DEFAULT 0,
    fecha_resolucion    DATE,
    fecha_vencimiento   DATE,
    clave_tecnica       TEXT NOT NULL DEFAULT '',
    ambiente            TEXT NOT NULL DEFAULT 'test',
    software_id         TEXT NOT NULL DEFAULT '',
    software_pin        TEXT NOT NULL DEFAULT '',
    provider            TEXT NOT NULL DEFAULT 'factus',
    provider_client_id  TEXT NOT NULL DEFAULT '',
    provider_secret     TEXT NOT NULL DEFAULT '',
    webhook_url         TEXT,  -- URL pública para webhook (NULL = usar polling)
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO fiscal_config (id) VALUES (1);

-- 4. electronic_documents (nueva)
CREATE TABLE IF NOT EXISTS electronic_documents (
    id               TEXT PRIMARY KEY,         -- UUID v4 generado localmente
    sale_id          INTEGER NOT NULL REFERENCES sales(id),
    type             TEXT NOT NULL DEFAULT 'doc_equivalente',
    -- doc_equivalente | factura_venta | nota_credito
    status           TEXT NOT NULL DEFAULT 'draft',
    -- draft | pending | accepted | rejected | cancelled
    provider         TEXT NOT NULL DEFAULT 'factus',
    provider_id      TEXT,                     -- ID asignado por el proveedor
    cude             TEXT,                     -- código único DIAN (asignado al aceptar)
    number           TEXT,                     -- número completo: 'POS-000001'
    xml_url          TEXT,
    pdf_url          TEXT,
    dian_response    TEXT,                     -- respuesta cruda de DIAN (JSON)
    rejection_reason TEXT,
    retry_count      INTEGER NOT NULL DEFAULT 0,
    issued_at        DATETIME,
    accepted_at      DATETIME,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_eledocs_sale_id ON electronic_documents(sale_id);
CREATE INDEX IF NOT EXISTS idx_eledocs_status ON electronic_documents(status);
CREATE INDEX IF NOT EXISTS idx_eledocs_provider_id ON electronic_documents(provider_id);
```

---

## 14. Nuevos endpoints API

```
-- Configuración fiscal
GET    /api/fiscal/config              Admin: ver fiscal_config
PUT    /api/fiscal/config              Admin: actualizar fiscal_config
GET    /api/fiscal/config/validate     Admin: verificar credenciales Factus (ping)

-- Documentos electrónicos
GET    /api/sales/{id}/factura         Estado del DEE de una venta
POST   /api/sales/{id}/factura/emit    Admin: emitir/reemitir manualmente
GET    /api/sales/{id}/factura/pdf     Proxy PDF → Factus (descarga o iframe)
GET    /api/sales/{id}/factura/xml     Proxy XML → Factus

-- Administración
GET    /api/facturas                   Admin: listar todos los documentos (filtros: status, fecha)
POST   /api/facturas/{id}/retry        Admin: reintentar envío fallido
GET    /api/facturas/stats             Admin: resumen (pendientes, aceptados, rechazados)

-- Webhook (sin autenticación JWT — verificar HMAC de Factus)
POST   /api/webhooks/facturas          Factus notifica resultado DIAN
```

---

## 15. Flujo void → Nota Crédito (diseño completo)

```
Admin → DELETE /api/sales/{id}/void
  │
  ├─ ¿Existe electronic_document para esta venta?
  │     NO → void normal (comportamiento actual)
  │
  └─ SÍ → ¿status del documento?
        ├─ draft / rejected → void normal (el doc no fue aceptado por DIAN)
        │
        └─ accepted / pending →
              Retornar 409 con body:
              {
                "error": "sale_has_accepted_invoice",
                "message": "Esta venta tiene un documento electrónico aceptado por DIAN. Para anularla debes emitir una Nota Crédito.",
                "document_id": "...",
                "cude": "...",
                "requires_credit_note": true
              }

Admin → POST /api/sales/{id}/void-with-credit-note
  Body: { "motive": "DEVOLUCIÓN_DINERO" }  -- códigos DIAN
  │
  ├─ factelect.CreateCreditNote(cude_original, motive, sale_data)
  │     → POST Factus /credit-notes
  │     → Si Factus OK:
  │           - Guardar nota crédito en electronic_documents (type='nota_credito')
  │           - Proceder con void normal de la venta
  │           - Retornar 200
  │     → Si Factus falla:
  │           - NO anular la venta
  │           - Retornar 502 con error de Factus
  │           - Guardar intento en log para seguimiento manual
```

**Códigos de motivo DIAN para nota crédito:**
- `DEVOLUCIÓN_PARCIAL` — devolución parcial de bienes
- `DEVOLUCIÓN_DINERO` — anulación total con reintegro
- `REBAJA_DESCUENTO` — descuento post-factura
- `AJUSTE_PRECIO` — ajuste de precio

---

## 16. Diseño del adaptador para vertical Taller

El paquete `internal/factelect` define una interfaz que cualquier vertical implementa:

```go
// adapter.go — interface pública del paquete
type SaleAdapter interface {
    // Convierte una venta (de cualquier vertical) a un BillRequest para Factus
    ToBillRequest(ctx context.Context, saleID int64) (*BillRequest, error)
}

// BillRequest — estructura genérica que le enviamos a Factus
type BillRequest struct {
    Issuer      IssuerInfo
    Buyer       BuyerInfo        // puede ser "consumidor final"
    Items       []BillItem
    Totals      BillTotals
    PaymentMethods []PaymentMethod
    IssuedAt    time.Time
    Notes       string
    Extra       map[string]string // datos adicionales por vertical (placa, km, etc.)
}
```

**Implementación para Pos-vertical** (`internal/factelect/pos_adapter.go`):
```go
type PosSaleAdapter struct {
    salesSvc    *service.SaleService
    productsSvc *service.ProductService
    setupRepo   repository.SetupRepository
    fiscalRepo  FiscalConfigRepository
}

func (a *PosSaleAdapter) ToBillRequest(ctx context.Context, saleID int64) (*BillRequest, error) {
    sale, _ := a.salesSvc.GetByID(ctx, saleID)
    config, _ := a.fiscalRepo.Get(ctx)
    // Mapea Sale.Items → BillItems
    // Sale.PaymentMethod → PaymentMethods (único método en V1)
    // Sale.CustomerID → BuyerInfo (o consumidor final si nil)
    // BusinessConfig.NIT + fiscal_config.CIIU → IssuerInfo
}
```

**Implementación para Taller** (`taller-repo/internal/factelect/taller_adapter.go`):
```go
type TallerSaleAdapter struct {
    ordenRepo  *taller.OrdenRepository  // órdenes de trabajo del taller
    fiscalRepo FiscalConfigRepository
}

func (a *TallerSaleAdapter) ToBillRequest(ctx context.Context, ordenID int64) (*BillRequest, error) {
    orden, _ := a.ordenRepo.GetByID(ctx, ordenID)
    // Items de la orden:
    //   repuestos → BillItem{ProductType: "01", ...}
    //   mano de obra → BillItem{ProductType: "04", ...}
    // Datos del vehículo van en Extra:
    //   Extra["placa"] = orden.Placa
    //   Extra["km"] = orden.KM
    //   Extra["marca"] = orden.Marca
    // El cliente del taller normalmente tiene NIT (B2B frecuente)
}
```

**Para que el taller use el paquete:**
- **Opción V1:** `replace` directive en `go.mod` del taller apuntando al path local de Pos-vertical.
- **Opción V2:** Extraer `internal/factelect` a un repositorio separado `github.com/tu-org/factelect-go` con su propio `go.mod`. Ambos repos lo importan como dependencia.

---

## 17. Decisiones pendientes (bloquean codificación)

| # | Decisión | Opciones | Impacto |
|---|----------|---------|---------|
| 1 | **Webhook vs polling** | Polling (recomendado offline-first) / Webhook con URL configurable | Cambia `webhook.go` completo y el job de retry |
| 2 | **Pagos mixtos en MVP** | Bloquear en UI (simple) / Tabla `sale_payment_methods` (correcto) | Si taller es piloto, los mixtos son frecuentes |
| 3 | **Modelo de datos del taller** | Ver structs del repo taller | Sin esto el `TallerSaleAdapter` no se puede diseñar |
| 4 | **`fiscal_config` vs extender `business_config`** | Tabla separada (ya decidido) / Añadir campos a `business_config` | Afecta cuántas tablas consulta el mapper |
| 5 | **`clave_tecnica` en BillRequest** | Confirmar con docs Factus | Si no hace falta, no la enviar (reducir superficie sensible) |

---

## 18. Test cases que deben pasar antes de producción

### Happy path
- [ ] Venta en efectivo, consumidor final → DEE emitido → Factus acepta → `status=accepted`, `cude` guardado
- [ ] Venta con cliente identificado (NIT) → DEE con datos del comprador → aceptado
- [ ] Venta con cliente CC (persona natural) → DEE con CC → aceptado

### Contingencia / offline
- [ ] Factus no responde → DEE queda `draft` → venta sigue registrada → cajero no bloqueado
- [ ] Factus vuelve → job retry corre → DEE pasa a `pending` → luego `accepted`
- [ ] Factus rechaza → `status=rejected`, `rejection_reason` guardado → admin ve el error

### Validaciones pre-emisión
- [ ] `fecha_vencimiento` de resolución pasada → emisión bloqueada con mensaje claro antes de llamar a Factus
- [ ] `fiscal_config.enabled = false` → no se intenta emitir nada (modo sin FE)
- [ ] Producto con `tax_rate` no válida DIAN (ej. 15%) → Factus rechaza → `rejected` con motivo específico

### Anulaciones
- [ ] Void de venta sin DEE → void normal sin nota crédito
- [ ] Void de venta con DEE `draft` → void normal (el doc no llegó a DIAN)
- [ ] Void de venta con DEE `accepted` → API retorna 409 con `requires_credit_note: true`
- [ ] Void-with-credit-note → nota crédito aceptada → venta anulada
- [ ] Void-with-credit-note → Factus rechaza nota crédito → venta NO anulada, error visible

### Taller (cuando el adaptador esté listo)
- [ ] Orden con repuestos + mano de obra → DEE con dos tipos de ítem → aceptado
- [ ] Orden con cliente empresa (NIT) → DEE B2B → aceptado
- [ ] Campos extra (placa, km) aparecen en PDF del DEE
