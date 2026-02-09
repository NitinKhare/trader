// Package main is the entry point for the algoTradingAgent engine.
//
// The engine:
//   1. Loads configuration
//   2. Initializes all components (broker, storage, calendar, strategies, risk)
//   3. Reads AI outputs (scores, regime) from file-based contract
//   4. Runs the strategy engine against the stock universe
//   5. Validates trade intents through risk management
//   6. Executes approved orders via the active broker
//   7. Logs every action for auditability
//
// Modes:
//   - "nightly": Run nightly jobs (data sync, AI scoring, watchlist generation)
//   - "market":  Run market-hour jobs (execute pre-planned trades, manage exits)
//   - "status":  Print current system and market status
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/broker"
	"github.com/nitinkhare/algoTradingAgent/internal/config"
	"github.com/nitinkhare/algoTradingAgent/internal/market"
	"github.com/nitinkhare/algoTradingAgent/internal/risk"
	"github.com/nitinkhare/algoTradingAgent/internal/scheduler"
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

func main() {
	configPath := flag.String("config", "config/config.json", "path to configuration file")
	mode := flag.String("mode", "status", "run mode: nightly | market | status")
	flag.Parse()

	logger := log.New(os.Stdout, "[engine] ", log.LstdFlags|log.Lshortfile)

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}
	logger.Printf("config loaded: broker=%s mode=%s capital=%.2f", cfg.ActiveBroker, cfg.TradingMode, cfg.Capital)

	// Initialize market calendar.
	cal, err := market.NewCalendar(cfg.MarketCalendarPath)
	if err != nil {
		logger.Fatalf("failed to load market calendar: %v", err)
	}

	// Initialize broker.
	var activeBroker broker.Broker
	if cfg.TradingMode == config.ModePaper {
		activeBroker = broker.NewPaperBroker(cfg.Capital)
		logger.Println("using PAPER broker")
	} else {
		brokerCfg, ok := cfg.BrokerConfig[cfg.ActiveBroker]
		if !ok {
			logger.Fatalf("no broker config found for %q", cfg.ActiveBroker)
		}
		activeBroker, err = broker.New(cfg.ActiveBroker, brokerCfg)
		if err != nil {
			logger.Fatalf("failed to initialize broker %q: %v", cfg.ActiveBroker, err)
		}
		logger.Printf("using LIVE broker: %s", cfg.ActiveBroker)
	}

	// Initialize risk manager.
	riskMgr := risk.NewManager(cfg.Risk, cfg.Capital)

	// Initialize strategies.
	strategies := []strategy.Strategy{
		strategy.NewTrendFollowStrategy(cfg.Risk),
	}
	logger.Printf("loaded %d strategies", len(strategies))

	// Initialize scheduler.
	sched := scheduler.New(cal, logger)

	switch *mode {
	case "status":
		runStatus(logger, cal, activeBroker, cfg)

	case "nightly":
		registerNightlyJobs(sched, cfg, logger)
		ctx := context.Background()
		if err := sched.RunNightlyJobs(ctx); err != nil {
			logger.Fatalf("nightly jobs failed: %v", err)
		}

	case "market":
		registerMarketJobs(sched, cfg, activeBroker, strategies, riskMgr, logger)
		ctx := context.Background()
		if err := sched.RunMarketHourJobs(ctx); err != nil {
			logger.Fatalf("market jobs failed: %v", err)
		}

	default:
		logger.Fatalf("unknown mode: %s (expected: nightly, market, status)", *mode)
	}
}

// runStatus prints the current state of the system.
func runStatus(logger *log.Logger, cal *market.Calendar, b broker.Broker, cfg *config.Config) {
	now := time.Now()
	logger.Println("=== System Status ===")
	logger.Printf("Time (IST): %s", now.In(market.IST).Format("2006-01-02 15:04:05"))
	logger.Printf("Trading day: %v", cal.IsTradingDay(now))
	logger.Printf("Market open: %v", cal.IsMarketOpen(now))
	logger.Printf("Next session in: %v", cal.TimeUntilNextSession(now).Round(time.Minute))
	logger.Printf("Mode: %s", cfg.TradingMode)
	logger.Printf("Broker: %s", cfg.ActiveBroker)

	if reason := cal.HolidayReason(now); reason != "" {
		logger.Printf("Holiday: %s", reason)
	}

	ctx := context.Background()
	funds, err := b.GetFunds(ctx)
	if err != nil {
		logger.Printf("Funds: error - %v", err)
	} else {
		logger.Printf("Available cash: %.2f", funds.AvailableCash)
		logger.Printf("Used margin: %.2f", funds.UsedMargin)
	}
}

// registerNightlyJobs sets up all nightly scheduled jobs.
func registerNightlyJobs(sched *scheduler.Scheduler, cfg *config.Config, logger *log.Logger) {
	// Job 0: Sync market data from Dhan API.
	// This runs BEFORE AI scoring so that fresh candles are available.
	sched.RegisterJob(scheduler.Job{
		Name: "sync_market_data",
		Type: scheduler.JobTypeNightly,
		RunFunc: func(ctx context.Context) error {
			logger.Println("syncing market data from Dhan API...")

			// Parse Dhan data config from broker config.
			dhanCfgJSON, ok := cfg.BrokerConfig["dhan"]
			if !ok {
				return fmt.Errorf("no dhan config found in broker_config")
			}
			var dataCfg market.DhanDataConfig
			if err := json.Unmarshal(dhanCfgJSON, &dataCfg); err != nil {
				return fmt.Errorf("parse dhan data config: %w", err)
			}
			dataCfg.MarketDataDir = cfg.Paths.MarketDataDir

			// Create Dhan data provider.
			provider, err := market.NewDhanDataProvider(dataCfg)
			if err != nil {
				return fmt.Errorf("create dhan data provider: %w", err)
			}

			// Load stock universe.
			universeData, err := os.ReadFile("config/stock_universe.json")
			if err != nil {
				return fmt.Errorf("read stock universe: %w", err)
			}
			var universe struct {
				Symbols []string `json:"symbols"`
			}
			if err := json.Unmarshal(universeData, &universe); err != nil {
				return fmt.Errorf("parse stock universe: %w", err)
			}

			// Add NIFTY50 for regime detection.
			symbols := append(universe.Symbols, "NIFTY50")

			// Fetch 1 year of history and export as CSVs for Python.
			today := time.Now().In(market.IST)
			from := today.AddDate(-1, 0, 0)

			successCount := 0
			failCount := 0
			for i, symbol := range symbols {
				logger.Printf("  [%d/%d] fetching %s...", i+1, len(symbols), symbol)
				candles, err := provider.FetchDailyCandles(ctx, symbol, from, today)
				if err != nil {
					logger.Printf("  WARN: %s failed: %v", symbol, err)
					failCount++
					continue
				}
				if err := provider.ExportCSV(symbol, candles); err != nil {
					logger.Printf("  WARN: %s CSV export failed: %v", symbol, err)
					failCount++
					continue
				}
				logger.Printf("  %s: %d candles exported", symbol, len(candles))
				successCount++
			}

			logger.Printf("market data sync complete: %d succeeded, %d failed out of %d symbols",
				successCount, failCount, len(symbols))

			if successCount == 0 {
				return fmt.Errorf("all symbols failed to sync — check Dhan credentials")
			}
			return nil
		},
	})

	// Job 1: Trigger AI scoring pipeline.
	sched.RegisterJob(scheduler.Job{
		Name: "run_ai_scoring",
		Type: scheduler.JobTypeNightly,
		RunFunc: func(ctx context.Context) error {
			logger.Println("triggering AI scoring pipeline...")
			// This would shell out to: python -m python_ai.run_scoring --date <today> --output-dir <path>
			// For now, we just check if outputs exist.
			today := time.Now().In(market.IST).Format("2006-01-02")
			regimePath := filepath.Join(cfg.Paths.AIOutputDir, today, "market_regime.json")
			scoresPath := filepath.Join(cfg.Paths.AIOutputDir, today, "stock_scores.parquet")

			if _, err := os.Stat(regimePath); os.IsNotExist(err) {
				logger.Printf("AI outputs not found at %s — run python scoring pipeline first", regimePath)
				return fmt.Errorf("AI scoring outputs not found for %s", today)
			}
			if _, err := os.Stat(scoresPath); os.IsNotExist(err) {
				logger.Printf("AI outputs not found at %s — run python scoring pipeline first", scoresPath)
				return fmt.Errorf("AI scoring outputs not found for %s", today)
			}

			logger.Printf("AI outputs verified for %s", today)
			return nil
		},
	})

	// Job 2: Generate watchlist from AI scores.
	sched.RegisterJob(scheduler.Job{
		Name: "generate_watchlist",
		Type: scheduler.JobTypeNightly,
		RunFunc: func(ctx context.Context) error {
			logger.Println("generating next-day watchlist from AI scores...")
			today := time.Now().In(market.IST).Format("2006-01-02")

			// Read market regime.
			regimePath := filepath.Join(cfg.Paths.AIOutputDir, today, "market_regime.json")
			regimeData, err := os.ReadFile(regimePath)
			if err != nil {
				return fmt.Errorf("read regime: %w", err)
			}

			var regime strategy.MarketRegimeData
			if err := json.Unmarshal(regimeData, &regime); err != nil {
				return fmt.Errorf("parse regime: %w", err)
			}

			logger.Printf("market regime: %s (confidence: %.2f)", regime.Regime, regime.Confidence)

			if regime.Regime == strategy.RegimeBear {
				logger.Println("BEAR regime — no new entries. Watchlist will focus on exits only.")
			}

			return nil
		},
	})
}

// registerMarketJobs sets up all market-hour scheduled jobs.
func registerMarketJobs(
	sched *scheduler.Scheduler,
	cfg *config.Config,
	b broker.Broker,
	strategies []strategy.Strategy,
	riskMgr *risk.Manager,
	logger *log.Logger,
) {
	// Job: Execute pre-planned trades from watchlist.
	sched.RegisterJob(scheduler.Job{
		Name: "execute_trades",
		Type: scheduler.JobTypeMarketHour,
		RunFunc: func(ctx context.Context) error {
			logger.Println("checking for trade execution opportunities...")

			// Read AI scores and regime for today.
			today := time.Now().In(market.IST).Format("2006-01-02")
			regimePath := filepath.Join(cfg.Paths.AIOutputDir, today, "market_regime.json")
			regimeData, err := os.ReadFile(regimePath)
			if err != nil {
				logger.Printf("no AI outputs for today: %v", err)
				return nil // Not an error — just no data yet.
			}

			var regime strategy.MarketRegimeData
			if err := json.Unmarshal(regimeData, &regime); err != nil {
				return fmt.Errorf("parse regime: %w", err)
			}

			logger.Printf("regime: %s | strategies: %d", regime.Regime, len(strategies))

			// Get current account state.
			funds, err := b.GetFunds(ctx)
			if err != nil {
				return fmt.Errorf("get funds: %w", err)
			}
			logger.Printf("available capital: %.2f", funds.AvailableCash)

			// TODO: Load stock scores from parquet, iterate through universe,
			// run strategies, validate through risk, execute orders.
			// This is the core trading loop that will be fully implemented
			// when the data layer is connected.

			return nil
		},
	})

	// Job: Monitor existing positions for exit conditions.
	sched.RegisterJob(scheduler.Job{
		Name: "monitor_exits",
		Type: scheduler.JobTypeMarketHour,
		RunFunc: func(ctx context.Context) error {
			logger.Println("monitoring open positions for exit conditions...")

			holdings, err := b.GetHoldings(ctx)
			if err != nil {
				return fmt.Errorf("get holdings: %v", err)
			}

			logger.Printf("open positions: %d", len(holdings))
			for _, h := range holdings {
				logger.Printf("  %s: qty=%d avg=%.2f last=%.2f pnl=%.2f",
					h.Symbol, h.Quantity, h.AveragePrice, h.LastPrice, h.PnL)
			}

			return nil
		},
	})
}
