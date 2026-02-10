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
	"sort"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/broker"
	"github.com/nitinkhare/algoTradingAgent/internal/config"
	"github.com/nitinkhare/algoTradingAgent/internal/market"
	"github.com/nitinkhare/algoTradingAgent/internal/risk"
	"github.com/nitinkhare/algoTradingAgent/internal/scheduler"
	"github.com/nitinkhare/algoTradingAgent/internal/storage"
	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
	"github.com/nitinkhare/algoTradingAgent/internal/webhook"
)

func main() {
	configPath := flag.String("config", "config/config.json", "path to configuration file")
	mode := flag.String("mode", "status", "run mode: nightly | market | status")
	confirmLive := flag.Bool("confirm-live", false, "required safety flag to run in live trading mode")
	flag.Parse()

	logger := log.New(os.Stdout, "[engine] ", log.LstdFlags|log.Lshortfile)

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}
	logger.Printf("config loaded: broker=%s mode=%s capital=%.2f", cfg.ActiveBroker, cfg.TradingMode, cfg.Capital)

	// ── Live mode safety gate ──
	// Both --confirm-live flag AND ALGO_LIVE_CONFIRMED=true env var are
	// required to start in live mode. This prevents accidental live trading.
	if cfg.TradingMode == config.ModeLive {
		envConfirmed := os.Getenv("ALGO_LIVE_CONFIRMED") == "true"
		if !*confirmLive || !envConfirmed {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "  ╔═══════════════════════════════════════════════════════════╗")
			fmt.Fprintln(os.Stderr, "  ║                    ⚠  LIVE MODE BLOCKED  ⚠                ║")
			fmt.Fprintln(os.Stderr, "  ╠═══════════════════════════════════════════════════════════╣")
			fmt.Fprintln(os.Stderr, "  ║  Live trading requires TWO explicit confirmations:       ║")
			fmt.Fprintln(os.Stderr, "  ║                                                           ║")
			fmt.Fprintln(os.Stderr, "  ║  1. CLI flag:   --confirm-live                            ║")
			fmt.Fprintln(os.Stderr, "  ║  2. Env var:    ALGO_LIVE_CONFIRMED=true                  ║")
			fmt.Fprintln(os.Stderr, "  ║                                                           ║")
			fmt.Fprintln(os.Stderr, "  ║  Example:                                                 ║")
			fmt.Fprintln(os.Stderr, "  ║  ALGO_LIVE_CONFIRMED=true go run ./cmd/engine \\            ║")
			fmt.Fprintln(os.Stderr, "  ║    --mode market --confirm-live                           ║")
			fmt.Fprintln(os.Stderr, "  ╚═══════════════════════════════════════════════════════════╝")
			fmt.Fprintln(os.Stderr, "")
			if !*confirmLive {
				fmt.Fprintln(os.Stderr, "  MISSING: --confirm-live flag")
			}
			if !envConfirmed {
				fmt.Fprintln(os.Stderr, "  MISSING: ALGO_LIVE_CONFIRMED=true environment variable")
			}
			fmt.Fprintln(os.Stderr, "")
			os.Exit(1)
		}
		logger.Println("LIVE MODE ACTIVE — real orders will be placed on the exchange")
	} else {
		logger.Println("PAPER MODE — simulated orders only, no real money at risk")
	}

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

	// Initialize storage (optional — engine works without DB).
	var store *storage.PostgresStore
	if cfg.DatabaseURL != "" {
		s, err := storage.NewPostgresStore(cfg.DatabaseURL)
		if err != nil {
			logger.Printf("WARNING: database not available: %v — trade logging disabled", err)
		} else {
			store = s
			defer store.Close()
			logger.Println("database connected — trade logging enabled")
		}
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
		// Start webhook server if enabled (receives order postback notifications).
		if cfg.Webhook.Enabled {
			whCfg := webhook.Config{
				Port:    cfg.Webhook.Port,
				Path:    cfg.Webhook.Path,
				Enabled: cfg.Webhook.Enabled,
			}
			whServer := webhook.NewServer(whCfg, logger)

			// Register a default handler that logs order updates.
			whServer.OnOrderUpdate(func(u webhook.OrderUpdate) {
				logger.Printf("[postback] order=%s symbol=%s status=%s filled=%d/%d avg=%.2f tag=%s",
					u.OrderID, u.Symbol, u.Status, u.FilledQty, u.Quantity, u.AveragePrice, u.CorrelationID)
				if u.ErrorCode != "" {
					logger.Printf("[postback] error: %s — %s", u.ErrorCode, u.ErrorMessage)
				}
			})

			if err := whServer.Start(); err != nil {
				logger.Fatalf("failed to start webhook server: %v", err)
			}
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = whServer.Shutdown(shutdownCtx)
			}()
		}

		registerMarketJobs(sched, cfg, activeBroker, strategies, riskMgr, store, logger)
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
	store *storage.PostgresStore,
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

			// Load stock scores from JSON.
			scoresPath := filepath.Join(cfg.Paths.AIOutputDir, today, "stock_scores.json")
			scoresData, err := os.ReadFile(scoresPath)
			if err != nil {
				logger.Printf("no stock scores for today: %v", err)
				return nil // Not an error — AI scoring hasn't run yet.
			}

			var scores []strategy.StockScore
			if err := json.Unmarshal(scoresData, &scores); err != nil {
				return fmt.Errorf("parse stock scores: %w", err)
			}
			logger.Printf("loaded %d stock scores", len(scores))

			// Sort by rank (ascending — rank 1 is best).
			sort.Slice(scores, func(i, j int) bool {
				return scores[i].Rank < scores[j].Rank
			})

			// Get current holdings to detect existing positions.
			holdings, err := b.GetHoldings(ctx)
			if err != nil {
				return fmt.Errorf("get holdings: %w", err)
			}

			// Build holdings map and open positions list for risk manager.
			holdingsMap := make(map[string]broker.Holding)
			var openPositions []strategy.PositionInfo
			for _, h := range holdings {
				holdingsMap[h.Symbol] = h
				openPositions = append(openPositions, strategy.PositionInfo{
					Symbol:     h.Symbol,
					EntryPrice: h.AveragePrice,
					Quantity:   h.Quantity,
				})
			}
			logger.Printf("existing positions: %d", len(openPositions))

			// Track available capital (decremented as orders are placed).
			availableCapital := funds.AvailableCash

			// Zero daily PnL for now (will be wired to Postgres in Task 4).
			dailyPnL := risk.DailyPnL{Date: time.Now().In(market.IST)}

			// Core loop: iterate scored stocks, run strategies, validate, execute.
			buyCount := 0
			skipCount := 0
			for _, score := range scores {
				// Load candle history for this symbol.
				csvPath := filepath.Join(cfg.Paths.MarketDataDir, score.Symbol+".csv")
				candles := market.LoadExistingCSV(csvPath)
				if len(candles) == 0 {
					logger.Printf("  %s: SKIP (no candle data)", score.Symbol)
					skipCount++
					continue
				}

				// Check if we already hold this stock.
				var currentPos *strategy.PositionInfo
				if h, exists := holdingsMap[score.Symbol]; exists {
					currentPos = &strategy.PositionInfo{
						Symbol:     h.Symbol,
						EntryPrice: h.AveragePrice,
						Quantity:   h.Quantity,
					}
				}

				// Build strategy input.
				input := strategy.StrategyInput{
					Date:              time.Now().In(market.IST),
					Regime:            regime,
					Score:             score,
					Candles:           candles,
					CurrentPosition:   currentPos,
					OpenPositionCount: len(openPositions),
					AvailableCapital:  availableCapital,
				}

				// Run each strategy.
				for _, strat := range strategies {
					intent := strat.Evaluate(input)

					switch intent.Action {
					case strategy.ActionSkip:
						logger.Printf("  %s [%s]: SKIP — %s", score.Symbol, strat.ID(), intent.Reason)
						skipCount++

					case strategy.ActionHold:
						logger.Printf("  %s [%s]: HOLD — %s", score.Symbol, strat.ID(), intent.Reason)

					case strategy.ActionBuy:
						// Validate through risk manager.
						result := riskMgr.Validate(intent, openPositions, dailyPnL, availableCapital)
						if !result.Approved {
							for _, r := range result.Rejections {
								logger.Printf("  %s [%s]: BUY REJECTED — %s: %s",
									score.Symbol, strat.ID(), r.Rule, r.Message)
							}
							logTradeAction(ctx, store, logger, "BUY_REJECTED", result.Rejections[0].Rule,
								fmt.Sprintf("rejected: %s", result.Rejections[0].Message), intent, string(regime.Regime))
							continue
						}

						// Convert TradeIntent to broker.Order and execute.
						order := broker.Order{
							Symbol:   intent.Symbol,
							Exchange: "NSE",
							Side:     broker.OrderSideBuy,
							Type:     broker.OrderTypeLimit,
							Quantity: intent.Quantity,
							Price:    intent.Price,
							Product:  "CNC",
							Tag:      intent.SignalID,
						}

						resp, err := b.PlaceOrder(ctx, order)
						if err != nil {
							logger.Printf("  %s [%s]: BUY ORDER FAILED — %v",
								score.Symbol, strat.ID(), err)
							continue
						}

						logger.Printf("  %s [%s]: BUY ORDER PLACED — id=%s status=%s qty=%d price=%.2f sl=%.2f tgt=%.2f | %s",
							score.Symbol, strat.ID(), resp.OrderID, resp.Status,
							intent.Quantity, intent.Price, intent.StopLoss, intent.Target, intent.Reason)

						logTradeAction(ctx, store, logger, "BUY_PLACED", "ORDER_PLACED",
							fmt.Sprintf("order=%s qty=%d price=%.2f", resp.OrderID, intent.Quantity, intent.Price),
							intent, string(regime.Regime))
						saveTradeRecord(ctx, store, logger, intent, resp.OrderID)

						// Update tracking state.
						availableCapital -= intent.Price * float64(intent.Quantity)
						openPositions = append(openPositions, strategy.PositionInfo{
							Symbol:     intent.Symbol,
							EntryPrice: intent.Price,
							Quantity:   intent.Quantity,
							StopLoss:   intent.StopLoss,
							Target:     intent.Target,
							StrategyID: intent.StrategyID,
							SignalID:   intent.SignalID,
							EntryTime:  time.Now().In(market.IST),
						})
						buyCount++

					case strategy.ActionExit:
						// Validate through risk manager (exits always pass).
						result := riskMgr.Validate(intent, openPositions, dailyPnL, availableCapital)
						if !result.Approved {
							logger.Printf("  %s [%s]: EXIT REJECTED (unexpected) — %v",
								score.Symbol, strat.ID(), result.Rejections)
							continue
						}

						order := broker.Order{
							Symbol:   intent.Symbol,
							Exchange: "NSE",
							Side:     broker.OrderSideSell,
							Type:     broker.OrderTypeLimit,
							Quantity: intent.Quantity,
							Price:    intent.Price,
							Product:  "CNC",
							Tag:      intent.SignalID,
						}

						resp, err := b.PlaceOrder(ctx, order)
						if err != nil {
							logger.Printf("  %s [%s]: EXIT ORDER FAILED — %v",
								score.Symbol, strat.ID(), err)
							continue
						}

						logger.Printf("  %s [%s]: EXIT ORDER PLACED — id=%s status=%s qty=%d price=%.2f | %s",
							score.Symbol, strat.ID(), resp.OrderID, resp.Status,
							intent.Quantity, intent.Price, intent.Reason)

						logTradeAction(ctx, store, logger, "EXIT_PLACED", "ORDER_PLACED",
							fmt.Sprintf("order=%s qty=%d price=%.2f reason=%s", resp.OrderID, intent.Quantity, intent.Price, intent.Reason),
							intent, string(regime.Regime))
					}
				}
			}

			logger.Printf("trade execution complete: %d buys placed, %d skipped", buyCount, skipCount)
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

			if len(holdings) == 0 {
				logger.Println("no open positions to monitor")
				return nil
			}

			logger.Printf("open positions: %d", len(holdings))

			// Load regime for today.
			today := time.Now().In(market.IST).Format("2006-01-02")
			regimePath := filepath.Join(cfg.Paths.AIOutputDir, today, "market_regime.json")
			regimeData, err := os.ReadFile(regimePath)
			if err != nil {
				logger.Printf("no regime data for exit monitoring: %v", err)
				return nil
			}
			var regime strategy.MarketRegimeData
			if err := json.Unmarshal(regimeData, &regime); err != nil {
				return fmt.Errorf("parse regime: %w", err)
			}

			// Load scores for today (needed for exit evaluation).
			scoresPath := filepath.Join(cfg.Paths.AIOutputDir, today, "stock_scores.json")
			scoresData, err := os.ReadFile(scoresPath)
			if err != nil {
				logger.Printf("no stock scores for exit monitoring: %v", err)
				return nil
			}
			var scores []strategy.StockScore
			if err := json.Unmarshal(scoresData, &scores); err != nil {
				return fmt.Errorf("parse stock scores: %w", err)
			}

			// Build score lookup map.
			scoreMap := make(map[string]strategy.StockScore)
			for _, s := range scores {
				scoreMap[s.Symbol] = s
			}

			// Build open positions list for risk manager.
			var openPositions []strategy.PositionInfo
			for _, h := range holdings {
				openPositions = append(openPositions, strategy.PositionInfo{
					Symbol:     h.Symbol,
					EntryPrice: h.AveragePrice,
					Quantity:   h.Quantity,
				})
			}

			dailyPnL := risk.DailyPnL{Date: time.Now().In(market.IST)}
			funds, err := b.GetFunds(ctx)
			if err != nil {
				return fmt.Errorf("get funds for exit monitoring: %w", err)
			}

			exitCount := 0
			for _, h := range holdings {
				logger.Printf("  checking %s: qty=%d avg=%.2f last=%.2f pnl=%.2f",
					h.Symbol, h.Quantity, h.AveragePrice, h.LastPrice, h.PnL)

				// Find score for this holding.
				score, hasScore := scoreMap[h.Symbol]
				if !hasScore {
					logger.Printf("  %s: no score data, skipping exit check", h.Symbol)
					continue
				}

				// Load candle data.
				csvPath := filepath.Join(cfg.Paths.MarketDataDir, h.Symbol+".csv")
				candles := market.LoadExistingCSV(csvPath)

				// Build strategy input with CurrentPosition set (triggers exit path).
				currentPos := &strategy.PositionInfo{
					Symbol:     h.Symbol,
					EntryPrice: h.AveragePrice,
					Quantity:   h.Quantity,
				}

				input := strategy.StrategyInput{
					Date:              time.Now().In(market.IST),
					Regime:            regime,
					Score:             score,
					Candles:           candles,
					CurrentPosition:   currentPos,
					OpenPositionCount: len(openPositions),
					AvailableCapital:  funds.AvailableCash,
				}

				// Run each strategy's exit evaluation.
				for _, strat := range strategies {
					intent := strat.Evaluate(input)

					if intent.Action == strategy.ActionExit {
						// Validate (exits always pass risk checks).
						result := riskMgr.Validate(intent, openPositions, dailyPnL, funds.AvailableCash)
						if !result.Approved {
							logger.Printf("  %s [%s]: EXIT REJECTED (unexpected) — %v",
								h.Symbol, strat.ID(), result.Rejections)
							continue
						}

						order := broker.Order{
							Symbol:   intent.Symbol,
							Exchange: "NSE",
							Side:     broker.OrderSideSell,
							Type:     broker.OrderTypeLimit,
							Quantity: intent.Quantity,
							Price:    intent.Price,
							Product:  "CNC",
							Tag:      intent.SignalID,
						}

						resp, err := b.PlaceOrder(ctx, order)
						if err != nil {
							logger.Printf("  %s [%s]: EXIT ORDER FAILED — %v",
								h.Symbol, strat.ID(), err)
							continue
						}

						logger.Printf("  %s [%s]: EXIT ORDER PLACED — id=%s status=%s qty=%d price=%.2f | %s",
							h.Symbol, strat.ID(), resp.OrderID, resp.Status,
							intent.Quantity, intent.Price, intent.Reason)

						logTradeAction(ctx, store, logger, "EXIT_PLACED", "ORDER_PLACED",
							fmt.Sprintf("order=%s qty=%d price=%.2f reason=%s", resp.OrderID, intent.Quantity, intent.Price, intent.Reason),
							intent, string(regime.Regime))
						exitCount++

					} else if intent.Action == strategy.ActionHold {
						logger.Printf("  %s [%s]: HOLD — %s", h.Symbol, strat.ID(), intent.Reason)
					}
				}
			}

			logger.Printf("exit monitoring complete: %d exit orders placed out of %d positions",
				exitCount, len(holdings))
			return nil
		},
	})
}

// logTradeAction persists a trade action to the database if the store is available.
// Errors are logged but never fatal — the engine must keep running even if DB is down.
func logTradeAction(ctx context.Context, store *storage.PostgresStore, logger *log.Logger,
	action, reasonCode, message string, intent strategy.TradeIntent, regime string,
) {
	if store == nil {
		return
	}
	tl := &storage.TradeLog{
		Timestamp:  time.Now(),
		StrategyID: intent.StrategyID,
		Symbol:     intent.Symbol,
		Action:     action,
		ReasonCode: reasonCode,
		Message:    message,
		InputsJSON: storage.LogInputs(intent, regime),
	}
	if err := store.SaveTradeLog(ctx, tl); err != nil {
		logger.Printf("[db] failed to log trade: %v", err)
	}
}

// saveTradeRecord persists a new open trade to the database if the store is available.
func saveTradeRecord(ctx context.Context, store *storage.PostgresStore, logger *log.Logger,
	intent strategy.TradeIntent, orderID string,
) {
	if store == nil {
		return
	}
	trade := &storage.TradeRecord{
		StrategyID: intent.StrategyID,
		SignalID:   intent.SignalID,
		Symbol:     intent.Symbol,
		Side:       string(strategy.ActionBuy),
		Quantity:   intent.Quantity,
		EntryPrice: intent.Price,
		StopLoss:   intent.StopLoss,
		Target:     intent.Target,
		EntryTime:  time.Now(),
		Status:     "open",
	}
	if err := store.SaveTrade(ctx, trade); err != nil {
		logger.Printf("[db] failed to save trade: %v", err)
	}
}
