"""
Market regime detection module.

Determines the overall market condition (BULL / SIDEWAYS / BEAR)
based on NIFTY 50 index data and breadth indicators.

Output is a JSON file consumed by the Go strategy engine.
All logic is deterministic and runs offline.
"""

import os
import json
from datetime import date

import pandas as pd
import numpy as np

from python_ai.features.indicators import (
    compute_sma,
    compute_ema,
    compute_rsi,
    compute_adx,
    compute_atr,
)


def detect_regime(nifty_df: pd.DataFrame) -> dict:
    """
    Detect market regime from NIFTY 50 OHLCV data.

    Args:
        nifty_df: NIFTY 50 daily OHLCV data with columns: date, open, high, low, close, volume.
                  Must have at least 200 rows of history.

    Returns:
        Dict with keys: date, regime, confidence.
        Regime is one of: "BULL", "SIDEWAYS", "BEAR".
        Confidence is 0.0 to 1.0.
    """
    if len(nifty_df) < 200:
        return {
            "date": str(date.today()),
            "regime": "SIDEWAYS",
            "confidence": 0.5,
        }

    df = nifty_df.copy().sort_values("date").reset_index(drop=True)

    # Compute indicators on NIFTY.
    close = df["close"]
    sma_50 = compute_sma(close, 50)
    sma_200 = compute_sma(close, 200)
    ema_21 = compute_ema(close, 21)
    rsi = compute_rsi(close, 14)
    adx = compute_adx(df, 14)

    # Get latest values.
    latest = df.iloc[-1]
    latest_close = latest["close"]
    latest_sma_50 = sma_50.iloc[-1]
    latest_sma_200 = sma_200.iloc[-1]
    latest_ema_21 = ema_21.iloc[-1]
    latest_rsi = rsi.iloc[-1]
    latest_adx = adx.iloc[-1] if pd.notna(adx.iloc[-1]) else 20

    # Scoring system for regime detection.
    bull_score = 0.0
    bear_score = 0.0

    # Factor 1: Price vs long-term MA (weight: 0.25).
    if latest_close > latest_sma_200:
        bull_score += 0.25
    else:
        bear_score += 0.25

    # Factor 2: SMA 50/200 golden/death cross (weight: 0.20).
    if pd.notna(latest_sma_50) and pd.notna(latest_sma_200):
        if latest_sma_50 > latest_sma_200:
            bull_score += 0.20
        else:
            bear_score += 0.20

    # Factor 3: Price vs 21 EMA — short-term trend (weight: 0.15).
    if pd.notna(latest_ema_21):
        if latest_close > latest_ema_21:
            bull_score += 0.15
        else:
            bear_score += 0.15

    # Factor 4: RSI zone (weight: 0.15).
    if pd.notna(latest_rsi):
        if latest_rsi > 55:
            bull_score += 0.15
        elif latest_rsi < 45:
            bear_score += 0.15
        # Otherwise neutral — neither gets points.

    # Factor 5: 20-day price change (weight: 0.15).
    if len(df) >= 20:
        price_change_20d = (latest_close - df.iloc[-20]["close"]) / df.iloc[-20]["close"]
        if price_change_20d > 0.02:
            bull_score += 0.15
        elif price_change_20d < -0.02:
            bear_score += 0.15

    # Factor 6: ADX trend strength (weight: 0.10).
    # Strong ADX amplifies the prevailing direction.
    if latest_adx > 25:
        if bull_score > bear_score:
            bull_score += 0.10
        elif bear_score > bull_score:
            bear_score += 0.10

    # Determine regime.
    total = bull_score + bear_score
    if total == 0:
        return {
            "date": str(latest["date"]),
            "regime": "SIDEWAYS",
            "confidence": 0.5,
        }

    bull_ratio = bull_score / (bull_score + bear_score)

    if bull_ratio >= 0.65:
        regime = "BULL"
        confidence = min(bull_ratio, 0.95)
    elif bull_ratio <= 0.35:
        regime = "BEAR"
        confidence = min(1.0 - bull_ratio, 0.95)
    else:
        regime = "SIDEWAYS"
        confidence = 1.0 - abs(bull_ratio - 0.5) * 2  # More sideways = higher confidence.

    return {
        "date": str(latest["date"]) if hasattr(latest["date"], "strftime") else str(latest["date"]),
        "regime": regime,
        "confidence": round(confidence, 4),
    }


def save_regime(regime_data: dict, output_dir: str, regime_date: date) -> str:
    """
    Save market regime to a JSON file in the date-partitioned output directory.

    Output path: {output_dir}/{YYYY-MM-DD}/market_regime.json
    """
    date_str = regime_date.strftime("%Y-%m-%d")
    date_dir = os.path.join(output_dir, date_str)
    os.makedirs(date_dir, exist_ok=True)

    output_path = os.path.join(date_dir, "market_regime.json")
    with open(output_path, "w") as f:
        json.dump(regime_data, f, indent=2)

    return output_path
