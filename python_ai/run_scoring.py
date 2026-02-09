"""
Main entry point for the nightly AI scoring pipeline.

This script is called by the Go scheduler as part of nightly jobs.
It:
  1. Loads OHLCV data from the database/files
  2. Computes technical features
  3. Scores all stocks in the universe
  4. Detects market regime from NIFTY data
  5. Writes outputs to ai_outputs/{date}/ for Go consumption

Usage:
    python -m python_ai.run_scoring --date 2026-02-08 --output-dir ./ai_outputs

All operations are offline and deterministic.
"""

import argparse
import os
import sys
import json
from datetime import date, datetime

import pandas as pd
import numpy as np

from python_ai.features.indicators import compute_all_features
from python_ai.scoring.stock_scorer import score_universe, save_scores
from python_ai.scoring.market_regime import detect_regime, save_regime


# Fixed seed for reproducibility.
np.random.seed(42)


def load_ohlcv_from_csv(data_dir: str, symbol: str) -> pd.DataFrame:
    """Load OHLCV data for a symbol from CSV files."""
    filepath = os.path.join(data_dir, f"{symbol}.csv")
    if not os.path.exists(filepath):
        return pd.DataFrame()

    df = pd.read_csv(filepath, parse_dates=["date"])
    required_cols = {"date", "open", "high", "low", "close", "volume"}
    if not required_cols.issubset(set(df.columns)):
        print(f"WARNING: {symbol} CSV missing required columns: {required_cols - set(df.columns)}")
        return pd.DataFrame()

    return df.sort_values("date").reset_index(drop=True)


def get_stock_universe(universe_file: str) -> list[str]:
    """Load stock universe from a JSON file."""
    if not os.path.exists(universe_file):
        print(f"WARNING: Universe file not found: {universe_file}")
        return []

    with open(universe_file) as f:
        data = json.load(f)

    return data.get("symbols", [])


def main():
    parser = argparse.ArgumentParser(description="Run nightly AI scoring pipeline")
    parser.add_argument("--date", type=str, default=str(date.today()),
                        help="Scoring date (YYYY-MM-DD)")
    parser.add_argument("--output-dir", type=str, default="./ai_outputs",
                        help="Output directory for AI scores")
    parser.add_argument("--data-dir", type=str, default="./market_data",
                        help="Directory containing OHLCV CSV files")
    parser.add_argument("--universe-file", type=str, default="./config/stock_universe.json",
                        help="Path to stock universe JSON file")
    args = parser.parse_args()

    scoring_date = datetime.strptime(args.date, "%Y-%m-%d").date()
    print(f"=== AI Scoring Pipeline: {scoring_date} ===")

    # Step 1: Load stock universe.
    symbols = get_stock_universe(args.universe_file)
    if not symbols:
        print("ERROR: No symbols in universe. Exiting.")
        sys.exit(1)
    print(f"Universe: {len(symbols)} stocks")

    # Step 2: Load and process NIFTY data for regime detection.
    nifty_df = load_ohlcv_from_csv(args.data_dir, "NIFTY50")
    if nifty_df.empty:
        print("WARNING: No NIFTY data found. Defaulting to SIDEWAYS regime.")
        regime_data = {"date": str(scoring_date), "regime": "SIDEWAYS", "confidence": 0.5}
    else:
        regime_data = detect_regime(nifty_df)
    print(f"Market regime: {regime_data['regime']} (confidence: {regime_data['confidence']})")

    # Step 3: Save regime.
    regime_path = save_regime(regime_data, args.output_dir, scoring_date)
    print(f"Regime saved: {regime_path}")

    # Step 4: Compute features and score each stock.
    all_features = []
    for symbol in symbols:
        df = load_ohlcv_from_csv(args.data_dir, symbol)
        if df.empty or len(df) < 50:
            print(f"  SKIP {symbol}: insufficient data ({len(df)} candles)")
            continue

        features = compute_all_features(df)
        if features.empty:
            continue

        last_row = features.iloc[-1].copy()
        last_row["symbol"] = symbol
        all_features.append(last_row)

    if not all_features:
        print("ERROR: No stocks could be scored. Exiting.")
        sys.exit(1)

    features_df = pd.DataFrame(all_features)
    print(f"Scored {len(features_df)} stocks")

    # Step 5: Score universe and save.
    scores_df = score_universe(features_df)
    scores_path = save_scores(scores_df, args.output_dir, scoring_date)
    print(f"Scores saved: {scores_path}")

    # Print top 10 for visibility.
    print("\n=== Top 10 Stocks ===")
    top10 = scores_df.head(10)
    for _, row in top10.iterrows():
        print(
            f"  {row['rank']:>3}. {row['symbol']:<15} "
            f"trend={row['trend_strength_score']:.2f} "
            f"breakout={row['breakout_quality_score']:.2f} "
            f"liq={row['liquidity_score']:.2f} "
            f"risk={row['risk_score']:.2f} "
            f"composite={row['composite_score']:.3f}"
        )

    print(f"\n=== Scoring complete for {scoring_date} ===")


if __name__ == "__main__":
    main()
