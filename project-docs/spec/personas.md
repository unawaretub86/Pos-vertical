# Personas

> **Tipo:** spec · **Vigente:** sí · **Manda sobre:** para quién se escribe cada US
> **Actualizado:** 2026-09-21 · Origen: `PRD.md:29-38`, verificado contra los roles del sistema.

Toda US se escribe como *"Como Don Carlos (dueño)…"* o *"Como Laura (cajera)…"*. Si una US no se puede atribuir a ninguno de los dos, probablemente es trabajo técnico y va como *"Como desarrollador…"*.

---

## Don Carlos — dueño

Distribuidora de alimentos pequeña, en Bogotá. 45–60 años. Manejo básico del computador: usa WhatsApp, Excel de vez en cuando.

Vende fiado a dueños de restaurantes y lleva las deudas en un cuaderno.

**Qué necesita:** vender simple, saber quién le debe, imprimir recibos.

**Qué lo hace desconfiar del sistema:** un número que no cuadra. Un cuadre de caja con una diferencia sin explicación, o dos pantallas mostrando cifras distintas del mismo dato, y vuelve al cuaderno.

**Rol técnico:** `admin`. Ve dashboard, reportes, ganancias y cartera.

## Laura — cajera

Atiende el mostrador, 50–150 ventas por día. 20–35 años, cómoda con el celular y apps básicas.

**Qué necesita:** búsqueda rápida de producto, cobro rápido, sin menús complicados.

**Qué la frustra:** cualquier cosa que le quite las manos del teclado en hora pico, y cualquier paso extra entre escanear y cobrar.

**Rol técnico:** `cashier`. **No ve** dashboard, reportes, ganancias ni cartera — están bajo `RequireAdmin`.

---

## Regla de permisos

```
roles del sistema (users.role): admin · cashier
```

Fuente: `internal/database/migrate.go:156`.

**Todo lo que exponga ventas totales, ganancias, margen o cartera va bajo `appmw.RequireAdmin`.** No por burocracia: Laura es empleada, y los márgenes y las deudas de los clientes son información del dueño.

Cuando se agregue una pantalla o un endpoint nuevo, la pregunta es siempre: *¿Laura debería poder ver esto?* Si la respuesta es no, va en un grupo admin.
