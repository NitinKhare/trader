#!/usr/bin/env python3
"""
Backtest individual strategies in isolation.

This script helps you backtest each strategy separately to understand
their individual performance and behavior.

Usage:
    # List all available strategies
    python scripts/backtest_strategy.py --list

    # Backtest a single strategy
    python scripts/backtest_strategy.py TrendFollow

    # Backtest multiple strategies
    python scripts/backtest_strategy.py TrendFollow Breakout Momentum

    # Backtest with analysis
    python scripts/backtest_strategy.py Breakout --analyze

    # Compare all strategies
    python scripts/backtest_strategy.py --compare-all
"""

import argparse
import json
import os
import subprocess
import sys
from datetime import datetime
from pathlib import Path


# Mapping of strategy display name to Go constructor
STRATEGIES = {
    "TrendFollow": "strategy.NewTrendFollowStrategy",
    "MeanReversion": "strategy.NewMeanReversionStrategy",
    "Breakout": "strategy.NewBreakoutStrategy",
    "Momentum": "strategy.NewMomentumStrategy",
    "VWAPReversion": "strategy.NewVWAPReversionStrategy",
    "Pullback": "strategy.NewPullbackStrategy",
    "ORB": "strategy.NewORBStrategy",
    "MACDCrossover": "strategy.NewMACDCrossoverStrategy",
    "BollingerSqueeze": "strategy.NewBollingerSqueezeStrategy",
}


def list_strategies():
    """Print available strategies."""
    print("Available Strategies:")
    print("=" * 50)
    for i, strategy in enumerate(STRATEGIES.keys(), 1):
        print(f"  {i:2}. {strategy}")
    print("=" * 50)


def get_database_trades(strategy_name: str) -> dict:
    """Query database for trades from specific strategy."""
    try:
        import sqlite3

        db_path = "algo_trading.db"
        if not os.path.exists(db_path):
            print(f"❌ Database not found: {db_path}")
            return {}

        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()

        # Get trades for this strategy
        cursor.execute("""
            SELECT
                symbol,
                entry_price,
                exit_price,
                quantity,
                entry_time,
                exit_time,
                pnl,
                pnl_pct
            FROM trades
            WHERE strategy_id = ?
            ORDER BY entry_time DESC
            LIMIT 100
        """, (strategy_name,))

        trades = cursor.fetchall()
        conn.close()

        return trades if trades else {}
    except Exception as e:
        print(f"⚠️  Could not query database: {e}")
        return {}


def analyze_trades(strategy_name: str) -> dict:
    """Analyze trades from backtest for a strategy."""
    trades = get_database_trades(strategy_name)

    if not trades:
        return {}

    # Calculate metrics
    total_trades = len(trades)
    winning_trades = sum(1 for trade in trades if trade[6] > 0)  # pnl
    losing_trades = total_trades - winning_trades

    total_pnl = sum(trade[6] for trade in trades)
    avg_pnl = total_pnl / total_trades if total_trades > 0 else 0

    gross_profit = sum(trade[6] for trade in trades if trade[6] > 0)
    gross_loss = abs(sum(trade[6] for trade in trades if trade[6] < 0))

    profit_factor = gross_profit / gross_loss if gross_loss > 0 else 0
    win_rate = (winning_trades / total_trades * 100) if total_trades > 0 else 0

    return {
        "strategy": strategy_name,
        "total_trades": total_trades,
        "winning_trades": winning_trades,
        "losing_trades": losing_trades,
        "win_rate": win_rate,
        "total_pnl": total_pnl,
        "avg_pnl": avg_pnl,
        "gross_profit": gross_profit,
        "gross_loss": gross_loss,
        "profit_factor": profit_factor,
    }


def print_strategy_analysis(analysis: dict):
    """Print formatted analysis for a strategy."""
    if not analysis:
        return

    print(f"\n{'='*60}")
    print(f"  {analysis['strategy']} Strategy Results")
    print(f"{'='*60}")
    print(f"  Total Trades:     {analysis['total_trades']}")
    print(f"  Winning:          {analysis['winning_trades']} ({analysis['win_rate']:.1f}%)")
    print(f"  Losing:           {analysis['losing_trades']}")
    print(f"  Total P&L:        ₹{analysis['total_pnl']:,.2f}")
    print(f"  Avg P&L/Trade:    ₹{analysis['avg_pnl']:.2f}")
    print(f"  Gross Profit:     ₹{analysis['gross_profit']:,.2f}")
    print(f"  Gross Loss:       ₹{analysis['gross_loss']:,.2f}")
    print(f"  Profit Factor:    {analysis['profit_factor']:.2f}")
    print(f"{'='*60}")


def run_backtest(strategies: list = None):
    """Run backtest with optional strategy filter.

    Note: This runs the full backtest. To backtest only specific strategies,
    you'd need to modify the Go code to support a --strategy flag.
    For now, this documents what trades each strategy made.
    """
    print("🔄 Running backtest...")
    print("━" * 60)

    # Run the Go backtest
    result = subprocess.run(
        ["go", "run", "./cmd/engine", "--mode", "backtest"],
        cwd=os.getcwd(),
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"❌ Backtest failed!")
        print(result.stderr)
        return False

    print(result.stdout)
    return True


def main():
    parser = argparse.ArgumentParser(
        description="Backtest individual trading strategies"
    )
    parser.add_argument(
        "strategies",
        nargs="*",
        help="Strategy names to backtest"
    )
    parser.add_argument(
        "--list",
        action="store_true",
        help="List all available strategies"
    )
    parser.add_argument(
        "--compare-all",
        action="store_true",
        help="Compare performance of all strategies"
    )
    parser.add_argument(
        "--analyze",
        action="store_true",
        help="Show detailed analysis after backtest"
    )

    args = parser.parse_args()

    # List strategies
    if args.list:
        list_strategies()
        return

    # Run backtest
    if not args.strategies and not args.compare_all:
        print("❌ No strategies specified")
        print("\nUsage:")
        print("  python scripts/backtest_strategy.py --list              # List strategies")
        print("  python scripts/backtest_strategy.py TrendFollow         # Test one strategy")
        print("  python scripts/backtest_strategy.py --compare-all       # Test all strategies")
        parser.print_help()
        return

    # Validate strategies
    if args.strategies:
        for strat in args.strategies:
            if strat not in STRATEGIES:
                print(f"❌ Unknown strategy: {strat}")
                print("\nAvailable strategies:")
                list_strategies()
                sys.exit(1)

    # Run backtest
    print("📊 Strategy Backtest")
    print("=" * 60)

    run_backtest(args.strategies)

    # Analysis
    if args.analyze or args.compare_all:
        print("\n📈 Strategy Analysis")
        print("=" * 60)

        if args.compare_all:
            strategies_to_analyze = list(STRATEGIES.keys())
        else:
            strategies_to_analyze = args.strategies

        results = []
        for strategy in strategies_to_analyze:
            analysis = analyze_trades(strategy)
            if analysis:
                results.append(analysis)
                print_strategy_analysis(analysis)

        # Summary comparison
        if len(results) > 1:
            print("\n📊 Strategy Comparison")
            print("=" * 60)
            print(f"{'Strategy':<20} {'Trades':<10} {'Win%':<10} {'PnL':<15} {'P-Factor':<10}")
            print("-" * 60)
            for r in sorted(results, key=lambda x: x['total_pnl'], reverse=True):
                print(
                    f"{r['strategy']:<20} "
                    f"{r['total_trades']:<10} "
                    f"{r['win_rate']:>8.1f}% "
                    f"₹{r['total_pnl']:>12,.0f} "
                    f"{r['profit_factor']:>8.2f}"
                )
            print("=" * 60)


if __name__ == "__main__":
    main()
