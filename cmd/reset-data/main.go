// reset-data clears the dry-run transactions and keeps master data, once,
// right before go-live (US-093). Dry-run by default; --apply deletes.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/andreyo/pos/internal/resetdata"
)

// resolvePort mirrors cmd/pos: --port, then POS_PORT, then 8080.
func resolvePort(flagVal, envVal string) int {
	for _, v := range []string{flagVal, envVal} {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 1 && n <= 65535 {
			return n
		}
	}
	return 8080
}

// printOrder is the display order of the per-table counts.
var printOrder = []string{
	"sales", "sale_items", "sale_payments", "customer_payments", "registers",
	"register_movements", "expenses", "orders", "order_items",
	"electronic_documents", "assistant_queries", "audit_log",
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "Error: %v\n", err)
	return resetdata.ExitCode(err)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reset-data", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "data.db", "ruta de la base de datos")
	portFlag := fs.String("port", "", "puerto del POS (por defecto 8080, o POS_PORT)")
	apply := fs.Bool("apply", false, "borrar de verdad (sin esto solo se muestra qué se borraría)")
	confirm := fs.String("confirm", "", `texto de confirmación, p. ej. "BORRAR 47" (para scripts)`)
	force := fs.Bool("force", false, "permitir repetir el reinicio si ya se ejecutó antes")
	discard := fs.Bool("discard-fiscal-documents", false, "borrar también documentos electrónicos aceptados o en curso (irreversible)")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	port := resolvePort(*portFlag, os.Getenv("POS_PORT"))
	ctx := context.Background()
	opts := resetdata.Options{Force: *force, DiscardFiscal: *discard}

	db, err := resetdata.Open(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer db.Close()

	if err := resetdata.CheckSchemaVersion(ctx, db); err != nil {
		return fail(stderr, err)
	}
	guardErr := resetdata.CheckGuards(ctx, db, opts)
	if guardErr != nil && *apply {
		return fail(stderr, guardErr)
	}
	plan, err := resetdata.Plan(ctx, db)
	if err != nil {
		return fail(stderr, err)
	}

	var shop string
	// The shop name is informational; an empty name is not an error.
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(name, '') FROM business_config WHERE id = 1`).Scan(&shop)

	fmt.Fprintf(stdout, "Base de datos: %s\n", *dbPath)
	if shop != "" {
		fmt.Fprintf(stdout, "Negocio: %s\n", shop)
	}
	fmt.Fprintln(stdout, "Se borrarán:")
	for _, t := range printOrder {
		if t == "audit_log" {
			fmt.Fprintf(stdout, "  %s %d (solo los registros que apuntan a filas borradas)\n", t, plan.Counts[t])
			continue
		}
		fmt.Fprintf(stdout, "  %s %d\n", t, plan.Counts[t])
	}
	printStock(stdout, plan)

	if !*apply {
		if guardErr != nil {
			fmt.Fprintf(stdout, "\nAVISO: con --apply esto se rechazaría: %v\n", guardErr)
		}
		fmt.Fprintln(stdout, "\nSimulación: no se cambió nada. Use --apply para borrar.")
		return 0
	}

	if err := resetdata.CheckPort(port); err != nil {
		return fail(stderr, err)
	}

	sales := plan.Counts["sales"]
	var in io.Reader = stdin
	if *confirm != "" {
		in = strings.NewReader(*confirm + "\n")
	} else {
		fmt.Fprintf(stdout, "\nEscriba BORRAR %d para continuar: ", sales)
	}
	if err := resetdata.Confirm(in, sales); err != nil {
		fmt.Fprintln(stdout, "Operación cancelada. No se borró nada.")
		return fail(stderr, err)
	}

	// A repeated run within the same second must not collide with the earlier backup.
	now := time.Now()
	for {
		if _, statErr := os.Stat(*dbPath + ".pre-reset-" + now.Format("20060102-150405")); statErr != nil {
			break
		}
		now = now.Add(time.Second)
	}
	info, err := resetdata.Backup(ctx, db, *dbPath, now)
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "Respaldo creado: %s (%s)\n", info.Path, humanSize(info.Size))

	opts.BackupName = filepath.Base(info.Path)
	opts.Fingerprint = info.Counts
	res, err := resetdata.Apply(ctx, db, opts)
	if err != nil {
		fmt.Fprintln(stdout, "No se borró nada. La base de datos quedó como estaba.")
		return fail(stderr, err)
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(stdout, "Aviso: %s\n", w)
	}
	fmt.Fprintln(stdout, "Listo. Datos de prueba borrados; el catálogo, los clientes y los usuarios se conservaron.")
	fmt.Fprintf(stdout, "Respaldo: %s\n", info.Path)
	return 0
}

func printStock(w io.Writer, plan *resetdata.PlanReport) {
	if len(plan.Stock) == 0 {
		fmt.Fprintln(w, "Stock: sin cambios")
		return
	}
	untraced := 0
	for _, s := range plan.Stock {
		if s.Untraced != 0 {
			untraced++
		}
	}
	fmt.Fprintf(w, "Stock de productos (incluye productos inactivos): %d cambian, %d sin respaldo en entradas\n", len(plan.Stock), untraced)
	for _, s := range plan.Stock {
		mark, inactive := "", ""
		if s.Untraced != 0 {
			mark = " !"
		}
		if !s.Active {
			inactive = " [inactivo]"
		}
		fmt.Fprintf(w, "  #%d %s%s: %g -> %g (entradas %g, vendido %g, sin respaldo %g)%s\n",
			s.ProductID, s.Name, inactive, s.Now, s.After, s.Entries, s.Sold, s.Untraced, mark)
	}
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return strings.Replace(fmt.Sprintf("%.1f MB", float64(n)/(1<<20)), ".", ",", 1)
	case n >= 1<<10:
		return strings.Replace(fmt.Sprintf("%.1f KB", float64(n)/(1<<10)), ".", ",", 1)
	}
	return fmt.Sprintf("%d B", n)
}
