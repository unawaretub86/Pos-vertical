# Producto — Pos-vertical

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** qué es el producto y qué no
> **Actualizado:** 2026-09-21

## Qué es

Un POS local que funciona sin internet, maneja el fiado de los clientes, y no cobra mensualidad.

Se distribuye como **un solo ejecutable** que el tendero copia a su computador y abre. Sin instalador complicado, sin cuenta en la nube, sin configuración de servidor.

## Para quién

Negocios pequeños y medianos en Colombia: tiendas de barrio, distribuidoras de alimentos, ferreterías, cafeterías. Ver [personas.md](personas.md).

El parque de hardware es **4GB de RAM como caso común**, 8GB es "bueno", muchos con disco mecánico y CPU sin GPU. Cualquier decisión técnica se mide contra esa máquina, no contra la del desarrollador.

## Las tres promesas

Son el producto. Romper cualquiera de las tres es cambiar de producto, y eso lo decide Andrey.

### 1. Funciona sin internet

No hay llamada a un servicio externo en el camino crítico de vender, cobrar, fiar o cerrar caja. Se desconecta el cable y todo sigue igual.

La única excepción aceptada es la **facturación electrónica DIAN**, que por ley requiere un proveedor autorizado — y está diseñada para no bloquear la venta cuando falla.

### 2. Maneja el fiado de verdad

Es el diferenciador. Ningún competidor —local ni mundial— lo hace bien. Ver [domain/fiado.md](domain/fiado.md).

### 3. No cobra mensualidad

Ninguna feature puede introducir un costo recurrente para el cliente. Eso descarta APIs de pago, envío automático por WhatsApp Business y hosting en nube como parte del producto base.

## Qué NO es

- **No es un ERP.** No hay contabilidad, nómina ni activos fijos.
- **No es multi-sede en la nube.** Multi-caja por LAN local está en roadmap (`R-18`); replicación entre locales no.
- **No es una pasarela de pagos.** Nequi y transferencia se registran como método de pago; no se integra con la API de nadie.
- **No presta plata.** El capital embebido (`R-41`) requiere partner fintech regulado y no se persigue en solitario.

## Cómo se decide qué construir

Orden de prioridad cuando hay que elegir:

1. **Lo que el primer cliente necesita para dejar el cuadernillo.** Si no bloquea eso, no es urgente.
2. **Lo que profundiza el fiado.** Es donde la competencia no llega.
3. **Lo que cierra brecha obvia con los locales** (Alegra, Siigo, Tiendatek, Vendty). Que no falte lo que todos dan por sentado.
4. **Lo que se ve bien en demo** — importa, pero nunca antes que 1 y 2.

Lo que viene está en [../roadmap.md](../roadmap.md), que es la única fuente.
