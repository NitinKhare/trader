// Package portfolio - correlation.go provides Pearson correlation analysis
// between stock daily returns. Used to reduce concentration risk by detecting
// highly correlated positions.
package portfolio

import (
	"math"

	"github.com/nitinkhare/algoTradingAgent/internal/strategy"
)

// CorrelationEngine computes and evaluates correlations between stock returns.
type CorrelationEngine struct {
	maxCorrelation float64
	lookbackDays   int
}

// CorrelationMatrix holds the pairwise correlation values between symbols.
type CorrelationMatrix struct {
	Symbols []string
	Values  [][]float64
}

// NewCorrelationEngine creates a CorrelationEngine with the given thresholds.
// Default maxCorrelation is 0.75 and lookbackDays is 30.
func NewCorrelationEngine(maxCorrelation float64, lookbackDays int) *CorrelationEngine {
	if maxCorrelation <= 0 {
		maxCorrelation = 0.75
	}
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	return &CorrelationEngine{
		maxCorrelation: maxCorrelation,
		lookbackDays:   lookbackDays,
	}
}

// ComputeCorrelation computes the Pearson correlation matrix of daily returns
// for all symbols provided. Daily return = (close_i - close_{i-1}) / close_{i-1}.
// Only the last lookbackDays candles are used. Symbols with insufficient data
// are included but will have 0 correlation with others.
func (ce *CorrelationEngine) ComputeCorrelation(candlesBySymbol map[string][]strategy.Candle) *CorrelationMatrix {
	symbols := make([]string, 0, len(candlesBySymbol))
	for sym := range candlesBySymbol {
		symbols = append(symbols, sym)
	}

	n := len(symbols)
	matrix := &CorrelationMatrix{
		Symbols: symbols,
		Values:  make([][]float64, n),
	}
	for i := range matrix.Values {
		matrix.Values[i] = make([]float64, n)
		matrix.Values[i][i] = 1.0 // self-correlation is always 1
	}

	// Compute daily returns for each symbol.
	returns := make(map[string][]float64, n)
	for _, sym := range symbols {
		candles := candlesBySymbol[sym]
		returns[sym] = computeDailyReturns(candles, ce.lookbackDays)
	}

	// Compute pairwise Pearson correlation.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			corr := pearsonCorrelation(returns[symbols[i]], returns[symbols[j]])
			matrix.Values[i][j] = corr
			matrix.Values[j][i] = corr
		}
	}

	return matrix
}

// ShouldReduceAllocation checks if a proposed symbol has high correlation with
// any of the existing symbols. Returns true if any correlation exceeds the threshold,
// along with the most correlated symbol and its correlation value.
func (ce *CorrelationEngine) ShouldReduceAllocation(proposed string, existing []string, matrix *CorrelationMatrix) (reduce bool, correlatedWith string, correlation float64) {
	if matrix == nil || len(existing) == 0 {
		return false, "", 0
	}

	// Find index of proposed symbol in matrix.
	proposedIdx := -1
	for i, sym := range matrix.Symbols {
		if sym == proposed {
			proposedIdx = i
			break
		}
	}
	if proposedIdx < 0 {
		return false, "", 0
	}

	// Check correlation with each existing symbol.
	var maxCorr float64
	var maxCorrSymbol string
	for _, existSym := range existing {
		existIdx := -1
		for i, sym := range matrix.Symbols {
			if sym == existSym {
				existIdx = i
				break
			}
		}
		if existIdx < 0 {
			continue
		}

		corr := math.Abs(matrix.Values[proposedIdx][existIdx])
		if corr > maxCorr {
			maxCorr = corr
			maxCorrSymbol = existSym
		}
	}

	if maxCorr > ce.maxCorrelation {
		return true, maxCorrSymbol, maxCorr
	}
	return false, "", 0
}

// computeDailyReturns computes the simple daily returns from candle close prices.
// Uses the last lookbackDays candles (needs lookbackDays+1 candles for lookbackDays returns).
func computeDailyReturns(candles []strategy.Candle, lookbackDays int) []float64 {
	if len(candles) < 2 {
		return nil
	}

	// Determine how many returns we can compute.
	start := len(candles) - lookbackDays - 1
	if start < 0 {
		start = 0
	}

	returns := make([]float64, 0, lookbackDays)
	for i := start + 1; i < len(candles); i++ {
		prevClose := candles[i-1].Close
		if prevClose == 0 {
			returns = append(returns, 0)
			continue
		}
		ret := (candles[i].Close - prevClose) / prevClose
		returns = append(returns, ret)
	}
	return returns
}

// pearsonCorrelation computes the Pearson correlation coefficient between two slices.
// Uses the overlapping portion (min length). Returns 0 if insufficient data.
func pearsonCorrelation(x, y []float64) float64 {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	if n < 2 {
		return 0
	}

	// Use only the last n elements from each.
	x = x[len(x)-n:]
	y = y[len(y)-n:]

	// Compute means.
	var sumX, sumY float64
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	// Compute covariance and standard deviations.
	var cov, varX, varY float64
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		cov += dx * dy
		varX += dx * dx
		varY += dy * dy
	}

	denom := math.Sqrt(varX * varY)
	if denom == 0 {
		return 0
	}

	return cov / denom
}
