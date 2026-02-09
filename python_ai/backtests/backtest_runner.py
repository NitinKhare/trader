"""
Backtest runner for validating strategies against historical data.

Design constraints:
  - Backtests must be reproducible.
  - Fixed random seeds.
  - Versioned outputs.
  - Same scoring logic as live system.
"""

import os
import json
from datetime import date, timedelta
from dataclasses import dataclass, field, asdict

import pandas as pd
import numpy as np

from python_ai.features.indicators import compute_all_features
from python_ai.scoring.stock_scorer import score_stock
from python_ai.scoring.market_regime import detect_regime


# Fixed seed for reproducibility.
RANDOM_SEED = 42
np.random.seed(RANDOM_SEED)


@dataclass
class BacktestTrade:
    """Represents a single trade in the backtest."""

    symbol: str
    entry_date: str
    entry_price: float
    exit_date: str = ""
    exit_price: float = 0.0
    stop_loss: float = 0.0
    target: float = 0.0
    quantity: int = 0
    pnl: float = 0.0
    exit_reason: str = ""


@dataclass
class BacktestResult:
    """Aggregated backtest results."""

    start_date: str
    end_date: str
    initial_capital: float
    final_capital: float
    total_return_pct: float = 0.0
    total_trades: int = 0
    winning_trades: int = 0
    losing_trades: int = 0
    win_rate: float = 0.0
    max_drawdown_pct: float = 0.0
    avg_trade_pnl: float = 0.0
    sharpe_ratio: float = 0.0
    trades: list = field(default_factory=list)


def run_backtest(
    ohlcv_data: dict[str, pd.DataFrame],
    nifty_data: pd.DataFrame,
    start_date: date,
    end_date: date,
    initial_capital: float = 1_000_000.0,
    max_risk_pct: float = 1.0,
    max_positions: int = 5,
) -> BacktestResult:
    """
    Run a backtest over the given date range.

    Args:
        ohlcv_data: Dict mapping symbol -> OHLCV DataFrame.
        nifty_data: NIFTY 50 OHLCV DataFrame for regime detection.
        start_date: Backtest start date.
        end_date: Backtest end date.
        initial_capital: Starting capital in INR.
        max_risk_pct: Max risk per trade as percentage.
        max_positions: Max concurrent open positions.

    Returns:
        BacktestResult with full trade history and metrics.
    """
    capital = initial_capital
    trades: list[BacktestTrade] = []
    open_positions: list[BacktestTrade] = []
    daily_equity: list[float] = []

    # Generate trading dates.
    current = start_date
    trading_dates = []
    while current <= end_date:
        if current.weekday() < 5:  # Skip weekends.
            trading_dates.append(current)
        current += timedelta(days=1)

    for trading_date in trading_dates:
        # Check exits for open positions.
        closed = []
        for pos in open_positions:
            symbol_data = ohlcv_data.get(pos.symbol)
            if symbol_data is None:
                continue

            day_data = symbol_data[symbol_data["date"] == pd.Timestamp(trading_date)]
            if day_data.empty:
                continue

            day = day_data.iloc[0]

            # Check stop loss.
            if day["low"] <= pos.stop_loss:
                pos.exit_date = str(trading_date)
                pos.exit_price = pos.stop_loss
                pos.pnl = (pos.exit_price - pos.entry_price) * pos.quantity
                pos.exit_reason = "stop_loss"
                capital += pos.exit_price * pos.quantity
                trades.append(pos)
                closed.append(pos)
                continue

            # Check target.
            if day["high"] >= pos.target:
                pos.exit_date = str(trading_date)
                pos.exit_price = pos.target
                pos.pnl = (pos.exit_price - pos.entry_price) * pos.quantity
                pos.exit_reason = "target"
                capital += pos.exit_price * pos.quantity
                trades.append(pos)
                closed.append(pos)
                continue

        for c in closed:
            open_positions.remove(c)

        # Score stocks and look for entries (only if we have capacity).
        if len(open_positions) < max_positions:
            # Detect regime.
            nifty_up_to = nifty_data[nifty_data["date"] <= pd.Timestamp(trading_date)]
            if len(nifty_up_to) >= 200:
                regime = detect_regime(nifty_up_to)

                if regime["regime"] == "BULL":
                    # Score each stock.
                    candidates = []
                    for symbol, df in ohlcv_data.items():
                        history = df[df["date"] <= pd.Timestamp(trading_date)].tail(250)
                        if len(history) < 50:
                            continue

                        features = compute_all_features(history)
                        if features.empty:
                            continue

                        last_row = features.iloc[-1].copy()
                        last_row["symbol"] = symbol
                        scores = score_stock(last_row)

                        # Apply entry filters.
                        if (
                            scores["trend_strength_score"] >= 0.6
                            and scores["breakout_quality_score"] >= 0.5
                            and scores["liquidity_score"] >= 0.4
                            and scores["risk_score"] <= 0.5
                        ):
                            scores["_close"] = last_row["close"]
                            scores["_atr"] = last_row.get("atr_14", last_row["close"] * 0.02)
                            candidates.append(scores)

                    # Sort by composite score, take best.
                    candidates.sort(
                        key=lambda x: (
                            0.30 * x["trend_strength_score"]
                            + 0.25 * x["breakout_quality_score"]
                            + 0.20 * x["liquidity_score"]
                        ),
                        reverse=True,
                    )

                    slots = max_positions - len(open_positions)
                    for cand in candidates[:slots]:
                        entry_price = cand["_close"]
                        atr = cand["_atr"] if pd.notna(cand["_atr"]) else entry_price * 0.02
                        stop_loss = entry_price - 2 * atr
                        risk_per_share = entry_price - stop_loss
                        target = entry_price + 2 * risk_per_share

                        max_risk = capital * (max_risk_pct / 100)
                        quantity = int(max_risk / risk_per_share) if risk_per_share > 0 else 0
                        if quantity <= 0:
                            continue

                        cost = entry_price * quantity
                        if cost > capital:
                            quantity = int(capital / entry_price)
                        if quantity <= 0:
                            continue

                        capital -= entry_price * quantity

                        pos = BacktestTrade(
                            symbol=cand["symbol"],
                            entry_date=str(trading_date),
                            entry_price=entry_price,
                            stop_loss=stop_loss,
                            target=target,
                            quantity=quantity,
                        )
                        open_positions.append(pos)

        # Track daily equity.
        equity = capital
        for pos in open_positions:
            symbol_data = ohlcv_data.get(pos.symbol)
            if symbol_data is not None:
                day_data = symbol_data[symbol_data["date"] == pd.Timestamp(trading_date)]
                if not day_data.empty:
                    equity += day_data.iloc[0]["close"] * pos.quantity
                else:
                    equity += pos.entry_price * pos.quantity
        daily_equity.append(equity)

    # Close any remaining positions at last available price.
    for pos in open_positions:
        symbol_data = ohlcv_data.get(pos.symbol)
        if symbol_data is not None and not symbol_data.empty:
            last_price = symbol_data.iloc[-1]["close"]
        else:
            last_price = pos.entry_price

        pos.exit_date = str(end_date)
        pos.exit_price = last_price
        pos.pnl = (pos.exit_price - pos.entry_price) * pos.quantity
        pos.exit_reason = "time_exit"
        capital += pos.exit_price * pos.quantity
        trades.append(pos)

    # Calculate metrics.
    winning = [t for t in trades if t.pnl > 0]
    losing = [t for t in trades if t.pnl <= 0]

    max_drawdown = 0.0
    if daily_equity:
        peak = daily_equity[0]
        for eq in daily_equity:
            if eq > peak:
                peak = eq
            dd = (peak - eq) / peak
            if dd > max_drawdown:
                max_drawdown = dd

    total_return = ((capital - initial_capital) / initial_capital) * 100

    result = BacktestResult(
        start_date=str(start_date),
        end_date=str(end_date),
        initial_capital=initial_capital,
        final_capital=round(capital, 2),
        total_return_pct=round(total_return, 2),
        total_trades=len(trades),
        winning_trades=len(winning),
        losing_trades=len(losing),
        win_rate=round(len(winning) / len(trades) * 100, 2) if trades else 0.0,
        max_drawdown_pct=round(max_drawdown * 100, 2),
        avg_trade_pnl=round(sum(t.pnl for t in trades) / len(trades), 2) if trades else 0.0,
        trades=[asdict(t) for t in trades],
    )

    return result


def save_backtest_result(result: BacktestResult, output_dir: str) -> str:
    """Save backtest results to a JSON file."""
    os.makedirs(output_dir, exist_ok=True)
    filename = f"backtest_{result.start_date}_to_{result.end_date}.json"
    output_path = os.path.join(output_dir, filename)

    with open(output_path, "w") as f:
        json.dump(asdict(result), f, indent=2, default=str)

    return output_path
