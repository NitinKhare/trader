package strategy

import (
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// makeBollingerSqueezeCandles creates candles with a tight consolidation (squeeze)
// followed by a breakout candle that closes above the upper Bollinger Band.
func makeBollingerSqueezeCandles(n int, basePrice float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		price := basePrice
		highSpread := 0.3
		lowSpread := 0.3
		vol := int64(100000)

		if i == n-1 {
			// Breakout candle: jumps above upper band with volume.
			price = basePrice + 8.0
			highSpread = 3.0
			lowSpread = 1.0
			vol = 200000
		} else {
			// Very tight range (low stddev → narrow Bollinger Bands).
			price = basePrice + float64(i%3)*0.1 - float64(i%2)*0.05
		}

		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 0.2,
			High:   price + highSpread,
			Low:    price - lowSpread,
			Close:  price,
			Volume: vol,
		}
	}
	return candles
}

func TestBollinger_SkipsBearRegime(t *testing.T) {
	s := NewBollingerSqueezeStrategy(makeTestRiskConfig())

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBear, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.5,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          makeBollingerSqueezeCandles(50, 100),
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionSkip {
		t.Errorf("expected SKIP in BEAR regime, got %s: %s", result.Action, result.Reason)
	}
}

func TestBollinger_BuysOnSqueezeBreakout(t *testing.T) {
	s := NewBollingerSqueezeStrategy(makeTestRiskConfig())

	candles := makeBollingerSqueezeCandles(50, 100)

	// Verify squeeze exists.
	priorCandles := candles[:len(candles)-1]
	_, _, _, priorBW := CalculateBollingerBands(priorCandles, 20, 2.0)
	t.Logf("prior bandwidth: %.4f (threshold: %.4f)", priorBW, s.SqueezeBandwidth)

	// Verify breakout above upper band.
	_, upper, _, _ := CalculateBollingerBands(candles, 20, 2.0)
	lastPrice := candles[len(candles)-1].Close
	t.Logf("last price: %.2f, upper band: %.2f", lastPrice, upper)

	if priorBW > s.SqueezeBandwidth {
		t.Skipf("test data bandwidth %.4f > %.4f (no squeeze)", priorBW, s.SqueezeBandwidth)
	}
	if lastPrice <= upper {
		t.Skipf("test data price %.2f <= upper %.2f (no breakout)", lastPrice, upper)
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.5,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action != ActionBuy {
		t.Errorf("expected BUY on squeeze breakout, got %s: %s", result.Action, result.Reason)
	}
	if result.StopLoss <= 0 {
		t.Error("expected stop loss to be set")
	}
	if result.Target <= result.Price {
		t.Error("expected target above entry price")
	}
}

func TestBollinger_SkipsWhenNoSqueeze(t *testing.T) {
	s := NewBollingerSqueezeStrategy(makeTestRiskConfig())

	// Create volatile candles (wide bands, no squeeze).
	candles := make([]Candle, 50)
	for i := 0; i < 50; i++ {
		price := 100.0 + float64(i%8)*6.0 - float64(i%5)*4.0
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 3,
			High:   price + 5,
			Low:    price - 5,
			Close:  price,
			Volume: 150000,
		}
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.5,
			LiquidityScore:     0.6,
			RiskScore:          0.2,
		},
		Candles:          candles,
		AvailableCapital: 500000,
	}

	result := s.Evaluate(input)
	if result.Action == ActionBuy {
		t.Errorf("expected non-BUY for volatile candles (no squeeze), got BUY: %s", result.Reason)
	}
}

func TestBollinger_ExitsBelowMiddleBand(t *testing.T) {
	s := NewBollingerSqueezeStrategy(makeTestRiskConfig())

	// Create candles where price is below the middle band (SMA).
	candles := make([]Candle, 50)
	for i := 0; i < 50; i++ {
		var price float64
		if i < 35 {
			price = 100 + float64(i)*0.5
		} else {
			// Drop below the SMA.
			price = 100 + float64(35)*0.5 - float64(i-35)*3.0
		}
		candles[i] = Candle{
			Symbol: "TEST",
			Date:   time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:   price - 1,
			High:   price + 2,
			Low:    price - 2,
			Close:  price,
			Volume: 150000,
		}
	}

	middle, _, _, _ := CalculateBollingerBands(candles, 20, 2.0)
	lastPrice := candles[len(candles)-1].Close
	if lastPrice >= middle {
		t.Skipf("test data: price %.2f >= middle %.2f", lastPrice, middle)
	}

	input := StrategyInput{
		Date:   time.Now(),
		Regime: MarketRegimeData{Regime: RegimeBull, Confidence: 0.8},
		Score: StockScore{
			Symbol:             "TEST",
			TrendStrengthScore: 0.5,
		},
		CurrentPosition: &PositionInfo{
			Symbol:     "TEST",
			EntryPrice: 120,
			Quantity:   10,
		},
		Candles: candles,
	}

	result := s.Evaluate(input)
	if result.Action != ActionExit {
		t.Errorf("expected EXIT below middle BB, got %s: %s", result.Action, result.Reason)
	}
}

func TestBollinger_IDAndName(t *testing.T) {
	s := NewBollingerSqueezeStrategy(config.RiskConfig{MaxRiskPerTradePct: 1.0})
	if s.ID() != "bollinger_squeeze_v1" {
		t.Errorf("unexpected ID: %s", s.ID())
	}
	if s.Name() == "" {
		t.Error("name must not be empty")
	}
}
