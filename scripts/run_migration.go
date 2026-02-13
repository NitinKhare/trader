package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dbURL := flag.String("db", "postgres://algo:algo123@localhost:5432/algo_trading?sslmode=disable", "database URL")
	migrationFile := flag.String("file", "", "migration SQL file to run")
	flag.Parse()

	if *migrationFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: run_migration -file <path-to-sql-file> [-db <url>]\n")
		os.Exit(1)
	}

	// Read migration file
	sqlBytes, err := os.ReadFile(*migrationFile)
	if err != nil {
		log.Fatalf("failed to read migration file: %v", err)
	}

	// Connect to database
	db, err := sql.Open("pgx", *dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	fmt.Printf("Connected to database\n")
	fmt.Printf("Running migration: %s\n", filepath.Base(*migrationFile))

	// Execute migration
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		log.Fatalf("failed to execute migration: %v", err)
	}

	fmt.Printf("✓ Migration applied successfully\n")
}
