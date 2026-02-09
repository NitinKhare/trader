"""
Stock scoring module.

Converts raw market data and technical features into actionable scores.
Each stock receives scores in [0, 1] for:
  - TrendStrengthScore
  - BreakoutQualityScore
  - VolatilityScore
  - RiskScore
  - LiquidityScore

All scoring is done offline (not during market hours).
Outputs are written to files for Go consumption.
Scoring is deterministic: same input -> same output.
"""

import os
import json
from datetime import date

import pandas as pd
import numpy as np


def score_trend_strength(features: pd.Series) -> float:
    """
    Score trend strength based on moving average alignment and ADX.

    High score = strong uptrend:
      - Price above key MAs
      - MAs in bullish order (9 > 21 > 50 > 200)
      - Strong ADX
    """
    score = 0.0
    weights_total = 0.0

    # Price above SMAs (weight: 0.3).
    ma_checks = [
        ("sma_20", 0.05),
        ("sma_50", 0.10),
        ("sma_200", 0.15),
    ]
    for ma_col, weight in ma_checks:
        weights_total += weight
        if pd.notna(features.get(ma_col)) and features["close"] > features[ma_col]:
            score += weight

    # MA alignment — bullish order (weight: 0.2).
    weights_total += 0.2
    if (
        pd.notna(features.get("ema_9"))
        and pd.notna(features.get("ema_21"))
        and pd.notna(features.get("sma_50"))
    ):
        if features["ema_9"] > features["ema_21"] > features["sma_50"]:
            score += 0.2

    # MACD bullish (weight: 0.15).
    weights_total += 0.15
    if pd.notna(features.get("macd_histogram")) and features["macd_histogram"] > 0:
        score += 0.15

    # RSI in bullish zone 50-70 (weight: 0.15).
    weights_total += 0.15
    if pd.notna(features.get("rsi_14")):
        rsi = features["rsi_14"]
        if 50 <= rsi <= 70:
            score += 0.15
        elif 40 <= rsi < 50:
            score += 0.07

    # ADX strength (weight: 0.2).
    weights_total += 0.2
    if pd.notna(features.get("adx_14")):
        adx = features["adx_14"]
        if adx > 25:
            score += 0.2 * min(adx / 50.0, 1.0)

    return min(score / weights_total, 1.0) if weights_total > 0 else 0.0


def score_breakout_quality(features: pd.Series) -> float:
    """
    Score breakout quality.

    High score = recent breakout with volume confirmation:
      - Price near Bollinger upper band
      - Volume above average
      - MACD crossing up
    """
    score = 0.0

    # Price position relative to Bollinger Bands (weight: 0.4).
    if pd.notna(features.get("bb_upper")) and pd.notna(features.get("bb_lower")):
        bb_range = features["bb_upper"] - features["bb_lower"]
        if bb_range > 0:
            position = (features["close"] - features["bb_lower"]) / bb_range
            if position > 0.8:
                score += 0.4
            elif position > 0.6:
                score += 0.2

    # Volume confirmation (weight: 0.35).
    if pd.notna(features.get("volume_sma_20")) and features["volume_sma_20"] > 0:
        vol_ratio = features["volume"] / features["volume_sma_20"]
        if vol_ratio > 1.5:
            score += 0.35
        elif vol_ratio > 1.2:
            score += 0.2
        elif vol_ratio > 1.0:
            score += 0.1

    # MACD histogram positive and increasing (weight: 0.25).
    if pd.notna(features.get("macd_histogram")):
        if features["macd_histogram"] > 0:
            score += 0.15
        if pd.notna(features.get("macd")) and pd.notna(features.get("macd_signal")):
            if features["macd"] > features["macd_signal"]:
                score += 0.10

    return min(score, 1.0)


def score_volatility(features: pd.Series) -> float:
    """
    Score volatility (0 = very volatile/risky, 1 = stable/favorable).

    For swing trading, moderate volatility is preferred.
    Too low = no opportunity. Too high = too risky.
    """
    if not pd.notna(features.get("atr_14")) or features["close"] == 0:
        return 0.5

    # ATR as percentage of price.
    atr_pct = features["atr_14"] / features["close"]

    # Sweet spot: 1-3% daily range for swing trading.
    if 0.01 <= atr_pct <= 0.03:
        return 0.8 + (0.2 * (1 - abs(atr_pct - 0.02) / 0.01))
    elif atr_pct < 0.01:
        return max(0.3, atr_pct / 0.01)  # Too quiet.
    else:
        return max(0.1, 1.0 - (atr_pct - 0.03) / 0.05)  # Too volatile.


def score_risk(features: pd.Series) -> float:
    """
    Score risk (0 = low risk, 1 = high risk).

    Note: LOWER is better for this score.
    """
    risk = 0.0

    # RSI extremes increase risk.
    if pd.notna(features.get("rsi_14")):
        rsi = features["rsi_14"]
        if rsi > 80:
            risk += 0.3  # Overbought.
        elif rsi < 20:
            risk += 0.2  # Oversold (risky for longs).

    # High ATR relative to price = higher risk.
    if pd.notna(features.get("atr_14")) and features["close"] > 0:
        atr_pct = features["atr_14"] / features["close"]
        if atr_pct > 0.04:
            risk += 0.3
        elif atr_pct > 0.03:
            risk += 0.15

    # Price below 200 SMA = higher risk.
    if pd.notna(features.get("sma_200")):
        if features["close"] < features["sma_200"]:
            risk += 0.2

    # Weak ADX with bearish price action.
    if pd.notna(features.get("adx_14")) and features["adx_14"] < 15:
        risk += 0.1  # No clear trend.

    # MACD bearish.
    if pd.notna(features.get("macd_histogram")) and features["macd_histogram"] < 0:
        risk += 0.1

    return min(risk, 1.0)


def score_liquidity(features: pd.Series) -> float:
    """
    Score liquidity (0 = illiquid, 1 = highly liquid).

    Based on volume relative to historical average.
    """
    if not pd.notna(features.get("volume_sma_20")) or features["volume_sma_20"] <= 0:
        return 0.0

    # Current volume vs 20-day average.
    vol_ratio = features["volume"] / features["volume_sma_20"]

    # Absolute volume threshold (in number of shares).
    abs_volume = features["volume"]

    # Combine relative and absolute liquidity.
    relative_score = min(vol_ratio / 2.0, 1.0)  # Normalize: 2x average = 1.0.
    absolute_score = min(abs_volume / 500_000, 1.0)  # 500k shares = fully liquid.

    return 0.6 * relative_score + 0.4 * absolute_score


def score_stock(features: pd.Series) -> dict:
    """
    Generate all scores for a single stock from its feature row.
    Returns a dict matching the Go StockScore struct.
    """
    return {
        "symbol": features.get("symbol", "UNKNOWN"),
        "trend_strength_score": round(score_trend_strength(features), 4),
        "breakout_quality_score": round(score_breakout_quality(features), 4),
        "volatility_score": round(score_volatility(features), 4),
        "risk_score": round(score_risk(features), 4),
        "liquidity_score": round(score_liquidity(features), 4),
    }


def score_universe(features_df: pd.DataFrame) -> pd.DataFrame:
    """
    Score all stocks in the universe.

    Args:
        features_df: DataFrame with one row per stock, containing all computed features.

    Returns:
        DataFrame with stock scores, sorted by composite rank.
    """
    scores = []
    for _, row in features_df.iterrows():
        scores.append(score_stock(row))

    scores_df = pd.DataFrame(scores)

    # Composite ranking: higher is better.
    # Weight: trend > breakout > liquidity > volatility > (1 - risk).
    scores_df["composite_score"] = (
        0.30 * scores_df["trend_strength_score"]
        + 0.25 * scores_df["breakout_quality_score"]
        + 0.20 * scores_df["liquidity_score"]
        + 0.15 * scores_df["volatility_score"]
        + 0.10 * (1.0 - scores_df["risk_score"])
    )

    scores_df = scores_df.sort_values("composite_score", ascending=False)
    scores_df["rank"] = range(1, len(scores_df) + 1)

    return scores_df


def save_scores(scores_df: pd.DataFrame, output_dir: str, scoring_date: date) -> str:
    """
    Save stock scores to a Parquet file in the date-partitioned output directory.

    Output path: {output_dir}/{YYYY-MM-DD}/stock_scores.parquet
    """
    date_str = scoring_date.strftime("%Y-%m-%d")
    date_dir = os.path.join(output_dir, date_str)
    os.makedirs(date_dir, exist_ok=True)

    output_path = os.path.join(date_dir, "stock_scores.parquet")
    scores_df.to_parquet(output_path, index=False)

    return output_path
