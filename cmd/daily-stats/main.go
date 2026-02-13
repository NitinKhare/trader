// Package main - Daily Trading Statistics CLI
// Shows trades taken, capital used, and P&L for the day
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TradeRecord represents a completed trade
type TradeRecord struct {
	Symbol      string
	Quantity    int
	EntryPrice  float64
	ExitPrice   float64
	PnL         float64
	ExitTime    time.Time
	StopLoss    float64
	Target      float64
	CapitalUsed float64
}

// DailySummary represents the daily statistics
type DailySummary struct {
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	DailyPnL       float64
	CapitalUsed    float64
	OpenPositions  int
	WinRate        float64
}

const (
	// ANSI color codes
	Reset   = "\033[0m"
	Red     = "\033[0;31m"
	Green   = "\033[0;32m"
	Yellow  = "\033[1;33m"
	Blue    = "\033[0;34m"
	Cyan    = "\033[0;36m"
	Magenta = "\033[0;35m"
)

func main() {
	dateFlag := flag.String("date", "", "Date in YYYY-MM-DD format (defaults to today)")
	flag.Parse()

	var date string
	if *dateFlag == "" {
		date = time.Now().Format("2006-01-02")
	} else {
		date = *dateFlag
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", date); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid date format. Use YYYY-MM-DD\n")
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open("pgx", "postgres://algo:algo123@localhost:5432/algo_trading?sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure PostgreSQL is running and credentials are correct\n")
		os.Exit(1)
	}

	// Get daily summary
	summary, err := getDailySummary(db, date)
	if err != nil {
		log.Fatalf("Failed to get daily summary: %v", err)
	}

	// Display results
	displaySummary(date, summary)

	// Get detailed trades
	trades, err := getDetailedTrades(db, date)
	if err != nil {
		log.Fatalf("Failed to get trades: %v", err)
	}

	if len(trades) > 0 {
		displayDetailedTrades(trades)
	}

	// Get open positions
	openTrades, err := getOpenPositions(db)
	if err != nil {
		log.Fatalf("Failed to get open positions: %v", err)
	}

	displayOpenPositions(openTrades)
}

func getDailySummary(db *sql.DB, date string) (*DailySummary, error) {
	query := `
SELECT
  COUNT(*) as total_trades,
  COALESCE(SUM(CASE WHEN exit_price > entry_price THEN 1 ELSE 0 END), 0) as winning_trades,
  COALESCE(SUM(CASE WHEN exit_price < entry_price THEN 1 ELSE 0 END), 0) as losing_trades,
  COALESCE(ROUND(SUM((exit_price - entry_price) * quantity), 2), 0) as daily_pnl,
  COALESCE(ROUND(SUM(ABS(entry_price * quantity)), 2), 0) as capital_used
FROM trades
WHERE DATE(exit_time AT TIME ZONE 'IST') = $1 AND status = 'closed';
`

	var summary DailySummary
	var totalTrades int

	err := db.QueryRow(query, date).Scan(
		&totalTrades,
		&summary.WinningTrades,
		&summary.LosingTrades,
		&summary.DailyPnL,
		&summary.CapitalUsed,
	)

	if err != nil {
		return nil, err
	}

	summary.TotalTrades = totalTrades

	// Calculate win rate
	if totalTrades > 0 {
		summary.WinRate = (float64(summary.WinningTrades) / float64(totalTrades)) * 100
	}

	// Get open positions count
	countQuery := "SELECT COUNT(*) FROM trades WHERE status = 'open';"
	db.QueryRow(countQuery).Scan(&summary.OpenPositions)

	return &summary, nil
}

func getDetailedTrades(db *sql.DB, date string) ([]TradeRecord, error) {
	query := `
SELECT
  symbol,
  quantity,
  entry_price,
  COALESCE(exit_price, entry_price) as exit_price,
  ROUND((COALESCE(exit_price, entry_price) - entry_price) * quantity, 2) as pnl,
  exit_time AT TIME ZONE 'IST' as exit_time,
  ROUND(entry_price * quantity, 2) as capital_used
FROM trades
WHERE DATE(exit_time AT TIME ZONE 'IST') = $1 AND status = 'closed'
ORDER BY exit_time DESC;
`

	rows, err := db.Query(query, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []TradeRecord
	for rows.Next() {
		var t TradeRecord
		if err := rows.Scan(&t.Symbol, &t.Quantity, &t.EntryPrice, &t.ExitPrice, &t.PnL, &t.ExitTime, &t.CapitalUsed); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}

	return trades, rows.Err()
}

func getOpenPositions(db *sql.DB) ([]TradeRecord, error) {
	query := `
SELECT
  symbol,
  quantity,
  entry_price,
  0 as exit_price,
  0 as pnl,
  entry_time as exit_time,
  ROUND(entry_price * quantity, 2) as capital_used,
  stop_loss,
  target
FROM trades
WHERE status = 'open'
ORDER BY entry_time DESC;
`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []TradeRecord
	for rows.Next() {
		var t TradeRecord
		if err := rows.Scan(&t.Symbol, &t.Quantity, &t.EntryPrice, &t.ExitPrice, &t.PnL, &t.ExitTime, &t.CapitalUsed, &t.StopLoss, &t.Target); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}

	return trades, rows.Err()
}

func displaySummary(date string, summary *DailySummary) {
	fmt.Printf("%s╔════════════════════════════════════════════════════════════╗%s\n", Cyan, Reset)
	fmt.Printf("%s║           DAILY TRADING STATISTICS                         ║%s\n", Cyan, Reset)
	fmt.Printf("%s║           Date: %-48s ║%s\n", Cyan, date, Reset)
	fmt.Printf("%s╚════════════════════════════════════════════════════════════╝%s\n", Cyan, Reset)
	fmt.Println()

	if summary.TotalTrades == 0 {
		fmt.Printf("%sNo trades found for %s%s\n\n", Yellow, date, Reset)
		return
	}

	// Color for P&L
	pnlColor := Green
	if summary.DailyPnL < 0 {
		pnlColor = Red
	}

	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Blue, Reset)
	fmt.Printf("%sSUMMARY%s\n", Blue, Reset)
	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Blue, Reset)

	fmt.Printf("  %sTotal Trades:%s      %s%d%s\n", Yellow, Reset, Green, summary.TotalTrades, Reset)
	fmt.Printf("  %sWinning Trades:%s    %s%d%s\n", Yellow, Reset, Green, summary.WinningTrades, Reset)
	fmt.Printf("  %sLosing Trades:%s     %s%d%s\n", Yellow, Reset, Red, summary.LosingTrades, Reset)
	fmt.Printf("  %sWin Rate:%s          %s%.1f%%%s\n", Yellow, Reset, Green, summary.WinRate, Reset)
	fmt.Println()

	fmt.Printf("  %sDaily P&L:%s         %s₹%.2f%s\n", Yellow, Reset, pnlColor, summary.DailyPnL, Reset)
	fmt.Printf("  %sCapital Used:%s      %s₹%.2f%s\n", Yellow, Reset, Cyan, summary.CapitalUsed, Reset)

	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Blue, Reset)
	fmt.Println()
}

func displayDetailedTrades(trades []TradeRecord) {
	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Blue, Reset)
	fmt.Printf("%sDETAILED TRADES%s\n", Blue, Reset)
	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Blue, Reset)
	fmt.Println()

	// Header
	fmt.Printf("%s%-12s %-10s %-12s %-12s %-12s %-12s%s\n",
		Magenta,
		"Symbol",
		"Quantity",
		"Entry Price",
		"Exit Price",
		"P&L",
		"Exit Time",
		Reset,
	)
	fmt.Printf("%s%s%s\n", Magenta, strings.Repeat("-", 81), Reset)

	// Rows
	for _, t := range trades {
		pnlColor := Green
		if t.PnL < 0 {
			pnlColor = Red
		}

		fmt.Printf("%-12s %-10d %-12.2f %-12.2f %s%-12.2f%s %-12s\n",
			t.Symbol,
			t.Quantity,
			t.EntryPrice,
			t.ExitPrice,
			pnlColor,
			t.PnL,
			Reset,
			t.ExitTime.Format("15:04:05"),
		)
	}

	fmt.Println()
}

func displayOpenPositions(trades []TradeRecord) {
	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Blue, Reset)
	fmt.Printf("%sOPEN POSITIONS%s\n", Blue, Reset)
	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Blue, Reset)
	fmt.Println()

	if len(trades) == 0 {
		fmt.Printf("  %sNo open positions%s\n", Green, Reset)
	} else {
		fmt.Printf("  %sOpen Positions: %s%d%s\n", Yellow, Green, len(trades), Reset)
		fmt.Println()

		// Header
		fmt.Printf("%s%-12s %-10s %-12s %-12s %-12s %-12s%s\n",
			Magenta,
			"Symbol",
			"Quantity",
			"Entry Price",
			"Capital",
			"Stop Loss",
			"Target",
			Reset,
		)
		fmt.Printf("%s%s%s\n", Magenta, strings.Repeat("-", 81), Reset)

		// Rows
		for _, t := range trades {
			fmt.Printf("%-12s %-10d %-12.2f %-12.2f %-12.2f %-12.2f\n",
				t.Symbol,
				t.Quantity,
				t.EntryPrice,
				t.CapitalUsed,
				t.StopLoss,
				t.Target,
			)
		}
	}

	fmt.Println()
	fmt.Printf("%s╔════════════════════════════════════════════════════════════╗%s\n", Cyan, Reset)
	fmt.Printf("%s║                    END OF REPORT                           ║%s\n", Cyan, Reset)
	fmt.Printf("%s╚════════════════════════════════════════════════════════════╝%s\n", Cyan, Reset)
}
