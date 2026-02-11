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
	"os/signal"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/analytics"
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

	// Initialize circuit breaker for automatic trading halt on repeated failures.
	cb := risk.NewCircuitBreaker(cfg.Risk.CircuitBreaker, logger)

	// Initialize strategies.
	strategies := []strategy.Strategy{
		strategy.NewTrendFollowStrategy(cfg.Risk),
		strategy.NewMeanReversionStrategy(cfg.Risk),
		strategy.NewBreakoutStrategy(cfg.Risk),
		strategy.NewMomentumStrategy(cfg.Risk),
	}
	logger.Printf("loaded %d strategies", len(strategies))

	// Load sector map from stock universe for sector concentration checks.
	sectorMap := loadSectorMap(logger)

	// Initialize scheduler.
	sched := scheduler.New(cal, logger)

	// Load open position state from database for context recovery on restart.
	var tc *tradeContext
	if store != nil {
		tc = restorePositions(context.Background(), store, activeBroker, cfg, logger)
	}

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
		// WaitGroup for tracking in-flight jobs during graceful shutdown.
		var wg sync.WaitGroup

		// Start webhook server if enabled (receives order postback notifications).
		var whServer *webhook.Server
		if cfg.Webhook.Enabled {
			whCfg := webhook.Config{
				Port:    cfg.Webhook.Port,
				Path:    cfg.Webhook.Path,
				Enabled: cfg.Webhook.Enabled,
			}
			whServer = webhook.NewServer(whCfg, logger)

			// Register postback-driven position update handler.
			registerPostbackHandler(whServer, activeBroker, store, tc, cb, cfg, logger)

			if err := whServer.Start(); err != nil {
				logger.Fatalf("failed to start webhook server: %v", err)
			}
		}

		registerMarketJobs(sched, cfg, activeBroker, strategies, riskMgr, store, tc, sectorMap, cb, &wg, logger)

		// Set up context with signal handling for graceful shutdown.
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		// Config hot-reload: watch config file for risk param changes.
		watcher := config.NewConfigWatcher(*configPath, cfg, logger)
		watcher.OnChange(func(old, new *config.Config) {
			riskMgr.UpdateRiskConfig(new.Risk)
			cb.UpdateConfig(new.Risk.CircuitBreaker)
			// Update cfg pointer so jobs see new trailing stop config etc.
			*cfg = *new
			logger.Printf("[hot-reload] risk config updated")
		})
		if watchErr := watcher.Start(); watchErr != nil {
			logger.Printf("WARNING: config watcher failed to start: %v", watchErr)
		}
		defer watcher.Stop()

		pollingInterval := time.Duration(cfg.PollingIntervalMinutes) * time.Minute
		if pollingInterval <= 0 {
			// Backward compatible: run once and exit.
			logger.Println("[market] polling_interval_minutes=0 — running jobs once")
			if err := sched.RunMarketHourJobs(ctx); err != nil {
				logger.Fatalf("market jobs failed: %v", err)
			}
		} else {
			logger.Printf("[market] continuous polling: interval=%v", pollingInterval)
			runContinuousMarketLoop(ctx, sched, cal, pollingInterval, logger)
		}

		// Graceful shutdown: wait for in-flight jobs to complete.
		gracefulShutdown(&wg, whServer, logger)

	case "analytics":
		runAnalytics(store, cfg, logger)

	case "backtest":
		runBacktest(cfg, strategies, riskMgr, sectorMap, logger)

	default:
		logger.Fatalf("unknown mode: %s (expected: nightly, market, status, analytics, backtest)", *mode)
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
	tc *tradeContext,
	sectorMap map[string]string,
	cb *risk.CircuitBreaker,
	wg *sync.WaitGroup,
	logger *log.Logger,
) {
	// Job: Execute pre-planned trades from watchlist.
	sched.RegisterJob(scheduler.Job{
		Name: "execute_trades",
		Type: scheduler.JobTypeMarketHour,
		RunFunc: func(ctx context.Context) error {
			if wg != nil {
				wg.Add(1)
				defer wg.Done()
			}
			logger.Println("checking for trade execution opportunities...")

			// Circuit breaker check: skip new entries if tripped.
			if cb != nil && cb.IsTripped() {
				logger.Printf("[circuit-breaker] trading halted: %s — skipping execute_trades", cb.TripReason())
				return nil
			}

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
				if cb != nil {
					cb.RecordFailure(fmt.Sprintf("GetFunds: %v", err))
				}
				return fmt.Errorf("get funds: %w", err)
			}
			if cb != nil {
				cb.RecordSuccess()
			}
			logger.Printf("available capital: %.2f", funds.AvailableCash)

			// Update risk manager's capital base from live broker balance.
			// This ensures risk percentages adapt when money is added/withdrawn.
			if funds.TotalBalance > 0 {
				riskMgr.UpdateCapital(funds.TotalBalance)
				logger.Printf("risk capital base: %.2f", funds.TotalBalance)
			}

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
				if cb != nil {
					cb.RecordFailure(fmt.Sprintf("GetHoldings: %v", err))
				}
				return fmt.Errorf("get holdings: %w", err)
			}
			if cb != nil {
				cb.RecordSuccess()
			}

			// Reconcile DB open trades with actual broker holdings.
			reconcilePositions(ctx, store, logger, tc, holdings)

			// Build holdings map and open positions list, enriched with DB context.
			holdingsMap := make(map[string]broker.Holding)
			var openPositions []strategy.PositionInfo
			for _, h := range holdings {
				holdingsMap[h.Symbol] = h
				openPositions = append(openPositions, enrichPosition(h, tc))
			}
			logger.Printf("existing positions: %d", len(openPositions))

			// Track available capital (decremented as orders are placed).
			availableCapital := funds.AvailableCash

			// Calculate daily PnL from DB (realized) and holdings (unrealized).
			dailyPnL := calculateDailyPnL(ctx, store, tc, holdings, logger)

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
					enriched := enrichPosition(h, tc)
					currentPos = &enriched
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
						result := riskMgr.Validate(intent, openPositions, dailyPnL, availableCapital, sectorMap)
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
							if cb != nil {
								cb.RecordFailure(fmt.Sprintf("PlaceOrder BUY %s: %v", score.Symbol, err))
							}
							continue
						}
						if cb != nil {
							cb.RecordSuccess()
						}

						logger.Printf("  %s [%s]: BUY ORDER PLACED — id=%s status=%s qty=%d price=%.2f sl=%.2f tgt=%.2f | %s",
							score.Symbol, strat.ID(), resp.OrderID, resp.Status,
							intent.Quantity, intent.Price, intent.StopLoss, intent.Target, intent.Reason)

						logTradeAction(ctx, store, logger, "BUY_PLACED", "ORDER_PLACED",
							fmt.Sprintf("order=%s qty=%d price=%.2f", resp.OrderID, intent.Quantity, intent.Price),
							intent, string(regime.Regime))

						// Poll order status to confirm fill before placing SL order.
						finalStatus, pollErr := pollOrderStatus(ctx, b, resp.OrderID, 3*time.Second, 30*time.Second, logger)
						if pollErr != nil {
							logger.Printf("  %s: order %s poll timeout: %v — skipping SL placement",
								score.Symbol, resp.OrderID, pollErr)
							saveTradeRecord(ctx, store, logger, intent, resp.OrderID)
							continue
						}

						if finalStatus.Status == broker.OrderStatusRejected {
							logger.Printf("  %s: order %s REJECTED — %s", score.Symbol, resp.OrderID, finalStatus.Message)
							logTradeAction(ctx, store, logger, "BUY_REJECTED_BY_BROKER", "ORDER_REJECTED",
								fmt.Sprintf("order=%s rejected: %s", resp.OrderID, finalStatus.Message),
								intent, string(regime.Regime))
							continue
						}

						if finalStatus.Status != broker.OrderStatusCompleted {
							logger.Printf("  %s: order %s not filled (status=%s) — saving without SL",
								score.Symbol, resp.OrderID, finalStatus.Status)
							saveTradeRecord(ctx, store, logger, intent, resp.OrderID)
							continue
						}

						// Order FILLED — save trade and place stop-loss order.
						logger.Printf("  %s: order %s FILLED avg=%.2f", score.Symbol, resp.OrderID, finalStatus.AveragePrice)
						tradeID := saveTradeRecord(ctx, store, logger, intent, resp.OrderID)

						// Update tradeContext so SL order ID can be tracked.
						if tc != nil && tradeID > 0 {
							tc.Set(intent.Symbol, &storage.TradeRecord{
								ID:         tradeID,
								StrategyID: intent.StrategyID,
								SignalID:   intent.SignalID,
								Symbol:     intent.Symbol,
								Side:       string(strategy.ActionBuy),
								Quantity:   intent.Quantity,
								EntryPrice: intent.Price,
								StopLoss:   intent.StopLoss,
								Target:     intent.Target,
								OrderID:    resp.OrderID,
								EntryTime:  time.Now(),
								Status:     "open",
							})
						}

						// Place SL-M order at the stop-loss price.
						slOrderID := placeStopLossOrder(ctx, b, store, logger,
							tradeID, intent.Symbol, "NSE", intent.Quantity, intent.StopLoss, intent.SignalID)

						// Update tradeContext with SL order ID for exit cancellation.
						if tc != nil && slOrderID != "" {
							if trade, ok := tc.Get(intent.Symbol); ok {
								trade.SLOrderID = slOrderID
							}
						}

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
						result := riskMgr.Validate(intent, openPositions, dailyPnL, availableCapital, sectorMap)
						if !result.Approved {
							logger.Printf("  %s [%s]: EXIT REJECTED (unexpected) — %v",
								score.Symbol, strat.ID(), result.Rejections)
							continue
						}

						// Cancel existing SL order before placing exit (prevents double sell).
						if tc != nil {
							if trade, ok := tc.Get(intent.Symbol); ok && trade.SLOrderID != "" {
								cancelStopLossOrder(ctx, b, logger, trade.SLOrderID, intent.Symbol)
							}
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

						// Mark trade as closed in the database.
						closeTradeRecord(ctx, store, logger, tc, intent.Symbol, intent.Price, intent.Reason)
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
			if wg != nil {
				wg.Add(1)
				defer wg.Done()
			}
			logger.Println("monitoring open positions for exit conditions...")

			holdings, err := b.GetHoldings(ctx)
			if err != nil {
				if cb != nil {
					cb.RecordFailure(fmt.Sprintf("GetHoldings (monitor_exits): %v", err))
				}
				return fmt.Errorf("get holdings: %v", err)
			}
			if cb != nil {
				cb.RecordSuccess()
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

			// Build open positions list, enriched with DB context.
			var openPositions []strategy.PositionInfo
			for _, h := range holdings {
				openPositions = append(openPositions, enrichPosition(h, tc))
			}

			// Calculate daily PnL from DB (realized) and holdings (unrealized).
			dailyPnL := calculateDailyPnL(ctx, store, tc, holdings, logger)
			funds, err := b.GetFunds(ctx)
			if err != nil {
				return fmt.Errorf("get funds for exit monitoring: %w", err)
			}

			exitCount := 0
			for _, h := range holdings {
				logger.Printf("  checking %s: qty=%d avg=%.2f last=%.2f pnl=%.2f",
					h.Symbol, h.Quantity, h.AveragePrice, h.LastPrice, h.PnL)

				// Max holding period check: force exit if held too long.
				if cfg.Risk.MaxHoldDays > 0 && tc != nil {
					if trade, ok := tc.Get(h.Symbol); ok {
						holdDays := int(time.Since(trade.EntryTime).Hours() / 24)
						if holdDays >= cfg.Risk.MaxHoldDays {
							logger.Printf("  %s: max hold period %d days exceeded (held %d days) — forcing exit",
								h.Symbol, cfg.Risk.MaxHoldDays, holdDays)

							// Cancel SL order.
							if trade.SLOrderID != "" {
								cancelStopLossOrder(ctx, b, logger, trade.SLOrderID, h.Symbol)
							}

							// Place exit order.
							exitPrice := h.LastPrice
							if exitPrice == 0 {
								exitPrice = h.AveragePrice
							}
							exitOrder := broker.Order{
								Symbol:   h.Symbol,
								Exchange: "NSE",
								Side:     broker.OrderSideSell,
								Type:     broker.OrderTypeLimit,
								Quantity: h.Quantity,
								Price:    exitPrice,
								Product:  "CNC",
								Tag:      "max-hold-exit",
							}
							resp, err := b.PlaceOrder(ctx, exitOrder)
							if err != nil {
								logger.Printf("  %s: MAX HOLD EXIT ORDER FAILED — %v", h.Symbol, err)
							} else {
								logger.Printf("  %s: MAX HOLD EXIT ORDER PLACED — id=%s price=%.2f (held %d days)",
									h.Symbol, resp.OrderID, exitPrice, holdDays)
								closeTradeRecord(ctx, store, logger, tc, h.Symbol, exitPrice, "max_hold_period")
								exitCount++
							}
							continue // skip strategy eval for this holding
						}
					}
				}

				// Trailing stop-loss adjustment: raise SL as price moves in favor.
				if cfg.Risk.TrailingStop.Enabled && tc != nil {
					adjustTrailingStopLoss(ctx, b, store, logger, tc, cb, h, cfg.Risk.TrailingStop)
				}

				// Find score for this holding.
				score, hasScore := scoreMap[h.Symbol]
				if !hasScore {
					logger.Printf("  %s: no score data, skipping exit check", h.Symbol)
					continue
				}

				// Load candle data.
				csvPath := filepath.Join(cfg.Paths.MarketDataDir, h.Symbol+".csv")
				candles := market.LoadExistingCSV(csvPath)

				// Build strategy input with CurrentPosition enriched from DB.
				enriched := enrichPosition(h, tc)

				input := strategy.StrategyInput{
					Date:              time.Now().In(market.IST),
					Regime:            regime,
					Score:             score,
					Candles:           candles,
					CurrentPosition:   &enriched,
					OpenPositionCount: len(openPositions),
					AvailableCapital:  funds.AvailableCash,
				}

				// Run each strategy's exit evaluation.
				for _, strat := range strategies {
					intent := strat.Evaluate(input)

					if intent.Action == strategy.ActionExit {
						// Validate (exits always pass risk checks).
						result := riskMgr.Validate(intent, openPositions, dailyPnL, funds.AvailableCash, sectorMap)
						if !result.Approved {
							logger.Printf("  %s [%s]: EXIT REJECTED (unexpected) — %v",
								h.Symbol, strat.ID(), result.Rejections)
							continue
						}

						// Cancel existing SL order before placing exit (prevents double sell).
						if tc != nil {
							if trade, ok := tc.Get(h.Symbol); ok && trade.SLOrderID != "" {
								cancelStopLossOrder(ctx, b, logger, trade.SLOrderID, h.Symbol)
							}
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

						// Mark trade as closed in the database.
						closeTradeRecord(ctx, store, logger, tc, h.Symbol, intent.Price, intent.Reason)
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
// Returns the trade ID (used for linking SL orders), or 0 if DB is unavailable.
func saveTradeRecord(ctx context.Context, store *storage.PostgresStore, logger *log.Logger,
	intent strategy.TradeIntent, orderID string,
) int64 {
	if store == nil {
		return 0
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
		OrderID:    orderID,
		EntryTime:  time.Now(),
		Status:     "open",
	}
	if err := store.SaveTrade(ctx, trade); err != nil {
		logger.Printf("[db] failed to save trade: %v", err)
		return 0
	}
	logger.Printf("[db] saved trade: id=%d symbol=%s order_id=%s", trade.ID, trade.Symbol, orderID)
	return trade.ID
}

// ────────────────────────────────────────────────────────────────────
// Position persistence and reconciliation
// ────────────────────────────────────────────────────────────────────

// tradeContext maps symbol to its open TradeRecord from the DB.
// This provides the full entry context (stop loss, target, strategy ID, etc.)
// that is lost when the engine restarts.
// Thread-safe: accessed from market-hour jobs and webhook postback handler concurrently.
type tradeContext struct {
	mu     sync.RWMutex
	trades map[string]*storage.TradeRecord // symbol -> open trade
}

// Get returns the trade record for a symbol (thread-safe).
func (tc *tradeContext) Get(symbol string) (*storage.TradeRecord, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	t, ok := tc.trades[symbol]
	return t, ok
}

// Set stores a trade record for a symbol (thread-safe).
func (tc *tradeContext) Set(symbol string, trade *storage.TradeRecord) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.trades[symbol] = trade
}

// Delete removes a trade record by symbol (thread-safe).
func (tc *tradeContext) Delete(symbol string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	delete(tc.trades, symbol)
}

// GetByOrderID finds a trade by its entry order ID (thread-safe).
func (tc *tradeContext) GetByOrderID(orderID string) (string, *storage.TradeRecord, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	for sym, t := range tc.trades {
		if t.OrderID == orderID {
			return sym, t, true
		}
	}
	return "", nil, false
}

// GetBySLOrderID finds a trade by its stop-loss order ID (thread-safe).
func (tc *tradeContext) GetBySLOrderID(slOrderID string) (string, *storage.TradeRecord, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	for sym, t := range tc.trades {
		if t.SLOrderID == slOrderID {
			return sym, t, true
		}
	}
	return "", nil, false
}

// Snapshot returns a copy of all trades for iteration (thread-safe).
func (tc *tradeContext) Snapshot() map[string]*storage.TradeRecord {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	snap := make(map[string]*storage.TradeRecord, len(tc.trades))
	for k, v := range tc.trades {
		snap[k] = v
	}
	return snap
}

// Len returns the number of tracked trades (thread-safe).
func (tc *tradeContext) Len() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return len(tc.trades)
}

// loadTradeContext loads open trades from the database and returns a lookup structure.
// Returns nil if the store is not available (graceful degradation).
func loadTradeContext(ctx context.Context, store *storage.PostgresStore, logger *log.Logger) *tradeContext {
	if store == nil {
		return nil
	}

	openTrades, err := store.GetOpenTrades(ctx)
	if err != nil {
		logger.Printf("[positions] WARNING: failed to load open trades from DB: %v", err)
		return nil
	}

	tc := &tradeContext{
		trades: make(map[string]*storage.TradeRecord, len(openTrades)),
	}
	for i := range openTrades {
		t := &openTrades[i]
		tc.trades[t.Symbol] = t // direct access OK during init (no concurrent access yet)
		logger.Printf("[positions] restored: %s strategy=%s entry=%.2f sl=%.2f tgt=%.2f qty=%d",
			t.Symbol, t.StrategyID, t.EntryPrice, t.StopLoss, t.Target, t.Quantity)
	}

	logger.Printf("[positions] loaded %d open trades from database", len(tc.trades))
	return tc
}

// enrichPosition creates a fully populated PositionInfo by merging
// broker holding data with trade context from the database.
// If no DB context exists for a symbol, returns a PositionInfo with only
// broker-sourced fields (current behavior, graceful degradation).
func enrichPosition(h broker.Holding, tc *tradeContext) strategy.PositionInfo {
	pos := strategy.PositionInfo{
		Symbol:     h.Symbol,
		EntryPrice: h.AveragePrice,
		Quantity:   h.Quantity,
	}

	if tc == nil {
		return pos
	}

	if trade, ok := tc.Get(h.Symbol); ok {
		pos.StopLoss = trade.StopLoss
		pos.Target = trade.Target
		pos.StrategyID = trade.StrategyID
		pos.SignalID = trade.SignalID
		pos.EntryTime = trade.EntryTime
		// Use the DB entry price as the authoritative entry price.
		// Broker's AveragePrice may differ slightly due to fill slippage.
		pos.EntryPrice = trade.EntryPrice
	}

	return pos
}

// reconcilePositions compares DB open trades with actual broker holdings
// and handles discrepancies:
//   - Position in DB but not in broker -> close the DB trade (manually sold/stopped out)
//   - Position in broker but not in DB -> log warning (opened outside engine)
//   - Quantity mismatch -> log warning (partial close)
func reconcilePositions(
	ctx context.Context,
	store *storage.PostgresStore,
	logger *log.Logger,
	tc *tradeContext,
	holdings []broker.Holding,
) {
	if store == nil || tc == nil {
		return
	}

	// Build holdings lookup.
	holdingsMap := make(map[string]broker.Holding, len(holdings))
	for _, h := range holdings {
		holdingsMap[h.Symbol] = h
	}

	// Check each DB open trade against broker holdings.
	snapshot := tc.Snapshot()
	for symbol, trade := range snapshot {
		h, brokerHasIt := holdingsMap[symbol]
		if !brokerHasIt {
			// Position was closed outside the engine (manual sell, broker-side SL, etc.)
			logger.Printf("[reconcile] %s: DB has open trade (id=%d) but broker has no holding — marking as closed (external_close)",
				symbol, trade.ID)
			if err := store.CloseTrade(ctx, trade.ID, 0, "external_close"); err != nil {
				logger.Printf("[reconcile] WARNING: failed to close trade %d: %v", trade.ID, err)
			}
			tc.Delete(symbol)
			continue
		}

		// Quantity mismatch: partial close outside engine.
		if h.Quantity != trade.Quantity {
			logger.Printf("[reconcile] %s: quantity mismatch — DB=%d broker=%d (partial close detected)",
				symbol, trade.Quantity, h.Quantity)
		}
	}

	// Check for positions in broker but not in DB.
	for symbol := range holdingsMap {
		if _, inDB := tc.Get(symbol); !inDB {
			logger.Printf("[reconcile] %s: broker has holding but no open trade in DB — position lacks SL/target context",
				symbol)
		}
	}
}

// restorePositions loads open trades from the DB and:
//  1. Builds a tradeContext for enriching PositionInfo throughout the engine.
//  2. Seeds the paper broker with restored holdings (if paper mode).
//
// Returns the tradeContext, or nil if DB is unavailable.
func restorePositions(
	ctx context.Context,
	store *storage.PostgresStore,
	b broker.Broker,
	cfg *config.Config,
	logger *log.Logger,
) *tradeContext {
	tc := loadTradeContext(ctx, store, logger)
	if tc == nil || len(tc.trades) == 0 {
		return tc
	}

	// For paper mode, seed the paper broker with restored holdings.
	if cfg.TradingMode == config.ModePaper {
		if pb, ok := b.(*broker.PaperBroker); ok {
			for _, trade := range tc.trades {
				pb.RestoreHolding(trade.Symbol, "NSE", trade.Quantity, trade.EntryPrice)
				logger.Printf("[positions] seeded paper broker: %s qty=%d price=%.2f",
					trade.Symbol, trade.Quantity, trade.EntryPrice)
			}
		}
	}

	return tc
}

// closeTradeRecord marks a trade as closed in the database after an exit order.
func closeTradeRecord(
	ctx context.Context,
	store *storage.PostgresStore,
	logger *log.Logger,
	tc *tradeContext,
	symbol string,
	exitPrice float64,
	exitReason string,
) {
	if store == nil || tc == nil {
		return
	}
	trade, ok := tc.Get(symbol)
	if !ok {
		return
	}
	if err := store.CloseTrade(ctx, trade.ID, exitPrice, exitReason); err != nil {
		logger.Printf("[db] failed to close trade %d: %v", trade.ID, err)
	} else {
		logger.Printf("[db] trade %d closed: %s exit=%.2f reason=%s",
			trade.ID, symbol, exitPrice, exitReason)
		tc.Delete(symbol)
	}
}

// ────────────────────────────────────────────────────────────────────
// Order status polling (Feature 2)
// ────────────────────────────────────────────────────────────────────

// pollOrderStatus polls GetOrderStatus until the order reaches a terminal state
// (COMPLETED, REJECTED, CANCELLED) or the timeout expires.
// Paper broker returns COMPLETED immediately, making this a no-op in paper mode.
func pollOrderStatus(
	ctx context.Context,
	b broker.Broker,
	orderID string,
	pollInterval time.Duration,
	maxDuration time.Duration,
	logger *log.Logger,
) (*broker.OrderStatusResponse, error) {
	// Immediate first check.
	status, err := b.GetOrderStatus(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("poll order %s: %w", orderID, err)
	}
	if isTerminalOrderStatus(status.Status) {
		return status, nil
	}

	// Poll until terminal or timeout.
	deadline := time.Now().Add(maxDuration)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return status, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				logger.Printf("[order-poll] %s: timeout (%v), last=%s filled=%d/%d",
					orderID, maxDuration, status.Status, status.FilledQty, status.FilledQty+status.PendingQty)
				return status, fmt.Errorf("order %s poll timeout after %v", orderID, maxDuration)
			}
			status, err = b.GetOrderStatus(ctx, orderID)
			if err != nil {
				logger.Printf("[order-poll] %s: check failed: %v", orderID, err)
				continue // transient error, keep polling
			}
			logger.Printf("[order-poll] %s: %s filled=%d avg=%.2f",
				orderID, status.Status, status.FilledQty, status.AveragePrice)
			if isTerminalOrderStatus(status.Status) {
				return status, nil
			}
		}
	}
}

// isTerminalOrderStatus returns true if the order has reached a final state.
func isTerminalOrderStatus(s broker.OrderStatus) bool {
	return s == broker.OrderStatusCompleted ||
		s == broker.OrderStatusRejected ||
		s == broker.OrderStatusCancelled
}

// ────────────────────────────────────────────────────────────────────
// Stop-loss order management (Feature 1)
// ────────────────────────────────────────────────────────────────────

// placeStopLossOrder places a SL-M (stop-loss market) sell order after an entry fill.
// Updates the trade record in the database with the SL order ID.
// Returns the SL order ID on success, or empty string on failure.
func placeStopLossOrder(
	ctx context.Context,
	b broker.Broker,
	store *storage.PostgresStore,
	logger *log.Logger,
	tradeID int64,
	symbol, exchange string,
	quantity int,
	stopLossPrice float64,
	signalID string,
) string {
	slOrder := broker.Order{
		Symbol:       symbol,
		Exchange:     exchange,
		Side:         broker.OrderSideSell,
		Type:         broker.OrderTypeSLM,
		Quantity:     quantity,
		Price:        0, // Market order on trigger
		TriggerPrice: stopLossPrice,
		Product:      "CNC",
		Tag:          signalID + "-SL",
	}

	logger.Printf("[sl-order] placing SL-M for %s: qty=%d trigger=%.2f", symbol, quantity, stopLossPrice)

	resp, err := b.PlaceOrder(ctx, slOrder)
	if err != nil {
		logger.Printf("[sl-order] FAILED for %s: %v", symbol, err)
		return ""
	}

	logger.Printf("[sl-order] placed for %s: order_id=%s status=%s trigger=%.2f",
		symbol, resp.OrderID, resp.Status, stopLossPrice)

	// Persist SL order ID in the trade record.
	if store != nil && tradeID > 0 {
		if err := store.UpdateTradeSLOrderID(ctx, tradeID, resp.OrderID); err != nil {
			logger.Printf("[sl-order] WARNING: failed to save sl_order_id for trade %d: %v", tradeID, err)
		}
	}

	return resp.OrderID
}

// cancelStopLossOrder cancels an existing SL order before placing a strategy exit.
// Returns true if the cancel succeeded or there was nothing to cancel.
func cancelStopLossOrder(
	ctx context.Context,
	b broker.Broker,
	logger *log.Logger,
	slOrderID string,
	symbol string,
) bool {
	if slOrderID == "" {
		return true
	}
	logger.Printf("[sl-order] canceling %s for %s before exit", slOrderID, symbol)
	if err := b.CancelOrder(ctx, slOrderID); err != nil {
		logger.Printf("[sl-order] WARNING: cancel %s failed: %v — proceeding with exit", slOrderID, err)
		return false
	}
	logger.Printf("[sl-order] canceled %s for %s", slOrderID, symbol)
	return true
}

// ────────────────────────────────────────────────────────────────────
// Daily PnL calculation (Feature 4)
// ────────────────────────────────────────────────────────────────────

// calculateDailyPnL computes realized and unrealized PnL for today.
// Realized: from closed trades in the DB (store.GetDailyPnL).
// Unrealized: from current holdings vs entry prices (using tradeContext).
func calculateDailyPnL(
	ctx context.Context,
	store *storage.PostgresStore,
	tc *tradeContext,
	holdings []broker.Holding,
	logger *log.Logger,
) risk.DailyPnL {
	today := time.Now().In(market.IST)
	pnl := risk.DailyPnL{Date: today}

	// Realized PnL from closed trades today.
	if store != nil {
		realized, err := store.GetDailyPnL(ctx, today)
		if err != nil {
			logger.Printf("[pnl] WARNING: realized PnL query failed: %v", err)
		} else {
			pnl.RealizedPnL = realized
		}
	}

	// Unrealized PnL from open holdings.
	for _, h := range holdings {
		entryPrice := h.AveragePrice
		if tc != nil {
			if trade, ok := tc.Get(h.Symbol); ok {
				entryPrice = trade.EntryPrice
			}
		}
		pnl.UnrealizedPnL += float64(h.Quantity) * (h.LastPrice - entryPrice)
	}

	if pnl.RealizedPnL != 0 || pnl.UnrealizedPnL != 0 {
		logger.Printf("[pnl] realized=%.2f unrealized=%.2f total=%.2f",
			pnl.RealizedPnL, pnl.UnrealizedPnL, pnl.RealizedPnL+pnl.UnrealizedPnL)
	}

	return pnl
}

// ────────────────────────────────────────────────────────────────────
// Continuous market-hour polling (Feature 3)
// ────────────────────────────────────────────────────────────────────

// runContinuousMarketLoop runs market-hour jobs repeatedly during trading hours.
// Exits when the market closes or the context is cancelled (SIGINT/SIGTERM).
func runContinuousMarketLoop(
	ctx context.Context,
	sched *scheduler.Scheduler,
	cal *market.Calendar,
	interval time.Duration,
	logger *log.Logger,
) {
	// Run immediately on startup if market is open.
	if cal.IsMarketOpen(time.Now()) {
		logger.Println("[market-loop] initial run...")
		if err := sched.RunMarketHourJobs(ctx); err != nil {
			logger.Printf("[market-loop] initial run failed: %v", err)
		}
	} else {
		logger.Println("[market-loop] market is currently closed, waiting for open...")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Println("[market-loop] shutdown signal received, exiting")
			return
		case <-ticker.C:
			now := time.Now()
			if !cal.IsMarketOpen(now) {
				logger.Printf("[market-loop] market closed at %s — exiting loop",
					now.In(market.IST).Format("15:04:05"))
				return
			}
			logger.Printf("[market-loop] running jobs at %s...",
				now.In(market.IST).Format("15:04:05"))
			if err := sched.RunMarketHourJobs(ctx); err != nil {
				logger.Printf("[market-loop] jobs failed: %v", err)
			}
		}
	}
}

// ────────────────────────────────────────────────────────────────────
// Sector map loading
// ────────────────────────────────────────────────────────────────────

// loadSectorMap reads sector information from stock_universe.json.
// Returns nil (not an error) if the file doesn't exist or lacks sector data.
func loadSectorMap(logger *log.Logger) map[string]string {
	data, err := os.ReadFile("config/stock_universe.json")
	if err != nil {
		logger.Printf("[sectors] WARNING: cannot load stock_universe.json: %v — sector limits disabled", err)
		return nil
	}

	var universe struct {
		Stocks []struct {
			Symbol string `json:"symbol"`
			Sector string `json:"sector"`
		} `json:"stocks"`
	}
	if err := json.Unmarshal(data, &universe); err != nil {
		logger.Printf("[sectors] WARNING: cannot parse stock_universe.json: %v — sector limits disabled", err)
		return nil
	}

	if len(universe.Stocks) == 0 {
		return nil
	}

	sectorMap := make(map[string]string, len(universe.Stocks))
	for _, s := range universe.Stocks {
		if s.Symbol != "" && s.Sector != "" {
			sectorMap[s.Symbol] = s.Sector
		}
	}

	logger.Printf("[sectors] loaded sector map: %d stocks", len(sectorMap))
	return sectorMap
}

// ────────────────────────────────────────────────────────────────────
// Analytics mode
// ────────────────────────────────────────────────────────────────────

// runAnalytics loads all closed trades from the database and prints a performance report.
func runAnalytics(store *storage.PostgresStore, cfg *config.Config, logger *log.Logger) {
	if store == nil {
		logger.Fatal("[analytics] database is required for analytics mode")
	}

	ctx := context.Background()
	trades, err := store.GetAllClosedTrades(ctx)
	if err != nil {
		logger.Fatalf("[analytics] failed to load closed trades: %v", err)
	}

	logger.Printf("[analytics] loaded %d closed trades", len(trades))

	report := analytics.Analyze(trades, cfg.Capital)
	fmt.Println(analytics.FormatReport(report))
}

// ────────────────────────────────────────────────────────────────────
// Graceful shutdown
// ────────────────────────────────────────────────────────────────────

// gracefulShutdown waits for in-flight jobs to complete and shuts down
// the webhook server. Called after the market loop exits.
func gracefulShutdown(wg *sync.WaitGroup, whServer *webhook.Server, logger *log.Logger) {
	logger.Println("[shutdown] waiting for in-flight jobs to complete...")

	done := make(chan struct{})
	go func() {
		if wg != nil {
			wg.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
		logger.Println("[shutdown] all jobs completed")
	case <-time.After(30 * time.Second):
		logger.Println("[shutdown] timeout waiting for jobs — forcing shutdown")
	}

	// Shut down webhook server.
	if whServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := whServer.Shutdown(shutdownCtx); err != nil {
			logger.Printf("[shutdown] webhook server error: %v", err)
		} else {
			logger.Println("[shutdown] webhook server stopped")
		}
	}

	logger.Println("[shutdown] complete")
}

// ────────────────────────────────────────────────────────────────────
// Trailing stop-loss adjustment
// ────────────────────────────────────────────────────────────────────

// adjustTrailingStopLoss checks if the current price warrants raising the
// stop-loss for a position. If so, cancels the old SL order and places a new
// one at the higher level. Updates both tradeContext and database.
func adjustTrailingStopLoss(
	ctx context.Context,
	b broker.Broker,
	store *storage.PostgresStore,
	logger *log.Logger,
	tc *tradeContext,
	cb *risk.CircuitBreaker,
	h broker.Holding,
	tsCfg config.TrailingStopConfig,
) {
	if tc == nil || !tsCfg.Enabled || tsCfg.TrailPct <= 0 {
		return
	}

	trade, ok := tc.Get(h.Symbol)
	if !ok || trade.StopLoss <= 0 {
		return
	}

	lastPrice := h.LastPrice
	if lastPrice <= 0 {
		return
	}

	// Check activation: trailing only starts after ActivationPct profit.
	profitPct := (lastPrice - trade.EntryPrice) / trade.EntryPrice * 100
	if profitPct < tsCfg.ActivationPct {
		return // not enough profit yet to start trailing
	}

	// Calculate new trailing stop.
	newSL := lastPrice * (1 - tsCfg.TrailPct/100)

	// Only raise the stop-loss, never lower it.
	if newSL <= trade.StopLoss {
		return
	}

	logger.Printf("[trailing-sl] %s: raising SL from %.2f to %.2f (price=%.2f profit=%.1f%%)",
		h.Symbol, trade.StopLoss, newSL, lastPrice, profitPct)

	// Cancel the existing SL order.
	if trade.SLOrderID != "" {
		cancelStopLossOrder(ctx, b, logger, trade.SLOrderID, h.Symbol)
	}

	// Place new SL-M order at the higher level.
	newSLOrderID := placeStopLossOrder(ctx, b, store, logger,
		trade.ID, h.Symbol, "NSE", h.Quantity, newSL, trade.SignalID)

	if newSLOrderID == "" {
		logger.Printf("[trailing-sl] WARNING: failed to place new SL for %s — position unprotected", h.Symbol)
		if cb != nil {
			cb.RecordFailure(fmt.Sprintf("trailing SL placement failed for %s", h.Symbol))
		}
		return
	}

	// Update tradeContext.
	trade.StopLoss = newSL
	trade.SLOrderID = newSLOrderID

	// Update database.
	if store != nil {
		if err := store.UpdateTradeStopLoss(ctx, trade.ID, newSL, newSLOrderID); err != nil {
			logger.Printf("[trailing-sl] WARNING: DB update failed for %s: %v", h.Symbol, err)
		}
	}

	if cb != nil {
		cb.RecordSuccess()
	}
}

// ────────────────────────────────────────────────────────────────────
// Postback-driven position updates
// ────────────────────────────────────────────────────────────────────

// registerPostbackHandler wires the webhook server to update trade records
// in real-time when Dhan sends order postback notifications.
func registerPostbackHandler(
	whServer *webhook.Server,
	b broker.Broker,
	store *storage.PostgresStore,
	tc *tradeContext,
	cb *risk.CircuitBreaker,
	cfg *config.Config,
	logger *log.Logger,
) {
	if whServer == nil {
		return
	}

	whServer.OnOrderUpdate(func(u webhook.OrderUpdate) {
		logger.Printf("[postback] order=%s symbol=%s side=%s status=%s filled=%d/%d avg=%.2f tag=%s",
			u.OrderID, u.Symbol, u.Side, u.Status, u.FilledQty, u.Quantity, u.AveragePrice, u.CorrelationID)

		if u.ErrorCode != "" {
			logger.Printf("[postback] error: %s — %s", u.ErrorCode, u.ErrorMessage)
		}

		if tc == nil {
			return
		}

		ctx := context.Background()

		switch u.Status {
		case "TRADED", "COMPLETED":
			// Check if this is an entry order fill.
			if u.Side == "BUY" {
				sym, trade, found := tc.GetByOrderID(u.OrderID)
				if found && trade.SLOrderID == "" {
					logger.Printf("[postback] entry filled for %s — placing SL order", sym)
					slOrderID := placeStopLossOrder(ctx, b, store, logger,
						trade.ID, sym, "NSE", trade.Quantity, trade.StopLoss, trade.SignalID)
					if slOrderID != "" {
						trade.SLOrderID = slOrderID
						logger.Printf("[postback] SL placed for %s: order_id=%s", sym, slOrderID)
					}
					if cb != nil {
						cb.RecordSuccess()
					}
				}
			}

			// Check if this is a SL order fill (stop-loss triggered).
			if u.Side == "SELL" {
				sym, trade, found := tc.GetBySLOrderID(u.OrderID)
				if found {
					logger.Printf("[postback] SL triggered for %s at avg=%.2f", sym, u.AveragePrice)
					exitPrice := u.AveragePrice
					if exitPrice == 0 {
						exitPrice = trade.StopLoss
					}
					if store != nil {
						if err := store.CloseTrade(ctx, trade.ID, exitPrice, "stop_loss"); err != nil {
							logger.Printf("[postback] WARNING: failed to close trade %d: %v", trade.ID, err)
						} else {
							logger.Printf("[postback] trade %d closed: %s exit=%.2f reason=stop_loss",
								trade.ID, sym, exitPrice)
						}
					}
					tc.Delete(sym)
				}
			}

		case "REJECTED":
			logger.Printf("[postback] ORDER REJECTED: %s %s — %s", u.OrderID, u.Symbol, u.ErrorMessage)
			if cb != nil {
				cb.RecordFailure(fmt.Sprintf("order rejected: %s %s: %s", u.OrderID, u.Symbol, u.ErrorMessage))
			}

			// Critical: if SL order was rejected, position is unprotected.
			if u.Side == "SELL" {
				if sym, _, found := tc.GetBySLOrderID(u.OrderID); found {
					logger.Printf("[postback] CRITICAL: SL order rejected for %s — position UNPROTECTED", sym)
				}
			}

		case "CANCELLED":
			logger.Printf("[postback] order cancelled: %s %s (normal for SL cancels before exits)",
				u.OrderID, u.Symbol)
		}
	})
}

// ────────────────────────────────────────────────────────────────────
// Backtest mode
// ────────────────────────────────────────────────────────────────────

// backtestPosition tracks a virtual open position during backtesting.
type backtestPosition struct {
	symbol     string
	entryPrice float64
	quantity   int
	stopLoss   float64
	target     float64
	strategyID string
	entryDate  time.Time
}

// runBacktest runs strategies against historical data to validate before going live.
// It iterates day-by-day through available AI outputs and market data.
func runBacktest(
	cfg *config.Config,
	strategies []strategy.Strategy,
	riskMgr *risk.Manager,
	sectorMap map[string]string,
	logger *log.Logger,
) {
	logger.Println("[backtest] starting backtest run...")

	// Scan ai_outputs directory for available dates.
	aiDir := cfg.Paths.AIOutputDir
	entries, err := os.ReadDir(aiDir)
	if err != nil {
		logger.Fatalf("[backtest] cannot read AI outputs dir %s: %v", aiDir, err)
	}

	// Collect dates that have both regime and scores.
	var dates []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dateStr := entry.Name()
		regimePath := filepath.Join(aiDir, dateStr, "market_regime.json")
		scoresPath := filepath.Join(aiDir, dateStr, "stock_scores.json")
		if _, err := os.Stat(regimePath); err == nil {
			if _, err := os.Stat(scoresPath); err == nil {
				dates = append(dates, dateStr)
			}
		}
	}

	sort.Strings(dates)

	if len(dates) == 0 {
		logger.Fatal("[backtest] no AI output data found — run nightly pipeline first")
	}

	logger.Printf("[backtest] found %d trading days with AI data: %s to %s",
		len(dates), dates[0], dates[len(dates)-1])

	// Virtual state.
	positions := make(map[string]*backtestPosition)
	var closedTrades []storage.TradeRecord
	capital := cfg.Capital
	tradeCounter := int64(0)

	for _, dateStr := range dates {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Load regime.
		regimePath := filepath.Join(aiDir, dateStr, "market_regime.json")
		regimeData, err := os.ReadFile(regimePath)
		if err != nil {
			continue
		}
		var regime strategy.MarketRegimeData
		if err := json.Unmarshal(regimeData, &regime); err != nil {
			continue
		}

		// Load scores.
		scoresPath := filepath.Join(aiDir, dateStr, "stock_scores.json")
		scoresData, err := os.ReadFile(scoresPath)
		if err != nil {
			continue
		}
		var scores []strategy.StockScore
		if err := json.Unmarshal(scoresData, &scores); err != nil {
			continue
		}

		sort.Slice(scores, func(i, j int) bool {
			return scores[i].Rank < scores[j].Rank
		})

		// Build open positions list for risk manager.
		var openPositions []strategy.PositionInfo
		for _, pos := range positions {
			openPositions = append(openPositions, strategy.PositionInfo{
				Symbol:     pos.symbol,
				EntryPrice: pos.entryPrice,
				Quantity:   pos.quantity,
				StopLoss:   pos.stopLoss,
				Target:     pos.target,
				StrategyID: pos.strategyID,
				EntryTime:  pos.entryDate,
			})
		}

		availableCapital := capital
		for _, pos := range positions {
			availableCapital -= pos.entryPrice * float64(pos.quantity)
		}
		if availableCapital < 0 {
			availableCapital = 0
		}

		dailyPnL := risk.DailyPnL{Date: date}

		for _, score := range scores {
			// Load candle data.
			csvPath := filepath.Join(cfg.Paths.MarketDataDir, score.Symbol+".csv")
			candles := market.LoadExistingCSV(csvPath)
			if len(candles) == 0 {
				continue
			}

			// Filter candles up to this date.
			var candlesUpToDate []strategy.Candle
			for _, c := range candles {
				if !c.Date.After(date) {
					candlesUpToDate = append(candlesUpToDate, c)
				}
			}
			if len(candlesUpToDate) == 0 {
				continue
			}

			// Check existing position.
			var currentPos *strategy.PositionInfo
			if pos, exists := positions[score.Symbol]; exists {
				cp := strategy.PositionInfo{
					Symbol:     pos.symbol,
					EntryPrice: pos.entryPrice,
					Quantity:   pos.quantity,
					StopLoss:   pos.stopLoss,
					Target:     pos.target,
					StrategyID: pos.strategyID,
					EntryTime:  pos.entryDate,
				}
				currentPos = &cp
			}

			input := strategy.StrategyInput{
				Date:              date,
				Regime:            regime,
				Score:             score,
				Candles:           candlesUpToDate,
				CurrentPosition:   currentPos,
				OpenPositionCount: len(positions),
				AvailableCapital:  availableCapital,
			}

			for _, strat := range strategies {
				intent := strat.Evaluate(input)

				switch intent.Action {
				case strategy.ActionBuy:
					if _, alreadyHeld := positions[intent.Symbol]; alreadyHeld {
						continue
					}
					result := riskMgr.Validate(intent, openPositions, dailyPnL, availableCapital, sectorMap)
					if !result.Approved {
						continue
					}

					positions[intent.Symbol] = &backtestPosition{
						symbol:     intent.Symbol,
						entryPrice: intent.Price,
						quantity:   intent.Quantity,
						stopLoss:   intent.StopLoss,
						target:     intent.Target,
						strategyID: intent.StrategyID,
						entryDate:  date,
					}
					availableCapital -= intent.Price * float64(intent.Quantity)
					openPositions = append(openPositions, strategy.PositionInfo{
						Symbol:     intent.Symbol,
						EntryPrice: intent.Price,
						Quantity:   intent.Quantity,
						StopLoss:   intent.StopLoss,
						Target:     intent.Target,
						StrategyID: intent.StrategyID,
						EntryTime:  date,
					})
					tradeCounter++

				case strategy.ActionExit:
					pos, exists := positions[intent.Symbol]
					if !exists {
						continue
					}
					exitPrice := intent.Price
					if exitPrice == 0 && len(candlesUpToDate) > 0 {
						exitPrice = candlesUpToDate[len(candlesUpToDate)-1].Close
					}
					pnl := float64(pos.quantity) * (exitPrice - pos.entryPrice)
					exitTime := date
					closedTrades = append(closedTrades, storage.TradeRecord{
						ID:         tradeCounter,
						StrategyID: pos.strategyID,
						Symbol:     pos.symbol,
						Side:       string(strategy.ActionBuy),
						Quantity:   pos.quantity,
						EntryPrice: pos.entryPrice,
						ExitPrice:  exitPrice,
						StopLoss:   pos.stopLoss,
						Target:     pos.target,
						EntryTime:  pos.entryDate,
						ExitTime:   &exitTime,
						ExitReason: intent.Reason,
						PnL:        pnl,
						Status:     "closed",
					})
					delete(positions, intent.Symbol)
				}
			}

			// Check SL/target hits on existing positions using today's candle data.
			if pos, exists := positions[score.Symbol]; exists {
				lastCandle := candlesUpToDate[len(candlesUpToDate)-1]

				// Trailing stop-loss adjustment in backtest.
				if cfg.Risk.TrailingStop.Enabled && cfg.Risk.TrailingStop.TrailPct > 0 {
					highPrice := lastCandle.High
					profitPct := (highPrice - pos.entryPrice) / pos.entryPrice * 100
					if profitPct >= cfg.Risk.TrailingStop.ActivationPct {
						newSL := highPrice * (1 - cfg.Risk.TrailingStop.TrailPct/100)
						if newSL > pos.stopLoss {
							pos.stopLoss = newSL
						}
					}
				}

				var exitPrice float64
				var exitReason string

				if lastCandle.Low <= pos.stopLoss {
					exitPrice = pos.stopLoss
					exitReason = "stop_loss"
				} else if pos.target > 0 && lastCandle.High >= pos.target {
					exitPrice = pos.target
					exitReason = "target"
				}

				if exitReason != "" {
					pnl := float64(pos.quantity) * (exitPrice - pos.entryPrice)
					exitTime := date
					closedTrades = append(closedTrades, storage.TradeRecord{
						ID:         tradeCounter,
						StrategyID: pos.strategyID,
						Symbol:     pos.symbol,
						Side:       string(strategy.ActionBuy),
						Quantity:   pos.quantity,
						EntryPrice: pos.entryPrice,
						ExitPrice:  exitPrice,
						StopLoss:   pos.stopLoss,
						Target:     pos.target,
						EntryTime:  pos.entryDate,
						ExitTime:   &exitTime,
						ExitReason: exitReason,
						PnL:        pnl,
						Status:     "closed",
					})
					delete(positions, score.Symbol)
				}
			}

			// Max hold period check.
			if cfg.Risk.MaxHoldDays > 0 {
				if pos, exists := positions[score.Symbol]; exists {
					holdDays := int(date.Sub(pos.entryDate).Hours() / 24)
					if holdDays >= cfg.Risk.MaxHoldDays {
						exitPrice := candlesUpToDate[len(candlesUpToDate)-1].Close
						pnl := float64(pos.quantity) * (exitPrice - pos.entryPrice)
						exitTime := date
						closedTrades = append(closedTrades, storage.TradeRecord{
							ID:         tradeCounter,
							StrategyID: pos.strategyID,
							Symbol:     pos.symbol,
							Side:       string(strategy.ActionBuy),
							Quantity:   pos.quantity,
							EntryPrice: pos.entryPrice,
							ExitPrice:  exitPrice,
							StopLoss:   pos.stopLoss,
							Target:     pos.target,
							EntryTime:  pos.entryDate,
							ExitTime:   &exitTime,
							ExitReason: "max_hold_period",
							PnL:        pnl,
							Status:     "closed",
						})
						delete(positions, score.Symbol)
					}
				}
			}
		}
	}

	// Close any remaining open positions at last known price.
	for symbol, pos := range positions {
		csvPath := filepath.Join(cfg.Paths.MarketDataDir, symbol+".csv")
		candles := market.LoadExistingCSV(csvPath)
		exitPrice := pos.entryPrice // fallback
		if len(candles) > 0 {
			exitPrice = candles[len(candles)-1].Close
		}
		pnl := float64(pos.quantity) * (exitPrice - pos.entryPrice)
		lastDate := time.Now()
		closedTrades = append(closedTrades, storage.TradeRecord{
			ID:         tradeCounter,
			StrategyID: pos.strategyID,
			Symbol:     pos.symbol,
			Side:       string(strategy.ActionBuy),
			Quantity:   pos.quantity,
			EntryPrice: pos.entryPrice,
			ExitPrice:  exitPrice,
			StopLoss:   pos.stopLoss,
			Target:     pos.target,
			EntryTime:  pos.entryDate,
			ExitTime:   &lastDate,
			ExitReason: "backtest_end",
			PnL:        pnl,
			Status:     "closed",
		})
	}

	// Generate report.
	logger.Printf("[backtest] completed: %d closed trades, %d still open at end",
		len(closedTrades), len(positions))

	report := analytics.Analyze(closedTrades, cfg.Capital)
	fmt.Println(analytics.FormatReport(report))
}
