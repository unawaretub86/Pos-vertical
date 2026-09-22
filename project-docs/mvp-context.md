# POS Genérico — MVP Context

> **Tipo:** histórico · **Vigente:** parcial · **Fecha:** era MVP
> La visión y las promesas del producto se extrajeron a [spec/product.md](spec/product.md), que es el documento vigente.


## Visión

POS local, offline-first, distribuible como un solo ejecutable. Genérico para cualquier negocio que venda productos físicos o servicios. Modelo híbrido: funciona 100% local, con sync a la nube opcional en el futuro.

**Pitch:** "Un POS que funciona sin internet, maneja el fiado de sus clientes, y no le cobra mensualidad."

---

## Mercado Target

Negocios pequeños/medianos en Colombia:

- Distribuidoras de insumos de comida rápida
- Bicicleterías
- Supermercados / tiendas de barrio
- Cualquier negocio con flujo: comprar inventario → almacenar → vender

---

## Modelo de Negocio

| Concepto              | Detalle                                        |
| --------------------- | ---------------------------------------------- |
| Licencia              | Pago único por instalación (~$200-500 USD)     |
| Soporte mensual       | Opcional (~$15 USD/mes)                        |
| Módulo cloud (futuro) | Suscripción para sync, reportes remotos, multi-sucursal |

### Costos de desarrollo

- Infraestructura MVP: ~$0-12 USD (dominio para landing)
- Landing page: Vercel (gratis)
- Sin costos de hosting para el cliente

### ROI estimado

- 5 clientes × $300 = $1,500 USD
- 10 clientes × $300 + soporte $15/mes = $3,000 + $150/mes recurrente

---

## Tech Stack

| Capa               | Tecnología                                | Razón                                            |
| ------------------- | ----------------------------------------- | ------------------------------------------------ |
| Backend             | Go + Chi router                           | Dominio del dev, binario compilado, distribuible  |
| Frontend            | Next.js 14 + TypeScript + Tailwind + shadcn/ui | Dominio del dev, UI profesional rápida        |
| Base de datos       | SQLite (modernc.org/sqlite, pure Go)      | Zero config, un archivo, ideal para local         |
| Distribución        | Binario Go que sirve frontend estático    | Un solo .exe, sin dependencias                    |
| Impresión tickets   | ESC/POS protocol                          | Estándar para impresoras térmicas                 |

### Arquitectura de distribución

```
pos.exe (binario Go)
├── Sirve API REST (localhost:8080)
├── Sirve frontend Next.js (exportado como static)
├── SQLite embebido (data.db)
└── El cliente abre el navegador en localhost:8080
```

### Arquitectura interna (preparada para híbrido)

```
Handler → Service → Repository (interface)
                         ↓
                    SQLite (hoy)
                    SQLite + API sync (futuro)
```

---

## Consideraciones Genérico

### Unidades de medida flexibles

- Venta por unidad (1 llanta, 1 gaseosa)
- Venta por peso (2.5 kg de queso)
- Conversión caja→unidad queda para v2

### Productos vs Servicios

- Campo `tipo: producto | servicio`
- Servicio no descuenta stock (ej: reparación bicicleta)

### Precios flexibles

- Precio base editable en punto de venta (con permiso)
- Precio mayorista opcional por producto
- Listas de precio complejas en v2

### Clientes y crédito (fiado)

- Registro básico de clientes
- Ventas a crédito (cuenta por cobrar)
- Abonos a deuda
- Vista de cartera — **diferenciador principal**

### Consideraciones Colombia

- Moneda: COP, sin decimales
- IVA: manejo de impuesto básico en productos
- Métodos de pago: efectivo (con cambio), transferencia (Nequi/Daviplata), tarjeta (datáfono externo)
- Facturación electrónica DIAN: NO en MVP

---

## Funcionalidades MVP

### Prioridad 1 — Sin esto no hay POS

**Configuración inicial (primer uso):**

- Nombre del negocio, logo, dirección, NIT/CC
- Moneda (COP por defecto)
- Formato de ticket

**Productos:**

- CRUD: nombre, categoría, precio venta, precio mayorista (opcional), costo, tipo (producto/servicio), unidad de medida (unidad/kg/lb/litro), stock mínimo
- Categorías personalizables
- Búsqueda rápida por nombre o código

**Punto de venta:**

- Carrito de venta
- Buscar/agregar productos
- Modificar precio en línea (con permiso)
- Descuento por ítem o global
- Métodos de pago: efectivo (cálculo de cambio), transferencia, tarjeta, fiado
- Asociar cliente a la venta (opcional)
- Impresión de ticket

### Prioridad 2 — Control de dinero

**Caja:**

- Apertura con monto base
- Cierre con cuadre (esperado vs real)
- Historial de movimientos (entradas, salidas, ventas)

### Prioridad 3 — Diferenciador

**Clientes y crédito:**

- CRUD clientes (nombre, teléfono, dirección)
- Registro de ventas a crédito
- Abonos a deuda
- Vista de cartera (quién debe, cuánto, desde cuándo)

### Prioridad 4 — Control de stock

**Inventario:**

- Stock actual por producto
- Alertas de stock bajo
- Entrada manual de inventario (cuando llega mercancía)

### Prioridad 5 — Visibilidad del negocio

**Reportes:**

- Ventas del día/semana/mes
- Productos más vendidos
- Cartera (deudas pendientes)
- Movimientos de caja

### Prioridad 6 — Profesionalización

**Impresión de tickets:**

- Ticket de venta con datos del negocio
- Formato configurable

### Prioridad 7 — Multi-usuario

**Usuarios:**

- Roles: admin y cajero
- Admin: todo acceso
- Cajero: solo vender, ver su caja

---

## Fuera del MVP

| Feature                           | Razón                                      | Cuándo |
| --------------------------------- | ------------------------------------------ | ------ |
| Facturación electrónica DIAN      | Proyecto complejo aparte                   | v3+    |
| Multi-sucursal                    | Requiere sync/cloud                        | v2     |
| Código de barras (lector)         | Nice to have, no blocker                   | v2     |
| Compras/proveedores               | Entrada manual de stock es suficiente       | v2     |
| Listas de precio múltiples        | 2 precios (normal/mayorista) es suficiente  | v2     |
| Conversión unidades (caja→unidad) | Complejidad innecesaria para MVP            | v2     |
| Sync a la nube                    | MVP es local puro                          | v2     |
| App móvil                         | Web responsive es suficiente                | v3+    |

---

## Modelo Híbrido — Roadmap

### Fase 1 (MVP): 100% Local

Cliente ejecuta `pos.exe` → abre navegador → trabaja offline. SQLite local, todo vive en su PC.

### Fase 2: Componente cloud opcional

- Sync de datos a servidor central
- Reportes accesibles desde celular
- Backup automático en la nube
- Multi-sucursal (sucursales sincronizan a un central)
- **Esto es lo que se monetiza como suscripción mensual**

---

## Competencia en Colombia

| Competencia (Siigo, Alegra, Vendty) | Este POS                              |
| ----------------------------------- | ------------------------------------- |
| Requieren internet                  | Funciona offline                      |
| Cobran $30-80 USD/mes               | Licencia única o bajo costo           |
| Genéricos pero complicados          | Simple, enfocado en lo esencial       |
| No manejan fiado bien               | Fiado/cartera integrado               |
| Setup complejo                      | Un exe, abrir navegador, listo        |
