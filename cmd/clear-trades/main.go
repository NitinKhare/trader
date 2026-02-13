// clear-trades - Delete all trades from today and start fresh
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	confirmFlag := flag.Bool("confirm", false, "Confirm deletion (must be explicit)")
	flag.Parse()

	if !*confirmFlag {
		fmt.Println("❌ SAFETY CHECK - Must confirm deletion")
		fmt.Println("")
		fmt.Println("This will DELETE all trades from TODAY:")
		fmt.Println("")
		fmt.Printf("Date: %s\n", time.Now().Format("2006-01-02"))
		fmt.Println("")
		fmt.Println("To proceed, run:")
		fmt.Println("  go run ./cmd/clear-trades --confirm")
		fmt.Println("")
		os.Exit(0)
	}

	// Connect to database
	db, err := sql.Open("pgx", "postgres://algo:algo123@localhost:5432/algo_trading?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	fmt.Printf("🗑️  Deleting all data from: %s\n", today)
	fmt.Println("")

	// Delete trades
	result, err := db.Exec(`
		DELETE FROM trades
		WHERE DATE(created_at AT TIME ZONE 'IST') = $1
	`, today)
	if err != nil {
		log.Fatalf("Failed to delete trades: %v", err)
	}
	tradesDeleted, _ := result.RowsAffected()
	fmt.Printf("  ✅ Deleted %d trades\n", tradesDeleted)

	// Delete signals
	result, err = db.Exec(`
		DELETE FROM signals
		WHERE DATE(signal_date) = $1
	`, today)
	if err != nil {
		log.Fatalf("Failed to delete signals: %v", err)
	}
	signalsDeleted, _ := result.RowsAffected()
	fmt.Printf("  ✅ Deleted %d signals\n", signalsDeleted)

	// Delete trade logs
	result, err = db.Exec(`
		DELETE FROM trade_logs
		WHERE DATE(timestamp) = $1
	`, today)
	if err != nil {
		log.Fatalf("Failed to delete trade logs: %v", err)
	}
	logsDeleted, _ := result.RowsAffected()
	fmt.Printf("  ✅ Deleted %d trade logs\n", logsDeleted)

	fmt.Println("")
	fmt.Println("✨ Clean slate ready!")
	fmt.Println("")
	fmt.Println("You can now run:")
	fmt.Println("  go run ./cmd/engine --mode market")
	fmt.Println("")
}
