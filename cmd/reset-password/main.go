// reset-password resets the admin user password in data.db.
// Usage: go run ./cmd/reset-password --db data.db --password nuevapassword
package main

import (
	"flag"
	"fmt"
	"log"

	"database/sql"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "data.db", "ruta al archivo data.db")
	password := flag.String("password", "", "nueva contraseña para admin (requerido)")
	flag.Parse()

	if *password == "" {
		log.Fatal("debes especificar --password")
	}
	db, err := sql.Open("sqlite", *dbPath+"?_pragma=busy_timeout(5000)") // wait for the running POS instead of failing with SQLITE_BUSY
	if err != nil {
		log.Fatalf("no se pudo abrir la base de datos: %v", err)
	}
	defer db.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("error generando hash: %v", err)
	}

	res, err := db.Exec(`UPDATE users SET password_hash = ? WHERE username = 'admin'`, string(hash))
	if err != nil {
		log.Fatalf("error actualizando contraseña: %v", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		log.Fatal("usuario 'admin' no encontrado en la base de datos")
	}

	fmt.Println("✓ Contraseña de admin actualizada correctamente.")
	fmt.Printf("  Usuario: admin\n  Contraseña: %s\n", *password)
}
